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
func TestLaunchRetirementProcesses(t *testing.T) {
	var calls atomic.Int32
	peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != "GET" || r.URL.Path != "/api/v4/earn/uni/currencies" {
			t.Errorf("unexpected retained Earn request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, "[]"); err != nil {
			t.Error(err)
		}
	}))
	defer peer.Close()
	commands := []string{
		"projects", "pledge", "redeem", "pledge-records", "reward-records",
		"hodler projects", "hodler order", "hodler order-records", "hodler airdrop-records",
		"candy-drop activities", "candy-drop rules", "candy-drop register",
		"candy-drop progress", "candy-drop participations", "candy-drop airdrops",
	}
	for _, mode := range []string{"anonymous", "hmac", "oauth", "both"} {
		for _, command := range commands {
			t.Run(mode+"/"+command, func(t *testing.T) {
				args := append([]string{"--format", "json", "cex", "launch"}, strings.Fields(command)...)
				_, stderr, err := credentialCLI(t, peer.URL, mode, "base earn", args...)
				if err == nil || !strings.Contains(string(stderr), "launch") {
					t.Fatalf("retired command accepted: err=%v stderr=%s", err, stderr)
				}
			})
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("retired API calls=%d", calls.Load())
	}
	// Positive controls prove the peer and retained Earn SDK wiring still work.
	for _, authenticated := range []bool{false, true} {
		t.Run(fmt.Sprintf("retained-earn/auth=%v", authenticated), func(t *testing.T) {
			before := calls.Load()
			stdout, stderr, err := runCLI(t, peer.URL, authenticated, "--format", "json", "cex", "earn", "uni", "currencies")
			if err != nil || !json.Valid(stdout) || calls.Load() != before+1 {
				t.Fatalf("Earn broken: err=%v stdout=%s stderr=%s", err, stdout, stderr)
			}
		})
	}
}

// Discovery must not advertise removed commands through help or completion.
func TestLaunchRetirementDiscovery(t *testing.T) {
	for _, args := range [][]string{{"cex", "--help"}, {"__complete", "cex", ""}} {
		stdout, stderr, err := runCLI(t, "http://127.0.0.1:1", false, args...)
		if err != nil || !strings.Contains(string(stdout), "earn") {
			t.Fatalf("discovery failed: err=%v stderr=%s", err, stderr)
		}
		for _, line := range strings.Split(string(stdout), "\n") {
			fields := strings.Fields(line)
			if len(fields) > 0 && fields[0] == "launch" {
				t.Fatalf("discovery advertises launch: %s", line)
			}
		}
	}
}
