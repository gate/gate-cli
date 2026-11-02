//go:build agentic_e2e

package agentic_test

import (
	"bytes"
	"testing"

	"github.com/gate/gate-cli/tests/contracts"
)

// Only the exchange HTTP peer and credentials are fixtures; argv, the executable,
// real SDK/transport, output and exit code remain inside this acceptance chain.
func TestCompatibleTradingExecutable(t *testing.T) {
	for _, tc := range contracts.CompatibilityScenarios(t) {
		for _, auth := range []string{"hmac", "oauth"} {
			t.Run(tc.ID+"/"+auth, func(t *testing.T) {
				t.Parallel()
				wire := auth
				if tc.Case.Public {
					wire = "anonymous"
				}
				peer := contracts.NewPeer(t, tc.Case, wire, "success")
				home := cexContractHome(t, peer.URL, auth)
				out, stderr, err := futuresConfirmedInvoke(t, cliBinary, home, peer.URL, auth, append([]string{"--format", "json"}, tc.Case.Args...)...)
				peer.AssertCalls(len(tc.Case.Exchanges))
				if err != nil || len(stderr) != 0 || !contracts.EqualJSON(out, tc.Case.Want) {
					t.Errorf("compatible command changed: %v %s %s", err, out, stderr)
				}
			})
		}
	}
}

func TestRejectTradingBoundariesExecutable(t *testing.T) {
	for _, tc := range contracts.RejectionScenarios(t) {
		for _, auth := range []string{"hmac", "oauth"} {
			t.Run(tc.ID+"/"+auth, func(t *testing.T) {
				t.Parallel()
				peer := contracts.NewPeer(t, tc.Case, auth, "success")
				home := cexContractHome(t, peer.URL, auth)
				out, _, err := futuresConfirmedInvoke(t, cliBinary, home, peer.URL, auth, append([]string{"--format", "json"}, tc.Case.Args...)...)
				if err == nil || len(out) != 0 || peer.Calls() >= len(tc.Case.Exchanges) {
					t.Errorf("invalid batch reached terminal write: err=%v calls=%d", err, peer.Calls())
				}
			})
		}
	}
}

func TestCommandPolicyExecutable(t *testing.T) {
	for _, tc := range contracts.ArgumentScenarios(t) {
		for _, auth := range []string{"hmac", "oauth"} {
			t.Run(tc.ID+"/"+auth, func(t *testing.T) {
				peer := contracts.NewPeer(t, tc.Case, auth, "success")
				home := cexContractHome(t, peer.URL, auth)
				out, stderr, err := futuresConfirmedInvoke(t, cliBinary, home, peer.URL, auth, append([]string{"--format", "json"}, tc.Case.Args...)...)
				if tc.Reject {
					peer.AssertCalls(0)
					if err == nil || len(out) != 0 {
						t.Errorf("invalid argv accepted: %v", err)
					}
					return
				}
				peer.AssertCalls(len(tc.Case.Exchanges))
				if err != nil || len(stderr) != 0 || !contracts.EqualJSON(out, tc.Case.Want) {
					t.Errorf("boolean argv changed: %v %s %s", err, out, stderr)
				}
			})
		}
	}
}

// A rejected extra positional must never echo supported credential flags through
// agent diagnostics. The peer proves rejection happens before any HTTP request.
func TestCommandPolicyCredentialPrivacy(t *testing.T) {
	for _, equal := range []bool{false, true} {
		t.Run(map[bool]string{false: "space", true: "equals"}[equal], func(t *testing.T) {
			tc := contracts.ArgumentScenarios(t)[0].Case
			peer := contracts.NewPeer(t, tc, "hmac", "success")
			args := []string{"--api-key", "key-canary-cli", "--api-secret", "secret-canary-cli"}
			if equal {
				args = []string{"--api-key=key-canary-cli", "--api-secret=secret-canary-cli"}
			}
			args = append(args, "--format", "json", "cex", "spot", "account", "list", "__unexpected__")
			out, stderr, err := runCLI(t, peer.URL, false, args...)
			peer.AssertCalls(0)
			if err == nil || len(out) != 0 {
				t.Errorf("invalid argv accepted: %v", err)
			}
			for _, canary := range []string{"key-canary-cli", "secret-canary-cli"} {
				if bytes.Contains(out, []byte(canary)) || bytes.Contains(stderr, []byte(canary)) {
					t.Errorf("credential reached command output")
				}
			}
		})
	}
}
