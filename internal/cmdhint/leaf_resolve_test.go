//go:build agent

package cmdhint

import "testing"

func TestResolveAgentIntentCuratedBeforeMCP(t *testing.T) {
	t.Parallel()
	got := ResolveAgentIntent("BTC news brief", 3, 3, "")
	if len(got) == 0 {
		t.Fatal("expected matches")
	}
	if got[0].MatchSource != "curated" {
		t.Fatalf("first source=%q intent=%q", got[0].MatchSource, got[0].Intent)
	}
	if got[0].Intent != "news_brief" {
		t.Fatalf("expected news_brief, got intent=%q cmd=%q", got[0].Intent, got[0].Command)
	}
}

func TestResolveAgentIntentFallsBackToMCPCatalog(t *testing.T) {
	t.Parallel()
	got := ResolveAgentIntent("info platformmetrics get-stablecoin-info", 3, 3, "")
	if len(got) == 0 {
		t.Fatal("expected mcp_catalog match")
	}
	foundMCP := false
	for _, r := range got {
		if r.MatchSource == "mcp_catalog" {
			foundMCP = true
		}
	}
	if !foundMCP {
		t.Fatalf("expected mcp_catalog source in %#v", got)
	}
}
