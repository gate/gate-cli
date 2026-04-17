package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gate/gate-cli/internal/config"
)

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgFile, []byte(`
default_profile: main
default_settle: usdt
intel:
  news_mcp_url: "https://cfg.example/mcp/news"
  info_mcp_url: "https://cfg.example/mcp/info"
  bearer_token: "file-bearer"
  extra_headers:
    rule: data-mcp
  http_timeout: "90s"
profiles:
  main:
    api_key: "key123"
    api_secret: "secret456"
    base_url: "https://api.gateio.ws"
`), 0600)

	cfg, err := config.Load(config.Options{ConfigFile: cfgFile, Profile: "main"})
	require.NoError(t, err)
	assert.Equal(t, "key123", cfg.APIKey)
	assert.Equal(t, "secret456", cfg.APISecret)
	assert.Equal(t, "https://api.gateio.ws", cfg.BaseURL)
	assert.Equal(t, "usdt", cfg.DefaultSettle)
	assert.Equal(t, "https://cfg.example/mcp/news", cfg.Intel.NewsMCPURL)
	assert.Equal(t, "https://cfg.example/mcp/info", cfg.Intel.InfoMCPURL)
	assert.Equal(t, "file-bearer", cfg.Intel.BearerToken)
	assert.Equal(t, "data-mcp", cfg.Intel.ExtraHeaders["rule"])
	assert.Equal(t, "90s", cfg.Intel.HTTPTimeout)
}

func TestEffectiveIntelMCPURLs_BuiltinDefaults(t *testing.T) {
	t.Setenv("GATE_INTEL_INFO_MCP_URL", "")
	t.Setenv("GATE_INTEL_NEWS_MCP_URL", "")
	infoURL, newsURL := config.EffectiveIntelMCPURLs(config.IntelFile{})
	assert.Equal(t, config.DefaultIntelInfoMCPURL, infoURL)
	assert.Equal(t, config.DefaultIntelNewsMCPURL, newsURL)
}

func TestEffectiveIntelMCPURLs_EnvOverridesFile(t *testing.T) {
	t.Setenv("GATE_INTEL_INFO_MCP_URL", "https://env.example/info")
	t.Setenv("GATE_INTEL_NEWS_MCP_URL", "")
	f := config.IntelFile{
		InfoMCPURL: "https://file.example/info",
		NewsMCPURL: "https://file.example/news",
	}
	infoURL, newsURL := config.EffectiveIntelMCPURLs(f)
	assert.Equal(t, "https://env.example/info", infoURL)
	assert.Equal(t, "https://file.example/news", newsURL)
}

func TestEnvVarOverridesFile(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgFile, []byte(`
profiles:
  default:
    api_key: "file-key"
    api_secret: "file-secret"
`), 0600)

	t.Setenv("GATE_API_KEY", "env-key")
	t.Setenv("GATE_API_SECRET", "env-secret")

	cfg, err := config.Load(config.Options{ConfigFile: cfgFile, Profile: "default"})
	require.NoError(t, err)
	assert.Equal(t, "env-key", cfg.APIKey)
	assert.Equal(t, "env-secret", cfg.APISecret)
}

func TestFlagOverridesEnv(t *testing.T) {
	t.Setenv("GATE_API_KEY", "env-key")

	cfg, err := config.Load(config.Options{
		Profile:    "default",
		FlagAPIKey: "flag-key",
	})
	require.NoError(t, err)
	assert.Equal(t, "flag-key", cfg.APIKey)
}

func TestDefaultBaseURL(t *testing.T) {
	cfg, err := config.Load(config.Options{Profile: "default"})
	require.NoError(t, err)
	assert.Equal(t, "https://api.gateio.ws", cfg.BaseURL)
}

func TestDefaultSettle(t *testing.T) {
	cfg, err := config.Load(config.Options{Profile: "default"})
	require.NoError(t, err)
	assert.Equal(t, "usdt", cfg.DefaultSettle)
}
