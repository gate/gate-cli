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
	"strings"
	"testing"
	"time"

	"github.com/gate/gate-cli/internal/oauth2"
	"gopkg.in/yaml.v3"
)

// TestAuthStatusIssuerJSON runs the actual executable over an isolated token
// store. No issuer/API server exists, proving status is a local observation.
func TestAuthStatusIssuerJSON(t *testing.T) {
	home := t.TempDir()
	issuer := "https://issuer.example.test/oauth2"
	config := map[string]interface{}{"oauth2": map[string]interface{}{"tokens": []interface{}{map[string]interface{}{
		"client_id": "synthetic", "access_token": "synthetic-private-at", "discovery_url": issuer,
		"resource": "http://127.0.0.1:1/api/v4", "expires_at": time.Now().Add(time.Hour).Format(time.RFC3339), "scopes": "base earn",
	}}}}
	raw, err := yaml.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(home, ".gate-cli"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".gate-cli/config.yaml"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	for _, selectIssuer := range []bool{false, true} {
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		cmd := exec.CommandContext(ctx, cliBinary, "--format", "json", "cex", "auth", "status")
		cmd.Env = childEnv(home, "http://127.0.0.1:1", false)
		if selectIssuer {
			cmd.Env = append(cmd.Env, "CLI_AGENTIC_TEST_ISSUER="+issuer)
		}
		out, err := cmd.Output()
		cancel()
		if err != nil {
			t.Fatal(err)
		}
		var result map[string]interface{}
		if err := json.Unmarshal(out, &result); err != nil {
			t.Fatal(err)
		}
		want := "not_logged_in"
		if selectIssuer {
			want = "authenticated"
			if result["issuer"] != issuer || result["resource"] != "http://127.0.0.1:1/api/v4" {
				t.Fatalf("wrong credential binding: %s", out)
			}
		}
		if result["status"] != want || result["remote_checked"] != false || bytes.Contains(out, []byte("synthetic-private-at")) {
			t.Fatalf("unexpected status: %s", out)
		}
	}
}

// TestEarnMissingRouteIsFailure keeps the CLI->SDK->HTTP->error-rendering
// chain intact; only the external peer is synthetic, for both auth modes.
func TestEarnMissingRouteIsFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Gate-Trace-ID", "synthetic-route-trace")
		http.Error(w, "404 page not found", http.StatusNotFound)
	}))
	defer server.Close()
	for _, mode := range []string{"oauth", "hmac"} {
		out, stderr, err := credentialCLI(t, server.URL, mode, "base earn", "--format", "json", "cex", "earn", "uni", "records", "--page", "1", "--limit", "1")
		if err == nil || len(bytes.TrimSpace(out)) != 0 {
			t.Fatalf("route failure swallowed: %v %s", err, out)
		}
		for _, want := range []string{"API_NOT_FOUND", "gateway route", "synthetic-route-trace"} {
			if !strings.Contains(string(stderr), want) {
				t.Fatalf("lost route diagnosis %q: %s", want, stderr)
			}
		}
		if strings.Contains(string(stderr), "EMPTY_RESULT") || strings.Contains(string(stderr), "no records matched") {
			t.Fatalf("route failure fabricated an empty account: %s", stderr)
		}
	}
}

// credentialCLI isolates HOME/credentials per process. Only synthetic unexpired
// tokens are persisted: OAuth endpoint/permission decisions are API mocks here.
func credentialCLI(t *testing.T, base, mode, scope string, args ...string) ([]byte, []byte, error) {
	t.Helper()
	home := t.TempDir()
	if mode == "oauth" || mode == "both" || mode == "wrong-resource" {
		resource := base + "/api/v4"
		if mode == "wrong-resource" || mode == "both" {
			resource = "https://other.example.test/api/v4"
		}
		token := oauth2.TokenData{ClientID: "synthetic-client", DiscoveryURL: oauth2.DefaultDiscoveryURL, Resource: resource, AccessToken: "synthetic-test-" + strings.ReplaceAll(scope, " ", "-"), RefreshToken: "synthetic-refresh", ExpiresAt: time.Now().Add(time.Hour), Scopes: scope}
		raw, err := yaml.Marshal(map[string]interface{}{"oauth2": map[string]interface{}{"token": token}})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(home, ".gate-cli"), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(home, ".gate-cli/config.yaml"), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, cliBinary, args...)
	cmd.Env = childEnv(home, base, mode == "hmac" || mode == "both")
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	return out.Bytes(), stderr.Bytes(), err
}
