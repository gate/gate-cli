//go:build agentic_e2e

package agentic_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Display tests use an isolated config and the real executable. The explicit
// opt-in remains available; default output must hide arbitrary custom headers.
func TestPublicationConfigurationCLI(t *testing.T) {
	for _, tc := range []struct {
		name, yaml string
		show, fail bool
	}{
		{"default", "intel: {extra_headers: {X-Api-Key: synthetic-display-secret}}\ndefault_settle: usdt\n", false, false},
		{"endpoint", "intel: {info_mcp_url: 'https://user:synthetic-display-secret@example.invalid/mcp?key=synthetic-display-secret'}\ndefault_settle: usdt\n", false, false},
		{"explicit", "intel: {extra_headers: {X-Api-Key: synthetic-display-secret}}\n", true, false},
		{"invalid", "intel: [synthetic-display-secret", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			if err := os.Mkdir(filepath.Join(home, ".gate-cli"), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(home, ".gate-cli/config.yaml"), []byte(tc.yaml), 0600); err != nil {
				t.Fatal(err)
			}
			args := []string{"config", "list"}
			if tc.show {
				args = append(args, "--show-secrets")
			}
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, cliBinary, args...)
			cmd.Env = childEnv(home, "", false)
			out, err := cmd.CombinedOutput()
			if (err != nil) != tc.fail {
				t.Fatalf("unexpected exit: %v %s", err, out)
			}
			if bytes.Contains(out, []byte("synthetic-display-secret")) != tc.show {
				t.Fatal("wrong config disclosure boundary")
			}
			if !tc.show && !tc.fail && !bytes.Contains(out, []byte("usdt")) {
				t.Fatal("normal configuration lost")
			}
		})
	}
}

// Only the remote JSON-RPC peer is mocked. Verify URL parameters still reach it
// while protocol-error diagnostics and the CLI error envelope remain redacted.
func TestPublicationDiagnosticsCLI(t *testing.T) {
	for _, backend := range []string{"info", "news"} {
		t.Run(backend, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Query().Get("api_key") != "synthetic-query-secret" {
					t.Error("request parameter changed")
				}
				var request struct {
					ID interface{} `json:"id"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(map[string]interface{}{"jsonrpc": "2.0", "id": request.ID, "error": map[string]interface{}{"code": -32001, "message": "upstream HTTPS://example.invalid/mcp?token=synthetic-echo-secret refused"}}); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, cliBinary, backend, "list", "--debug")
			cmd.Env = append(childEnv(t.TempDir(), "", false), "GATE_INTEL_"+strings.ToUpper(backend)+"_MCP_URL="+server.URL+"?api_key=synthetic-query-secret")
			out, err := cmd.CombinedOutput()
			if err == nil || calls.Load() != 1 {
				t.Fatalf("failure/request count: %v %d", err, calls.Load())
			}
			for _, secret := range []string{"synthetic-query-secret", "synthetic-echo-secret"} {
				if bytes.Contains(out, []byte(secret)) {
					t.Fatal("diagnostic exposed credentials")
				}
			}
			if !bytes.Contains(out, []byte("rpc_method=initialize")) || !bytes.Contains(out, []byte("INTEL_PROTOCOL_ERROR")) {
				t.Fatalf("safe failure context lost: %s", out)
			}
		})
	}
}

// Embedded public specifications and help are real offline output surfaces.
func TestPublicationSpecificationsCLI(t *testing.T) {
	private := regexp.MustCompile(`\b(?:dim|dwd|dws)_[a-zA-Z0-9_]+\b|source_of_truth|source_repo|internal/mcphost/`)
	for _, args := range [][]string{{"info", "mcp-spec", "--format", "json"}, {"news", "mcp-spec", "--format", "json"}, {"news", "prediction", "get-event-signal", "--help"}, {"info", "mcp-spec", "--help"}} {
		t.Run(strings.Join(args, "/"), func(t *testing.T) {
			stdout, stderr, err := runCLI(t, "", false, args...)
			if err != nil {
				t.Fatalf("offline output: %v %s", err, stderr)
			}
			if private.Match(stdout) || private.Match(stderr) {
				t.Fatal("public specification exposes implementation metadata")
			}
			if len(stdout) == 0 {
				t.Fatal("public output disappeared")
			}
		})
	}
}
