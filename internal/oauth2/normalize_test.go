package oauth2

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeDiscoveryURL(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty passes through", "", ""},
		{"whitespace trimmed to empty", "   ", ""},
		{"bare prefix", "https://www.gate.com", "https://www.gate.com/.well-known/oauth-authorization-server"},
		{"prefix with trailing slash", "https://www.gate.com/", "https://www.gate.com/.well-known/oauth-authorization-server"},
		{"full URL pass through", "https://www.gate.com/.well-known/oauth-authorization-server", "https://www.gate.com/.well-known/oauth-authorization-server"},
		{"http scheme OK", "http://dev.example.test", "http://dev.example.test/.well-known/oauth-authorization-server"},
		{"preprod prefix", "https://staging.example.test", "https://staging.example.test/.well-known/oauth-authorization-server"},
		{"whitespace trimmed around prefix", "  https://www.gate.com  ", "https://www.gate.com/.well-known/oauth-authorization-server"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NormalizeDiscoveryURL(tc.in)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestResolveDiscoveryURL_AcceptsShortForm 旧生产覆盖项被忽略，仍使用固定默认值。
func TestResolveDiscoveryURL_AcceptsShortForm(t *testing.T) {
	home := isolateHome(t)
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".gate-cli"), 0700))
	cfgYAML := []byte("oauth2:\n  discovery_url: https://www.gate.com\n")
	require.NoError(t, os.WriteFile(filepath.Join(home, ".gate-cli", "config.yaml"), cfgYAML, 0600))

	got := resolveDiscoveryURL()
	assert.Equal(t,
		"https://www.gate.com/.well-known/oauth-authorization-server",
		got,
		"short form in config.yaml should be normalized to full RFC 8414 URL",
	)
}

func TestShortDiscoveryURL(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"whitespace", "   ", ""},
		{"already short", "https://www.gate.com", "https://www.gate.com"},
		{"short with trailing slash", "https://www.gate.com/", "https://www.gate.com"},
		{"full form", "https://www.gate.com/.well-known/oauth-authorization-server", "https://www.gate.com"},
		{"full form preprod", "https://staging.example.test/.well-known/oauth-authorization-server", "https://staging.example.test"},
		{"http scheme", "http://dev.example.test/.well-known/oauth-authorization-server", "http://dev.example.test"},
		{"whitespace padded", "  https://www.gate.com/.well-known/oauth-authorization-server  ", "https://www.gate.com"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ShortDiscoveryURL(tc.in))
		})
	}
}

// TestSaveLoadShortForm_Roundtrip 验证 SaveToken 用完整 URL 作 DiscoveryURL 写入，
// 落盘后 YAML 里是短形式；LoadToken 读回时内存展开回完整 URL。
func TestSaveLoadShortForm_Roundtrip(t *testing.T) {
	_, cleanup := setupTestTokenDir(t)
	defer cleanup()

	fullURL := "https://example.invalid/.well-known/oauth-authorization-server"
	shortURL := "https://example.invalid"
	ensureActiveDiscoveryURL(t, shortURL) // config.yaml 里用户写短形式

	require.NoError(t, SaveToken(&TokenData{
		ClientID:     "pkce_roundtrip",
		DiscoveryURL: fullURL,
		AccessToken:  "at",
		RefreshToken: "rt",
	}))

	// YAML 落盘应为短形式。
	cfgPath, err := ConfigYAMLPath()
	require.NoError(t, err)
	raw, err := os.ReadFile(cfgPath)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "discovery_url: "+shortURL)
	assert.NotContains(t, string(raw), "discovery_url: "+fullURL,
		"落盘的 discovery_url 不应出现完整 URL")

	// LoadToken 在内存里把 DiscoveryURL 展开回完整 URL。
	loaded, err := LoadToken()
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, fullURL, loaded.DiscoveryURL,
		"内存 TokenData.DiscoveryURL 应为完整 URL")
}

// TestResolveDiscoveryURL_PrefixWithTrailingSlash 旧非生产覆盖项也不能切换信任源。
func TestResolveDiscoveryURL_PrefixWithTrailingSlash(t *testing.T) {
	home := isolateHome(t)
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".gate-cli"), 0700))
	cfgYAML := []byte("oauth2:\n  discovery_url: https://staging.example.test/\n")
	require.NoError(t, os.WriteFile(filepath.Join(home, ".gate-cli", "config.yaml"), cfgYAML, 0600))

	got := resolveDiscoveryURL()
	assert.Equal(t,
		DefaultDiscoveryURL,
		got,
	)
}
