package cmd

import (
	"testing"

	"github.com/gate/gate-cli/tests/contracts"
)

// Normal wire behavior is the compatibility gate, not merely an exit status.
func TestCompatibleTradingHandlers(t *testing.T) {
	for _, tc := range contracts.CompatibilityScenarios(t) {
		t.Run(tc.ID, func(t *testing.T) {
			auth := "hmac"
			if tc.Case.Public {
				auth = "anonymous"
			}
			peer := contracts.NewPeer(t, tc.Case, auth, "success")
			out, err := invokeContractHandler(t, tc.Case, peer.URL, t.Context())
			peer.AssertCalls(len(tc.Case.Exchanges))
			if err != nil || !contracts.EqualJSON(out, tc.Case.Want) {
				t.Errorf("compatible behavior changed: %v output=%s", err, out)
			}
		})
	}
}

func TestRejectTradingBoundaries(t *testing.T) {
	for _, tc := range contracts.RejectionScenarios(t) {
		t.Run(tc.ID, func(t *testing.T) {
			peer := contracts.NewPeer(t, tc.Case, "hmac", "success")
			out, err := invokeContractHandler(t, tc.Case, peer.URL, t.Context())
			if err == nil || len(out) != 0 || peer.Calls() >= len(tc.Case.Exchanges) {
				t.Errorf("invalid intent reached terminal write: err=%v calls=%d", err, peer.Calls())
			}
		})
	}
}
