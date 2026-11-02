package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const defaultBaseURL = "https://api.gateio.ws"
const defaultSettle = "usdt"

// DefaultIntelInfoMCPURL and DefaultIntelNewsMCPURL are built-in MCP HTTP bases (public QC),
// analogous to defaultBaseURL for REST; override with GATE_INTEL_*_MCP_URL or config intel:.
const (
	DefaultIntelInfoMCPURL = "https://api.gatemcp.ai/mcp/info"
	DefaultIntelNewsMCPURL = "https://api.gatemcp.ai/mcp/news"
)

// Options controls how config is loaded.
type Options struct {
	ConfigFile    string
	Profile       string
	FlagAPIKey    string
	FlagAPISecret string
}

// Config holds resolved configuration for a single API session.
type Config struct {
	APIKey        string
	APISecret     string
	BaseURL       string
	DefaultSettle string
	Debug         bool
	// Intel holds optional ~/.gate-cli/config.yaml "intel" defaults for MCP (info/news).
	// Non-empty environment variables override these values; see toolconfig.Resolve.
	Intel IntelFile
	// OAuth2复用既有PKCE配置与凭据布局，resource由CLI自动绑定。
	OAuth2 OAuth2File
}

// IntelFile is the optional "intel" section in config.yaml (MCP base URLs, auth, headers, timeout).
type IntelFile struct {
	NewsMCPURL      string            `yaml:"news_mcp_url,omitempty"`
	InfoMCPURL      string            `yaml:"info_mcp_url,omitempty"`
	BearerToken     string            `yaml:"bearer_token,omitempty"`
	NewsBearerToken string            `yaml:"news_bearer_token,omitempty"`
	InfoBearerToken string            `yaml:"info_bearer_token,omitempty"`
	ExtraHeaders    map[string]string `yaml:"extra_headers,omitempty"`
	HTTPTimeout     string            `yaml:"http_timeout,omitempty"`
}

// OAuth2File 只保存自动生成的凭据。发现入口及手动回调为固定常量，
// 不提供用户覆盖项；tokens[] 的 discovery_url 仅是历史凭据元数据。
type OAuth2File struct {
	// Token 是**正式环境专属**的单例 token。当凭据 issuer 为
	// https://www.gate.com 时，SaveToken 写入此节点，不序列化 discovery_url；
	// LoadToken 优先读取此节点，为空再回退到 Tokens[] 里匹配 https://www.gate.com 的条目。
	Token *TokenEntry `yaml:"token,omitempty"`
	// Tokens 存储**非正式环境**（测试/预发等）的 token 数组，用短形式 discovery_url
	// 作主键。生产环境 token 不写入此处；但存量数据在此处也可被读兼容。
	Tokens []TokenEntry `yaml:"tokens,omitempty"`
}

// TokenEntry 是 OAuth2File.Tokens 的单条记录，也被 OAuth2File.Token 复用。
// DiscoveryURL 使用 omitempty —— 在生产环境的单例 oauth2.token 节点下为空，
// 在 tokens[] 数组下为短形式 URL 主键。
type TokenEntry struct {
	// Resource为空表示旧凭据，必须重新授权，不从当前目的地推断。
	Resource               string   `yaml:"resource,omitempty"`
	DiscoveryURL           string   `yaml:"discovery_url,omitempty"`
	ClientID               string   `yaml:"client_id"`
	AccessToken            string   `yaml:"access_token,omitempty"`
	RefreshToken           string   `yaml:"refresh_token,omitempty"`
	TokenType              string   `yaml:"token_type,omitempty"`
	ExpiresAt              string   `yaml:"expires_at,omitempty"`
	Scopes                 string   `yaml:"scopes,omitempty"`
	RegisteredRedirectURIs []string `yaml:"registered_redirect_uris,omitempty"`
}

type fileConfig struct {
	OAuth2         OAuth2File         `yaml:"oauth2,omitempty"`
	DefaultProfile string             `yaml:"default_profile"`
	DefaultSettle  string             `yaml:"default_settle"`
	Intel          IntelFile          `yaml:"intel,omitempty"`
	Profiles       map[string]profile `yaml:"profiles"`
}

type profile struct {
	APIKey    string `yaml:"api_key"`
	APISecret string `yaml:"api_secret"`
	BaseURL   string `yaml:"base_url"`
}

// DefaultConfigPath returns ~/.gate-cli/config.yaml
func DefaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".gate-cli", "config.yaml")
}

// Load resolves config with priority: CLI flag > env var > config file.
func Load(opts Options) (*Config, error) {
	cfg := &Config{
		BaseURL:       defaultBaseURL,
		DefaultSettle: defaultSettle,
	}

	cfgPath := opts.ConfigFile
	if cfgPath == "" {
		cfgPath = DefaultConfigPath()
	}

	if data, err := os.ReadFile(cfgPath); err == nil {
		var fc fileConfig
		if err := yaml.Unmarshal(data, &fc); err != nil {
			return nil, fmt.Errorf("invalid config file: %w", err)
		}
		if fc.DefaultSettle != "" {
			cfg.DefaultSettle = fc.DefaultSettle
		}
		profileName := opts.Profile
		if profileName == "" || profileName == "default" {
			if fc.DefaultProfile != "" {
				profileName = fc.DefaultProfile
			}
		}
		if p, ok := fc.Profiles[profileName]; ok {
			cfg.APIKey = p.APIKey
			cfg.APISecret = p.APISecret
			if p.BaseURL != "" {
				cfg.BaseURL = p.BaseURL
			}
		}
		cfg.Intel = fc.Intel
		cfg.OAuth2 = fc.OAuth2
	}

	// Env vars override file
	if v := os.Getenv("GATE_API_KEY"); v != "" {
		cfg.APIKey = v
	}
	if v := os.Getenv("GATE_API_SECRET"); v != "" {
		cfg.APISecret = v
	}
	if v := os.Getenv("GATE_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}

	// CLI flags override env
	if opts.FlagAPIKey != "" {
		cfg.APIKey = opts.FlagAPIKey
	}
	if opts.FlagAPISecret != "" {
		cfg.APISecret = opts.FlagAPISecret
	}

	return cfg, nil
}

// EffectiveIntelMCPURLs returns Info and News MCP base URLs (infoURL, newsURL).
// Precedence: non-empty GATE_INTEL_*_MCP_URL, then config file intel.*_mcp_url, then built-in defaults.
func EffectiveIntelMCPURLs(f IntelFile) (infoURL, newsURL string) {
	infoURL = strings.TrimSpace(os.Getenv("GATE_INTEL_INFO_MCP_URL"))
	if infoURL == "" {
		infoURL = strings.TrimSpace(f.InfoMCPURL)
	}
	if infoURL == "" {
		infoURL = DefaultIntelInfoMCPURL
	}
	newsURL = strings.TrimSpace(os.Getenv("GATE_INTEL_NEWS_MCP_URL"))
	if newsURL == "" {
		newsURL = strings.TrimSpace(f.NewsMCPURL)
	}
	if newsURL == "" {
		newsURL = DefaultIntelNewsMCPURL
	}
	return infoURL, newsURL
}
