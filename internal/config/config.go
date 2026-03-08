package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const defaultBaseURL = "https://api.gateio.ws"
const defaultSettle = "usdt"

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
}

type fileConfig struct {
	DefaultProfile string             `yaml:"default_profile"`
	DefaultSettle  string             `yaml:"default_settle"`
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
