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

type fileConfig struct {
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
