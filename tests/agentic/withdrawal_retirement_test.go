//go:build agentic_e2e

package agentic_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// Only the remote API and credentials are synthetic. The actual executable,
// command parser, help, completion and failure exit codes remain under test.
func TestWithdrawalRetirementProcesses(t *testing.T) {
	var calls atomic.Int32
	peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(peer.Close)
	for _, mode := range []string{"anonymous", "hmac", "oauth", "both"} {
		for _, args := range [][]string{
			{"create", "--currency", "USDT", "--amount", "1", "--address", "fixture-address", "--chain", "ETH", "--memo", "fixture-memo"},
			{"cancel", "--id", "9007199254740993"},
			{"create"}, {"cancel"}, {"create", "--help"}, {"cancel", "--help"},
			{"create", "--unknown"}, {"cancel", "--unknown"},
		} {
			t.Run(mode+"/"+strings.Join(args, "-"), func(t *testing.T) {
				argv := append([]string{"--format", "json", "cex", "withdrawal"}, args...)
				out, stderr, err := credentialCLI(t, peer.URL, mode, "base asset", argv...)
				if err == nil || len(out) != 0 || !strings.Contains(string(stderr), "unknown") {
					t.Fatalf("removed leaf was not rejected by parsing: err=%v out=%s stderr=%s", err, out, stderr)
				}
			})
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("removed withdrawal commands made %d API calls", calls.Load())
	}
}

// Removing the implementation must also remove discovery and cached help paths;
// otherwise an agent can keep selecting a capability the product no longer has.
func TestWithdrawalRetirementDiscovery(t *testing.T) {
	for _, args := range [][]string{{"cex", "withdrawal", "--help"}, {"__complete", "cex", "withdrawal", ""}} {
		out, stderr, err := runCLI(t, "http://127.0.0.1:1", false, args...)
		if err != nil {
			t.Fatalf("retained transfer discovery failed: %v %s", err, stderr)
		}
		if !strings.Contains(string(out), "push-order") {
			t.Fatalf("discovery lost the retained UID transfer: %s", out)
		}
		for _, line := range strings.Split(string(out), "\n") {
			fields := strings.Fields(line)
			if len(fields) > 0 && (fields[0] == "create" || fields[0] == "cancel") {
				t.Errorf("discovery advertises a removed withdrawal: %s", line)
			}
		}
	}
	for _, leaf := range []string{"create", "cancel"} {
		_, stderr, err := runCLI(t, "http://127.0.0.1:1", false, "help", "cex", "withdrawal", leaf)
		if err == nil || !strings.Contains(string(stderr), "unknown") {
			t.Errorf("cached help accepted removed leaf %s: %v %s", leaf, err, stderr)
		}
	}
}
