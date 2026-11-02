//go:build agentic_e2e

package agentic_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// Exercise the actual executable and its old command paths, not a mocked Cobra
// tree. Synthetic credentials/API peers prevent touching real user accounts.
func TestSubaccountRetirementProcesses(t *testing.T) {
	var calls atomic.Int32
	peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != "GET" || r.URL.Path != "/api/v4/wallet/sub_account_balances" {
			t.Errorf("unexpected retained wallet request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, "[]"); err != nil {
			t.Error(err)
		}
	}))
	defer peer.Close()
	commands := []string{"create", "get", "list", "lock", "unlock", "unified-mode", "key create", "key delete", "key get", "key list", "key update"}

	for _, mode := range []string{"anonymous", "hmac", "oauth", "both"} {
		for _, command := range commands {
			for _, suffix := range [][]string{nil, {"--help"}, {"--unknown"}} {
				t.Run(mode+"/"+command+"/"+strings.Join(suffix, "-"), func(t *testing.T) {
					args := append([]string{"--format", "json", "cex", "sub-account"}, strings.Fields(command)...)
					args = append(args, suffix...)
					_, stderr, err := credentialCLI(t, peer.URL, mode, "base account asset", args...)
					// Cobra rejects malformed flags before resolving the retired command.
					expected := "sub-account"
					if len(suffix) > 0 && suffix[0] == "--unknown" {
						expected = "unknown flag: --unknown"
					}
					if err == nil || !strings.Contains(string(stderr), expected) {
						t.Fatalf("retired command accepted: err=%v stderr=%s", err, stderr)
					}
				})
			}
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("retired API calls=%d", calls.Load())
	}
	// Positive controls prove the peer and retained wallet SDK wiring still work.
	for _, authenticated := range []bool{true} {
		t.Run(fmt.Sprintf("retained-earn/auth=%v", authenticated), func(t *testing.T) {
			before := calls.Load()
			stdout, stderr, err := runCLI(t, peer.URL, authenticated, "--format", "json", "cex", "wallet", "balance", "sub")
			if err != nil || !json.Valid(stdout) || calls.Load() != before+1 {
				t.Fatalf("Wallet broken: err=%v stdout=%s stderr=%s", err, stdout, stderr)
			}
		})
	}
}

// Discovery must not advertise removed commands through help or completion.
func TestSubaccountRetirementDiscovery(t *testing.T) {
	for _, args := range [][]string{{"cex", "--help"}, {"__complete", "cex", ""}} {
		stdout, stderr, err := runCLI(t, "http://127.0.0.1:1", false, args...)
		if err != nil || !strings.Contains(string(stdout), "wallet") {
			t.Fatalf("discovery failed: err=%v stderr=%s", err, stderr)
		}
		for _, line := range strings.Split(string(stdout), "\n") {
			fields := strings.Fields(line)
			if len(fields) > 0 && fields[0] == "sub-account" {
				t.Fatalf("discovery advertises launch: %s", line)
			}
		}
	}
}
