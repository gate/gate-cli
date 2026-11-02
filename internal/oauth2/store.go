package oauth2

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/gate/gate-cli/internal/config"
)

// TokenData holds persisted OAuth2 token data per endpoint.
//
// DiscoveryURL binds the cached ClientID to the oauth2.discovery_url that was
// active at DCR time. When the user switches discovery_url (e.g. between
// staging and prod), the old client registration is invalid on the new
// authorization server, so login must detect the mismatch and re-register.
type TokenData struct {
	// Resource绑定获批目标；缺失旧值在发送凭据前拒绝。
	Resource               string    `yaml:"resource,omitempty"`
	ClientID               string    `yaml:"client_id"`
	DiscoveryURL           string    `yaml:"discovery_url,omitempty"`
	AccessToken            string    `yaml:"access_token"`
	RefreshToken           string    `yaml:"refresh_token"`
	TokenType              string    `yaml:"token_type,omitempty"`
	ExpiresAt              time.Time `yaml:"expires_at"`
	Scopes                 string    `yaml:"scopes"`
	RegisteredRedirectURIs []string  `yaml:"registered_redirect_uris,omitempty"`
}

// String implements fmt.Stringer. Both tokens are redacted so that accidental
// %s / %v / log output cannot leak credential material.
func (t *TokenData) String() string {
	if t == nil {
		return "<nil TokenData>"
	}
	return fmt.Sprintf("TokenData{client_id=%s access=%s refresh=%s expires=%s scopes=%s}",
		t.ClientID,
		RedactToken(t.AccessToken),
		RedactToken(t.RefreshToken),
		t.ExpiresAt.Format(time.RFC3339),
		t.Scopes,
	)
}

// GoString implements fmt.GoStringer for %#v (uses the same redaction).
func (t *TokenData) GoString() string { return t.String() }

// IsValid returns true if access token exists and is not expired.
func (t *TokenData) IsValid() bool {
	return t.AccessToken != "" && time.Now().Before(t.ExpiresAt)
}

// HasRefreshToken returns true if a refresh token is available.
func (t *TokenData) HasRefreshToken() bool {
	return t.RefreshToken != ""
}

// legacyTokenFile is the YAML wrapper for the deprecated ~/.gate-cli/tokens.yaml
// layout. Only read during one-time migration; no longer written.
type legacyTokenFile struct {
	OAuth2 *TokenData `yaml:"oauth2"`
}

var (
	cachedHomeOnce sync.Once
	cachedHome     string
	cachedHomeErr  error
)

// resolveHome returns os.UserHomeDir() result, cached after first call to avoid repeated syscalls.
func resolveHome() (string, error) {
	cachedHomeOnce.Do(func() {
		cachedHome, cachedHomeErr = os.UserHomeDir()
		if cachedHomeErr == nil && cachedHome == "" {
			cachedHomeErr = fmt.Errorf("home directory is empty (set HOME environment variable)")
		}
	})
	return cachedHome, cachedHomeErr
}

// resetHomeCache resets the cached home directory. Only for use in tests.
func resetHomeCache() {
	cachedHomeOnce = sync.Once{}
	cachedHome = ""
	cachedHomeErr = nil
}

// ResetHomeCacheForTest is the exported version of resetHomeCache for use by external test packages.
func ResetHomeCacheForTest() { resetHomeCache() }

// tokenDir returns ~/.gate-cli/, validating that HOME is resolvable.
func tokenDir() (string, error) {
	home, err := resolveHome()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".gate-cli"), nil
}

// TokenPath returns ~/.gate-cli/tokens.yaml — the **legacy** token storage path.
// After the tokens-in-config.yaml migration this file is no longer read or
// written by the CLI; the function is retained so `doctor` / `migrate` can
// still locate a leftover legacy file for cleanup.
func TokenPath() (string, error) {
	dir, err := tokenDir()
	if err != nil {
		return "", fmt.Errorf("token path: %w", err)
	}
	return filepath.Join(dir, "tokens.yaml"), nil
}

// ConfigYAMLPath returns ~/.gate-cli/config.yaml — the **active** token store
// after the migration. Tokens now live under oauth2.tokens[] inside this file.
func ConfigYAMLPath() (string, error) {
	dir, err := tokenDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// CheckTokenPermissions verifies that the active token store has 0600
// permissions. After the migration the active store is config.yaml; we also
// warn if a legacy tokens.yaml exists with insecure perms.
func CheckTokenPermissions() error {
	if runtime.GOOS == "windows" {
		return nil
	}
	// Active store: config.yaml (contains access_token/refresh_token after migration).
	cfgPath, err := ConfigYAMLPath()
	if err != nil {
		return err
	}
	if err := checkPerm0600(cfgPath, "config.yaml"); err != nil {
		return err
	}
	// Legacy store: tokens.yaml (may still exist until migration runs).
	legacyPath, err := TokenPath()
	if err != nil {
		return err
	}
	return checkPerm0600(legacyPath, "tokens.yaml")
}

func checkPerm0600(path, label string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat %s: %w", label, err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		return fmt.Errorf("%s has insecure permissions %04o (expected 0600)\n  Fix: chmod 600 %s", label, perm, path)
	}
	return nil
}

// LoadToken reads the OAuth2 token for the currently configured discovery_url.
//
// Routing:
//   - 正式环境 (ShortDiscoveryURL == https://www.gate.com)：先读 oauth2.token
//     单例；若为空再回退到 oauth2.tokens[] 里匹配 https://www.gate.com 的条目。
//   - 非正式环境：只读 oauth2.tokens[]，按短形式 discovery_url 匹配。
//
// Returns nil (no error) if no matching entry exists.
//
// Hot-path note : this function is called by BearerTransport on every
// OAuth2-authenticated request. config.yaml is therefore read + yaml.Unmarshal
// ed at most ONCE per call — the derived discovery URL is taken from the
// same *Config rather than re-loaded.
func LoadToken() (*TokenData, error) {
	if err := CheckTokenPermissions(); err != nil {
		return nil, err
	}

	cfg, err := config.Load(config.Options{Profile: "default"})
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if cfg == nil {
		return nil, nil
	}
	target := resolveDiscoveryURL()

	if isProductionDiscovery(target) {
		// 1. 先找 oauth2.token
		if cfg.OAuth2.Token != nil && cfg.OAuth2.Token.ClientID != "" {
			entry := *cfg.OAuth2.Token
			// 单例不带 discovery_url，补上当前生产 URL 以便内存比较语义一致。
			entry.DiscoveryURL = target
			return tokenDataFromEntry(entry)
		}
		// 2. fallback：tokens[] 里找生产条目
	}

	// 非生产 or 生产 fallback：在 tokens[] 里按短形式匹配。
	targetShort := ShortDiscoveryURL(target)
	for _, entry := range cfg.OAuth2.Tokens {
		if ShortDiscoveryURL(entry.DiscoveryURL) == targetShort {
			td, err := tokenDataFromEntry(entry)
			if err != nil {
				return nil, err
			}
			return td, nil
		}
	}
	return nil, nil
}

// SaveToken upserts t into config.yaml under an exclusive config.yaml.lock
// flock so concurrent gate-cli processes cannot interleave read-modify-write
// and clobber each other's rename(). Callers that are already holding the
// lock (e.g. refreshInternal) must use saveTokenLocked directly.
//
// Routing:
//   - 正式环境：写入 oauth2.token 单例（不含 discovery_url）；同时从 tokens[]
//     里移除任何匹配生产 URL 的存量条目，避免双源。
//   - 非正式环境：按短形式 discovery_url upsert 到 oauth2.tokens[]。
//
// DiscoveryURL 入参允许前缀/完整任一形式；内部自动归一。
func SaveToken(t *TokenData) error {
	ctx, cancel := context.WithTimeout(context.Background(), LockTimeout)
	defer cancel()
	return WithConfigLock(ctx, func() error {
		return saveTokenLocked(t)
	})
}

// saveTokenLocked is the unlocked core of SaveToken. Caller MUST hold
// config.yaml.lock via WithConfigLock; use SaveToken instead when in doubt.
func saveTokenLocked(t *TokenData) error {
	if t == nil {
		return fmt.Errorf("nil token")
	}
	key := NormalizeDiscoveryURL(t.DiscoveryURL)
	if key == "" {
		key = resolveDiscoveryURL()
	}
	entry := tokenEntryFromData(t, key)

	cfgPath, err := ConfigYAMLPath()
	if err != nil {
		return err
	}
	err = config.UpdateFile(cfgPath, func(root *yaml.Node) error {
		if isProductionDiscovery(key) {
			if err := upsertProdToken(root, entry); err != nil {
				return err
			}
			// tokens[] 里若有生产条目，顺手清理，保持单一来源。
			return removeProdFromTokensArray(root)
		}
		return upsertTokenEntry(root, entry)
	}, 0600)
	if err != nil {
		return fmt.Errorf("write tokens into config.yaml: %w", err)
	}
	return nil
}

// RemoveToken clears credential fields for the current discovery_url entry
// but preserves client_id and registered_redirect_uris so the next login can
// skip dynamic client registration.
func RemoveToken() error {
	existing, err := LoadToken()
	if err != nil {
		return err
	}
	if existing == nil {
		return nil
	}
	cleared := &TokenData{
		Resource:               existing.Resource,
		ClientID:               existing.ClientID,
		DiscoveryURL:           existing.DiscoveryURL,
		RegisteredRedirectURIs: existing.RegisteredRedirectURIs,
	}
	return SaveToken(cleared)
}

// --- Legacy tokens.yaml migration ---------------------------------------

// MigrateLegacyTokenFile detects a leftover ~/.gate-cli/tokens.yaml and moves
// its contents into config.yaml's oauth2.tokens[]. Idempotent: if the legacy
// file is absent or the config already contains an entry for the same
// discovery_url, the function is a no-op. On success the legacy file is
// renamed to tokens.yaml.migrated-<RFC3339> and a notice is printed on stderr.
//
// 登录入口在领取授权操作权之前调用；旧凭据仍缺 resource，必须重新授权。
func MigrateLegacyTokenFile(stderr io.Writer) error {
	ctx, cancel := context.WithTimeout(context.Background(), LockTimeout)
	defer cancel()
	// 去重、操作权校验、写入和归档共享同一短锁；不允许旧文件覆盖授权中的状态。
	return WithConfigLock(ctx, func() error { return migrateLegacyTokenFileLocked(stderr) })
}

func migrateLegacyTokenFileLocked(stderr io.Writer) error {
	legacyPath, err := TokenPath()
	if err != nil {
		return err
	}
	info, err := os.Lstat(legacyPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("stat legacy tokens.yaml: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() > 1<<20 {
		return fmt.Errorf("unsafe or oversized legacy tokens.yaml")
	}
	if err := CheckTokenPermissions(); err != nil {
		return err
	}
	record, err := readOperation()
	if err != nil {
		return err
	}
	if record != nil {
		return fmt.Errorf("authentication operation exists; finish or recover it before legacy import")
	}
	file, err := os.Open(legacyPath)
	if err != nil {
		return err
	}
	data, readErr := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	closeErr := file.Close()
	if readErr != nil {
		return fmt.Errorf("read legacy tokens.yaml: %w", readErr)
	}
	if closeErr != nil {
		return closeErr
	}
	if len(data) > 1<<20 {
		return fmt.Errorf("oversized legacy tokens.yaml")
	}
	if len(data) == 0 {
		return nil
	}
	var f legacyTokenFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("parse legacy tokens.yaml: %w", err)
	}
	if f.OAuth2 == nil || f.OAuth2.ClientID == "" {
		// File exists but is empty / post-logout placeholder — nothing to migrate.
		return nil
	}

	// Ensure the token entry has a discovery_url: legacy files written before
	// the DiscoveryURL field was introduced may lack it; default to current.
	// 同时归一化：legacy 文件里的值如果是短形式也要补齐为完整 URL，避免后续匹配失败。
	if v := NormalizeDiscoveryURL(f.OAuth2.DiscoveryURL); v != "" {
		f.OAuth2.DiscoveryURL = v
	} else {
		f.OAuth2.DiscoveryURL = resolveDiscoveryURL()
	}

	// If config.yaml already contains an entry for this discovery_url, leave
	// it untouched — the user may have re-logged in after the legacy file
	// became stale. 匹配时两侧都走短形式，兼容存量长/短混存。
	//
	// Review: production tokens live under oauth2.token (singleton), not
	// oauth2.tokens[]; we must check BOTH locations or the legacy file gets
	// re-written into the array and creates a dual source of truth.
	//
	// review 残留修复: config.Load 失败时不能静默降级到直接 SaveToken，否则
	// 去重检查会被整体跳过。文件不存在是合法场景（首次迁移）→ cfg == nil 且
	// cfgErr == nil，继续走下面的 SaveToken；文件存在但损坏/权限错误 →
	// cfgErr != nil，必须向上报错，由用户/调用方决定是否修复后再试。
	cfg, cfgErr := config.Load(config.Options{Profile: "default"})
	if cfgErr != nil {
		return fmt.Errorf("legacy token migration: cannot load config.yaml for dedup check: %w", cfgErr)
	}
	if cfg != nil {
		legacyShort := ShortDiscoveryURL(f.OAuth2.DiscoveryURL)
		if isProductionDiscovery(legacyShort) && cfg.OAuth2.Token != nil && cfg.OAuth2.Token.ClientID != "" {
			return archiveLegacyFile(legacyPath, stderr, "production token already present in config.yaml oauth2.token")
		}
		for _, e := range cfg.OAuth2.Tokens {
			if ShortDiscoveryURL(e.DiscoveryURL) == legacyShort {
				return archiveLegacyFile(legacyPath, stderr, "already present in config.yaml")
			}
		}
	}

	if err := saveTokenLocked(f.OAuth2); err != nil {
		return fmt.Errorf("migrate tokens.yaml → config.yaml: %w", err)
	}
	return archiveLegacyFile(legacyPath, stderr, "migrated into config.yaml oauth2.tokens[]")
}

func archiveLegacyFile(legacyPath string, stderr io.Writer, reason string) error {
	stamp := time.Now().UTC().Format("20060102T150405Z")
	target := legacyPath + ".migrated-" + stamp
	if err := os.Rename(legacyPath, target); err != nil {
		return fmt.Errorf("archive legacy tokens.yaml: %w", err)
	}
	if stderr != nil {
		log.New(stderr, "", 0).Printf(
			"Notice: %s %s; archived as %s\n",
			filepath.Base(legacyPath), reason, filepath.Base(target),
		)
	}
	return nil
}

// --- YAML Node upsert ---------------------------------------------------

// upsertProdToken 写入正式环境 token 到 root.oauth2.token（单例 mapping）。
// 入参 e 的 DiscoveryURL 会被置空以避免序列化。保留原节点注释（EncodeInto）。
func upsertProdToken(root *yaml.Node, e config.TokenEntry) error {
	e.DiscoveryURL = "" // 生产单例不落盘 discovery_url
	oauth2Node, err := config.EnsureMapChild(root, "oauth2")
	if err != nil {
		return err
	}
	if _, tokenNode := config.FindChild(oauth2Node, "token"); tokenNode != nil {
		return config.EncodeInto(tokenNode, e)
	}
	// 首次写入：新建 mapping 节点挂到 oauth2 下。
	kn := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "token"}
	vn := &yaml.Node{}
	if err := vn.Encode(e); err != nil {
		return err
	}
	oauth2Node.Content = append(oauth2Node.Content, kn, vn)
	return nil
}

// removeProdFromTokensArray 删除 root.oauth2.tokens[] 里短形式 URL 匹配生产的条目。
// 若 tokens 为空数组则连同 tokens key 一并移除，保持 YAML 干净。
func removeProdFromTokensArray(root *yaml.Node) error {
	oauth2Node, err := config.EnsureMapChild(root, "oauth2")
	if err != nil {
		return err
	}
	_, seq := config.FindChild(oauth2Node, "tokens")
	if seq == nil || seq.Kind != yaml.SequenceNode {
		return nil
	}
	prodShort := ShortDiscoveryURL(DefaultDiscoveryURL)
	filtered := seq.Content[:0]
	for _, item := range seq.Content {
		if item.Kind != yaml.MappingNode {
			filtered = append(filtered, item)
			continue
		}
		_, v := config.FindChild(item, "discovery_url")
		if v != nil && v.Kind == yaml.ScalarNode && ShortDiscoveryURL(v.Value) == prodShort {
			continue // skip production entry
		}
		filtered = append(filtered, item)
	}
	seq.Content = filtered
	if len(seq.Content) == 0 {
		config.DeleteKey(oauth2Node, "tokens")
	}
	return nil
}

// upsertTokenEntry replaces or appends an entry in root.oauth2.tokens[] whose
// discovery_url matches e.DiscoveryURL. Preserves siblings and comments.
func upsertTokenEntry(root *yaml.Node, e config.TokenEntry) error {
	oauth2Node, err := config.EnsureMapChild(root, "oauth2")
	if err != nil {
		return err
	}
	seq, err := config.EnsureSeqChild(oauth2Node, "tokens")
	if err != nil {
		return err
	}
	// Locate existing entry by discovery_url（两侧短形式对齐，兼容存量完整 URL 的条目）。
	eShort := ShortDiscoveryURL(e.DiscoveryURL)
	for _, item := range seq.Content {
		if item.Kind != yaml.MappingNode {
			continue
		}
		_, v := config.FindChild(item, "discovery_url")
		if v != nil && v.Kind == yaml.ScalarNode && ShortDiscoveryURL(v.Value) == eShort {
			return config.EncodeInto(item, e)
		}
	}
	// Not found → append.
	var fresh yaml.Node
	if err := fresh.Encode(e); err != nil {
		return err
	}
	seq.Content = append(seq.Content, &fresh)
	return nil
}

// --- Conversions --------------------------------------------------------

// tokenEntryFromData 把内存 TokenData 转成 YAML 落盘条目。落盘使用短形式
// discovery_url（与 oauth2.discovery_url 用户输入保持一致）。
func tokenEntryFromData(t *TokenData, discoveryURL string) config.TokenEntry {
	var expires string
	if !t.ExpiresAt.IsZero() {
		expires = t.ExpiresAt.Format(time.RFC3339Nano)
	}
	return config.TokenEntry{
		Resource:               t.Resource,
		DiscoveryURL:           ShortDiscoveryURL(discoveryURL),
		ClientID:               t.ClientID,
		AccessToken:            t.AccessToken,
		RefreshToken:           t.RefreshToken,
		TokenType:              t.TokenType,
		ExpiresAt:              expires,
		Scopes:                 t.Scopes,
		RegisteredRedirectURIs: append([]string(nil), t.RegisteredRedirectURIs...),
	}
}

// tokenDataFromEntry 把 YAML 条目转成内存 TokenData；内存里 DiscoveryURL 统一
// 展开为完整 RFC 8414 URL，方便下游（如 login.go、refresh.go）直接比较。
func tokenDataFromEntry(e config.TokenEntry) (*TokenData, error) {
	var expires time.Time
	if s := strings.TrimSpace(e.ExpiresAt); s != "" {
		t, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			// Fall back to RFC3339 (no nanoseconds) — legacy values may omit them.
			t, err = time.Parse(time.RFC3339, s)
			if err != nil {
				return nil, fmt.Errorf("parse expires_at %q: %w", s, err)
			}
		}
		expires = t
	}
	return &TokenData{
		Resource:               e.Resource,
		ClientID:               e.ClientID,
		DiscoveryURL:           NormalizeDiscoveryURL(e.DiscoveryURL),
		AccessToken:            e.AccessToken,
		RefreshToken:           e.RefreshToken,
		TokenType:              e.TokenType,
		ExpiresAt:              expires,
		Scopes:                 e.Scopes,
		RegisteredRedirectURIs: append([]string(nil), e.RegisteredRedirectURIs...),
	}, nil
}
