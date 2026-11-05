//go:build agentic_e2e

package agentic_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// publishedBinaryPath never downloads, replaces, builds or owns the artifact.
// The operator verifies its official checksum/provenance before selecting it.
func publishedBinaryPath(path string, agentBuild bool) (string, error) {
	// A standard release cannot certify an Agent-tagged harness.
	if agentBuild {
		return "", fmt.Errorf("published CLI mode requires the standard test build")
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("published CLI binary path must be absolute")
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("published CLI binary: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("published CLI binary must be a regular file")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0111 == 0 {
		return "", fmt.Errorf("published CLI binary must be executable")
	}
	return filepath.Clean(path), nil
}

// Validation failures must never fall back to compiling a different binary.
func TestPublishedBinaryPath(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "fixture")
	if err := os.WriteFile(file, []byte("validation fixture; never executed"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, path string }{
		{"empty", ""}, {"relative", "fixture"}, {"missing", filepath.Join(dir, "missing")}, {"directory", dir},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := publishedBinaryPath(tc.path, false); err == nil || got != "" {
				t.Fatalf("invalid artifact accepted: %q %v", got, err)
			}
		})
	}
	if runtime.GOOS != "windows" {
		if got, err := publishedBinaryPath(file, false); err == nil || got != "" {
			t.Fatalf("non-executable artifact accepted: %q %v", got, err)
		}
	}
	if err := os.Chmod(file, 0700); err != nil {
		t.Fatal(err)
	}
	if got, err := publishedBinaryPath(file, true); err == nil || got != "" {
		t.Fatalf("standard artifact accepted for Agent harness: %q %v", got, err)
	}
	got, err := publishedBinaryPath(file, false)
	if err != nil || got != file {
		t.Fatalf("valid absolute executable rejected: %q %v", got, err)
	}
}
