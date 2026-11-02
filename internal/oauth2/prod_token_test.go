package oauth2

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSaveToken_ProductionGoesToOAuth2Token 生产环境 token 落盘到 oauth2.token
// 单例节点，且 YAML 里不带 discovery_url 字段。
func TestSaveToken_ProductionGoesToOAuth2Token(t *testing.T) {
	_, cleanup := setupTestTokenDir(t)
	defer cleanup()

	ensureActiveDiscoveryURL(t, "https://www.gate.com") // 生产
	require.NoError(t, SaveToken(&TokenData{
		ClientID:     "pkce_prod",
		DiscoveryURL: DefaultDiscoveryURL,
		AccessToken:  "at_prod",
		RefreshToken: "rt_prod",
		ExpiresAt:    time.Now().Add(time.Hour).Truncate(time.Second),
		Scopes:       "read",
	}))

	cfgPath, _ := ConfigYAMLPath()
	raw, err := os.ReadFile(cfgPath)
	require.NoError(t, err)
	out := string(raw)

	// 落盘到单例 oauth2.token
	assert.Contains(t, out, "token:")
	assert.Contains(t, out, "client_id: pkce_prod")
	assert.Contains(t, out, "access_token: at_prod")
	// 且单例节点内没有 discovery_url（omitempty + 写时置空）
	// 生产单例不写 issuer 字段，且不再创建根级发现覆盖项。
	assert.Equal(t, 0, strings.Count(out, "discovery_url:"),
		"生产 oauth2.token 下不应含 discovery_url 字段")
	// 不应写 tokens[]
	assert.NotContains(t, out, "tokens:")
}

// TestSaveToken_NonProductionGoesToTokensArray 非生产环境 token 写 oauth2.tokens[]。
func TestSaveToken_NonProductionGoesToTokensArray(t *testing.T) {
	_, cleanup := setupTestTokenDir(t)
	defer cleanup()

	preprod := "https://staging.example.test"
	ensureActiveDiscoveryURL(t, preprod)
	require.NoError(t, SaveToken(&TokenData{
		ClientID:     "pkce_preprod",
		DiscoveryURL: preprod,
		AccessToken:  "at_pp",
		RefreshToken: "rt_pp",
		ExpiresAt:    time.Now().Add(time.Hour).Truncate(time.Second),
	}))

	cfgPath, _ := ConfigYAMLPath()
	raw, _ := os.ReadFile(cfgPath)
	out := string(raw)

	assert.Contains(t, out, "tokens:")
	assert.Contains(t, out, "discovery_url: "+preprod)
	assert.Contains(t, out, "client_id: pkce_preprod")
	// 生产单例 key "token:" 不应出现（除了 tokens: 里的 access_token/refresh_token）。
	assert.NotContains(t, out, "\n    token:\n", "不应写入生产单例")
}

// TestLoadToken_ProductionReadsOAuth2TokenFirst 读路由：oauth2.token 存在即返回，
// 不再读 tokens[] 的同 URL 条目。
func TestLoadToken_ProductionReadsOAuth2TokenFirst(t *testing.T) {
	_, cleanup := setupTestTokenDir(t)
	defer cleanup()

	ensureActiveDiscoveryURL(t, "https://www.gate.com")
	// 同时人工放一条"过期的 tokens[]"旁路条目，确保被绕过。
	home, _ := resolveHome()
	cfgPath := filepath.Join(home, ".gate-cli", "config.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(`oauth2:
  discovery_url: https://www.gate.com
  token:
    client_id: pkce_singleton
    access_token: at_singleton
    refresh_token: rt_singleton
    expires_at: "2099-01-01T00:00:00Z"
    scopes: read
  tokens:
    - discovery_url: https://www.gate.com
      client_id: pkce_stale
      access_token: at_stale
      refresh_token: rt_stale
`), 0600))

	loaded, err := LoadToken()
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, "pkce_singleton", loaded.ClientID, "应从 oauth2.token 读取")
	assert.Equal(t, "at_singleton", loaded.AccessToken)
	// 内存 DiscoveryURL 被展开成完整 URL
	assert.Equal(t, DefaultDiscoveryURL, loaded.DiscoveryURL)
}

// TestLoadToken_ProductionFallbackToTokensArray 当 oauth2.token 不存在时，
// 回退到 tokens[] 里匹配 https://www.gate.com 的条目。
func TestLoadToken_ProductionFallbackToTokensArray(t *testing.T) {
	_, cleanup := setupTestTokenDir(t)
	defer cleanup()

	home, _ := resolveHome()
	cfgPath := filepath.Join(home, ".gate-cli", "config.yaml")
	// 只放 tokens[] 的生产条目，不写 oauth2.token
	require.NoError(t, os.WriteFile(cfgPath, []byte(`oauth2:
  discovery_url: https://www.gate.com/
  tokens:
    - discovery_url: https://www.gate.com
      client_id: pkce_fallback
      access_token: at_fb
`), 0600))

	loaded, err := LoadToken()
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, "pkce_fallback", loaded.ClientID)
	assert.Equal(t, "at_fb", loaded.AccessToken)
}

// TestSaveToken_ProductionRemovesFromTokensArray 写生产时，tokens[] 里的存量
// 生产条目被清理掉，保持单一来源。
func TestSaveToken_ProductionRemovesFromTokensArray(t *testing.T) {
	_, cleanup := setupTestTokenDir(t)
	defer cleanup()

	home, _ := resolveHome()
	cfgPath := filepath.Join(home, ".gate-cli", "config.yaml")
	// 初始：tokens[] 里有一条 https://www.gate.com 的旧条目 + 一条预发条目
	require.NoError(t, os.WriteFile(cfgPath, []byte(`oauth2:
  discovery_url: https://www.gate.com
  tokens:
    - discovery_url: https://www.gate.com
      client_id: pkce_old_prod
      access_token: at_old
    - discovery_url: https://staging.example.test
      client_id: pkce_preprod
      access_token: at_pp
`), 0600))

	// 写一个生产新 token
	require.NoError(t, SaveToken(&TokenData{
		ClientID:     "pkce_new_prod",
		DiscoveryURL: DefaultDiscoveryURL,
		AccessToken:  "at_new",
		RefreshToken: "rt_new",
		ExpiresAt:    time.Now().Add(time.Hour).Truncate(time.Second),
	}))

	raw, _ := os.ReadFile(cfgPath)
	out := string(raw)

	// 生产条目从 tokens[] 被移除
	assert.NotContains(t, out, "pkce_old_prod")
	assert.NotContains(t, out, "at_old")
	// 预发条目保留
	assert.Contains(t, out, "pkce_preprod")
	assert.Contains(t, out, "https://staging.example.test")
	// 新 token 落到单例
	assert.Contains(t, out, "pkce_new_prod")
	assert.Contains(t, out, "at_new")
}

// TestIsProductionDiscovery 斜杠变体都应判定为生产。
func TestIsProductionDiscovery(t *testing.T) {
	for _, in := range []string{
		"https://www.gate.com",
		"https://www.gate.com/",
		"https://www.gate.com/.well-known/oauth-authorization-server",
		"  https://www.gate.com/  ",
	} {
		assert.True(t, isProductionDiscovery(in), "should be production: %q", in)
	}
	for _, in := range []string{
		"",
		"https://staging.example.test",
		"http://dev.example.test",
		"https://www.gate.io",
	} {
		assert.False(t, isProductionDiscovery(in), "should NOT be production: %q", in)
	}
}
