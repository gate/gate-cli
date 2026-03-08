//go:build integration

// Package integration provides helpers for integration tests that call live Gate API endpoints.
// Tests in this package require testdata/integration.yaml with valid api_key and api_secret.
package integration

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/config"
)

const configFile = "testdata/integration.yaml"

// projectRoot returns the repository root (two levels up from this file's package).
func projectRoot() string {
	_, file, _, _ := runtime.Caller(0)
	// file = .../internal/integration/helper.go → go up two dirs
	return filepath.Join(filepath.Dir(file), "..", "..")
}

// LoadClient loads the integration test config and returns a ready client.
// It calls t.Fatal if:
//   - testdata/integration.yaml is missing
//   - api_key or api_secret is empty
func LoadClient(t *testing.T) *client.Client {
	t.Helper()

	cfgPath := filepath.Join(projectRoot(), configFile)
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Fatalf("integration config not found: %s\n"+
			"Copy testdata/integration.yaml.example → testdata/integration.yaml and fill in your keys.", cfgPath)
	}

	cfg, err := config.Load(config.Options{ConfigFile: cfgPath, Profile: "testnet"})
	if err != nil {
		t.Fatalf("failed to load integration config: %v", err)
	}

	if cfg.APIKey == "" || cfg.APISecret == "" {
		t.Fatalf("integration config %s has empty api_key or api_secret\n"+
			"Fill in your Gate testnet credentials to run integration tests.", cfgPath)
	}

	c, err := client.New(cfg)
	if err != nil {
		t.Fatalf("failed to create Gate client: %v", err)
	}
	return c
}
