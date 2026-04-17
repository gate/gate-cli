package toolconfig

import (
	"testing"
	"time"

	"github.com/gate/gate-cli/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveNewsFromEnv(t *testing.T) {
	t.Setenv("GATE_INTEL_NEWS_MCP_URL", "https://example.com/mcp/news")
	t.Setenv("GATE_INTEL_BEARER_TOKEN", "global-token")
	t.Setenv("GATE_INTEL_EXTRA_HEADERS", `{"rule":"data-mcp","x-custom":123}`)
	t.Setenv("GATE_INTEL_HTTP_TIMEOUT", "45s")

	cfg, err := Resolve(ResolveOptions{Backend: "news"})
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/mcp/news", cfg.BaseURL)
	assert.Equal(t, "global-token", cfg.BearerToken)
	assert.Equal(t, "data-mcp", cfg.ExtraHeaders["rule"])
	assert.Equal(t, "123", cfg.ExtraHeaders["x-custom"])
	assert.Equal(t, 45*time.Second, cfg.Timeout)
}

func TestResolvePerBackendTokenWins(t *testing.T) {
	t.Setenv("GATE_INTEL_NEWS_MCP_URL", "https://example.com/mcp/news")
	t.Setenv("GATE_INTEL_BEARER_TOKEN", "global-token")
	t.Setenv("GATE_INTEL_NEWS_BEARER_TOKEN", "news-token")

	cfg, err := Resolve(ResolveOptions{Backend: "news"})
	require.NoError(t, err)
	assert.Equal(t, "news-token", cfg.BearerToken)
}

func TestResolveNewsBuiltinDefaultURL(t *testing.T) {
	t.Setenv("GATE_INTEL_NEWS_MCP_URL", "")
	cfg, err := Resolve(ResolveOptions{Backend: "news"})
	require.NoError(t, err)
	assert.Equal(t, config.DefaultIntelNewsMCPURL, cfg.BaseURL)
}

func TestResolveUnknownBackendMissingURL(t *testing.T) {
	_, err := Resolve(ResolveOptions{Backend: "docs"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "intel endpoint URL is required")
}

func TestResolveInvalidHeaders(t *testing.T) {
	t.Setenv("GATE_INTEL_NEWS_MCP_URL", "https://example.com/mcp/news")
	t.Setenv("GATE_INTEL_EXTRA_HEADERS", "{bad json")

	_, err := Resolve(ResolveOptions{Backend: "news"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "GATE_INTEL_EXTRA_HEADERS")
}

func TestResolveDeniedExtraHeader(t *testing.T) {
	t.Setenv("GATE_INTEL_NEWS_MCP_URL", "https://example.com/mcp/news")
	t.Setenv("GATE_INTEL_EXTRA_HEADERS", `{"Authorization":"x"}`)

	_, err := Resolve(ResolveOptions{Backend: "news"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed")
}

func TestResolveTimeoutSecondsFallback(t *testing.T) {
	t.Setenv("GATE_INTEL_NEWS_MCP_URL", "https://example.com/mcp/news")
	t.Setenv("GATE_INTEL_HTTP_TIMEOUT", "30")

	cfg, err := Resolve(ResolveOptions{Backend: "news"})
	require.NoError(t, err)
	assert.Equal(t, 30*time.Second, cfg.Timeout)
}

func TestResolveNewsURLFromConfigFile(t *testing.T) {
	t.Setenv("GATE_INTEL_NEWS_MCP_URL", "")
	t.Setenv("GATE_INTEL_BEARER_TOKEN", "")

	cfg, err := Resolve(ResolveOptions{
		Backend: "news",
		IntelFile: config.IntelFile{
			NewsMCPURL:  "https://file.example/mcp/news",
			BearerToken: "yaml-token",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "https://file.example/mcp/news", cfg.BaseURL)
	assert.Equal(t, "yaml-token", cfg.BearerToken)
}

func TestResolveEnvOverridesFileURL(t *testing.T) {
	t.Setenv("GATE_INTEL_NEWS_MCP_URL", "https://env.example/mcp/news")

	cfg, err := Resolve(ResolveOptions{
		Backend: "news",
		IntelFile: config.IntelFile{
			NewsMCPURL: "https://file.example/mcp/news",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "https://env.example/mcp/news", cfg.BaseURL)
}

func TestResolveExtraHeadersFromFileWhenEnvUnset(t *testing.T) {
	t.Setenv("GATE_INTEL_NEWS_MCP_URL", "https://example.com/mcp/news")
	t.Setenv("GATE_INTEL_EXTRA_HEADERS", "")

	cfg, err := Resolve(ResolveOptions{
		Backend: "news",
		IntelFile: config.IntelFile{
			ExtraHeaders: map[string]string{"rule": "data-mcp", "x-from": "yaml"},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "data-mcp", cfg.ExtraHeaders["rule"])
	assert.Equal(t, "yaml", cfg.ExtraHeaders["x-from"])
}

func TestResolveHTTPTimeoutFromFileWhenEnvUnset(t *testing.T) {
	t.Setenv("GATE_INTEL_NEWS_MCP_URL", "https://example.com/mcp/news")
	t.Setenv("GATE_INTEL_HTTP_TIMEOUT", "")

	cfg, err := Resolve(ResolveOptions{
		Backend: "news",
		IntelFile: config.IntelFile{
			HTTPTimeout: "120",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 120*time.Second, cfg.Timeout)
}
