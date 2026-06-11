//go:build agent

package cmdhint

import "testing"

func TestEnrichDiagnosticSkipsHelpCrawlBlock(t *testing.T) {
	t.Parallel()
	d := &Diagnostic{
		Blocked:             true,
		Reason:              "HELP_CRAWL_FORBIDDEN",
		SuggestedNextAction: "use gate-cli agent-resolve --query <intent>",
		Retryable:           false,
	}
	EnrichDiagnosticWithAgentLeaf(d, []string{"gate-cli", "info", "+coin-overview", "-h"})
	if d.Suggested != "" {
		t.Fatalf("must not set suggested on help block, got %q", d.Suggested)
	}
	if d.SuggestedNextAction != "use gate-cli agent-resolve --query <intent>" {
		t.Fatalf("must not overwrite next action, got %q", d.SuggestedNextAction)
	}
}

func TestEnrichDiagnosticSkipsWrongTopLevel(t *testing.T) {
	t.Parallel()
	d := &Diagnostic{
		Blocked:             true,
		Reason:              "wrong_top_level",
		Suggested:           "gate-cli cex earn uni",
		SuggestedNextAction: "use the suggested command prefix; run gate-cli agent-leaves --format json",
	}
	EnrichDiagnosticWithAgentLeaf(d, []string{"gate-cli", "earn", "uni"})
	if d.SuggestedNextAction != "use the suggested command prefix; run gate-cli agent-leaves --format json" {
		t.Fatalf("must not overwrite blocked top-level next action, got %q", d.SuggestedNextAction)
	}
}

func TestEnrichDiagnosticFillsEmptyNextAction(t *testing.T) {
	t.Parallel()
	d := &Diagnostic{
		ErrorType: "COMMAND_NOT_FOUND",
		Message:   "unknown command",
	}
	EnrichDiagnosticWithAgentLeaf(d, []string{"gate-cli", "news", "feed", "search-news"})
	if d.Suggested == "" || d.SuggestedNextAction == "" {
		t.Fatalf("expected enrich, got %#v", d)
	}
}
