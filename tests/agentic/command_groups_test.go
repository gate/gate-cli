//go:build agentic_e2e

package agentic_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"gopkg.in/yaml.v3"
)

// Execute every independently reviewed visible group through the actual CLI.
// Only the HTTP peer is mocked; invalid paths and bare-group help must never reach it.
func TestCommandGroupsCLI(t *testing.T) {
	raw, err := os.ReadFile("fixtures/command_groups.json")
	if err != nil {
		t.Fatal(err)
	}
	var groups [][]string
	if err := json.Unmarshal(raw, &groups); err != nil || len(groups) < 85 {
		t.Fatalf("invalid group fixture: %v groups=%d", err, len(groups))
	}
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)
	for _, path := range groups {
		name := strings.Join(path, " ")
		if name == "" {
			name = "root"
		}
		for _, format := range []string{"json", "pretty", "table"} {
			t.Run(name+"/invalid/"+format, func(t *testing.T) {
				args := append([]string{"--format", format}, path...)
				args = append(args, "nosuch")
				stdout, stderr, err := runCLI(t, server.URL, false, args...)
				var exit *exec.ExitError
				if !errors.As(err, &exit) || exit.ExitCode() != 1 || len(stdout) != 0 {
					t.Fatalf("unknown path exit=%v stdout=%s stderr=%s", err, stdout, stderr)
				}
				parent := "gate-cli"
				if len(path) != 0 {
					parent += " " + strings.Join(path, " ")
				}
				if !strings.Contains(string(stderr), "unknown command") || !strings.Contains(string(stderr), "nosuch") || !strings.Contains(string(stderr), parent) {
					t.Fatalf("incomplete diagnostic: %s", stderr)
				}
				if format == "json" {
					// Cobra may prefix root errors with context and append a Hint.
					// Require exactly one complete standard error envelope between them.
					var envelope struct {
						Error struct {
							Status  int
							Label   string
							Message string
						}
					}
					var count int
					for _, line := range bytes.Split(stderr, []byte{'\n'}) {
						if !bytes.HasPrefix(line, []byte(`{"error":`)) {
							continue
						}
						if err := json.Unmarshal(line, &envelope); err != nil {
							t.Fatalf("malformed error envelope: %v %s", err, line)
						}
						count++
					}
					if count != 1 || envelope.Error.Status != 400 || envelope.Error.Label != "INVALID_ARGUMENTS" || !strings.Contains(envelope.Error.Message, "unknown command") {
						t.Fatalf("error envelope=%+v count=%d stderr=%s", envelope, count, stderr)
					}
				}
			})
		}
		t.Run(name+"/bare-help", func(t *testing.T) {
			args := append([]string{"--format", "json"}, path...)
			stdout, stderr, err := runCLI(t, server.URL, false, args...)
			parent := "gate-cli"
			if len(path) != 0 {
				parent += " " + strings.Join(path, " ")
			}
			if err != nil || len(stderr) != 0 || !strings.Contains(string(stdout), parent+" [command]") {
				t.Fatalf("bare help target=%q exit=%v stdout=%s stderr=%s", parent, err, stdout, stderr)
			}
		})
	}
	if calls.Load() != 0 {
		t.Fatalf("command routing made %d HTTP requests", calls.Load())
	}
	t.Logf("validated %d reviewed groups in three error formats and bare help", len(groups))
}

// Cover syntax boundaries separately from the group inventory, including the
// unsupported relative-help form, framework completion and positional config writes.
func TestCommandGroupBoundariesCLI(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)
	for _, tc := range []struct {
		name string
		args []string
		bad  bool
		want string
	}{
		{"relative-help-unknown", []string{"cex", "spot", "help", "nosuch"}, true, `unknown command "help"`},
		{"relative-help-known-word", []string{"cex", "spot", "help", "market"}, true, `unknown command "help"`},
		{"false-help", []string{"cex", "spot", "market", "nosuch", "--help=false"}, true, "unknown command"},
		{"false-short-help", []string{"cex", "spot", "market", "nosuch", "-h=false"}, true, "unknown command"},
		{"separator", []string{"cex", "spot", "market", "--", "nosuch"}, true, "unknown command"},
		{"flag-value", []string{"--profile", "nosuch", "cex", "spot", "market", "bad-child"}, true, `unknown command "bad-child"`},
		{"missing-flag-value", []string{"cex", "spot", "market", "--profile"}, true, "flag needs an argument"},
		{"explicit-help", []string{"help", "cex", "spot", "market"}, false, "gate-cli cex spot market [command]"},
		{"flag-help", []string{"cex", "spot", "market", "-h"}, false, "gate-cli cex spot market [command]"},
		{"profile-is-not-topic", []string{"--profile", "nosuch", "cex", "spot", "market", "--help"}, false, "gate-cli cex spot market [command]"},
		{"required-flag-not-needed", []string{"cex", "alpha", "order", "place", "--help"}, false, "Usage:"},
		{"alias-help", []string{"help", "info", "coin-overview"}, false, "Usage:"},
		{"completion-help", []string{"help", "completion", "bash", "--help"}, false, "Usage:"},
		{"completion-script", []string{"completion", "bash"}, false, "bash completion"},
		{"dynamic-completion", []string{"__complete", "cex", "spot", "market", ""}, false, "ticker"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, err := runCLI(t, server.URL, false, append([]string{"--format", "table"}, tc.args...)...)
			if tc.bad {
				var exit *exec.ExitError
				if !errors.As(err, &exit) || exit.ExitCode() != 1 || len(stdout) != 0 || !strings.Contains(string(stderr), tc.want) {
					t.Fatalf("invalid syntax exit=%v stdout=%s stderr=%s", err, stdout, stderr)
				}
			} else if err != nil || !strings.Contains(string(stdout), tc.want) {
				t.Fatalf("valid syntax exit=%v stdout=%s stderr=%s", err, stdout, stderr)
			}
		})
	}
	if calls.Load() != 0 {
		t.Fatalf("routing/help/config made %d HTTP requests", calls.Load())
	}
}

// Preserve the leaf's two positional arguments through real Cobra dispatch and
// verify the resulting file, using a pre-existing configuration required by set.
func TestCommandGroupPositionalLeafCLI(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".gate-cli", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("profiles:\n  default:\n    base_url: http://127.0.0.1:4321\n    api_key: synthetic-preserved\n"), 0600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, err := sameHomeCLI(t, home, "http://127.0.0.1:1", "config", "set", "base-url", "http://127.0.0.1:1234")
	if err != nil || len(stderr) != 0 || !strings.Contains(string(stdout), "Set base-url") {
		t.Fatalf("positional config leaf exit=%v stdout=%s stderr=%s", err, stdout, stderr)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var saved struct {
		Profiles map[string]struct {
			BaseURL string `yaml:"base_url"`
			APIKey  string `yaml:"api_key"`
		}
	}
	if err := yaml.Unmarshal(raw, &saved); err != nil || saved.Profiles["default"].BaseURL != "http://127.0.0.1:1234" || saved.Profiles["default"].APIKey != "synthetic-preserved" {
		t.Fatalf("positional values were lost: decode=%v saved=%s", err, raw)
	}
}
