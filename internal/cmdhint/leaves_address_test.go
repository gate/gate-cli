//go:build agent

package cmdhint

import "testing"

func TestMatchAgentLeavesTokenRiskByAddress(t *testing.T) {
	t.Parallel()
	got := MatchAgentLeaves("token security contract address eth", 3)
	if len(got) == 0 {
		t.Fatal("expected matches")
	}
	found := false
	for _, leaf := range got {
		if leaf.Intent == "info_token_risk_by_address" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected info_token_risk_by_address in %#v", got)
	}
}

func TestMatchAgentLeavesTokenOnchainByAddress(t *testing.T) {
	t.Parallel()
	got := MatchAgentLeaves("token onchain holder address contract", 3)
	if len(got) == 0 {
		t.Fatal("expected matches")
	}
	found := false
	for _, leaf := range got {
		if leaf.Intent == "info_token_onchain_by_address" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected info_token_onchain_by_address in %#v", got)
	}
}
