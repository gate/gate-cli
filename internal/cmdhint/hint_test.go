//go:build agent

package cmdhint

import (
	"bytes"
	"strings"
	"testing"
)

func TestSuggestTopLevelEarn(t *testing.T) {
	t.Parallel()
	d := SuggestTopLevel([]string{"gate-cli", "earn", "uni", "records"})
	if d == nil {
		t.Fatal("expected suggestion")
	}
	if d.Reason != "wrong_top_level" {
		t.Fatalf("reason=%q", d.Reason)
	}
	if !strings.Contains(d.Suggested, "gate-cli cex earn") {
		t.Fatalf("suggested=%q", d.Suggested)
	}
}

func TestSuggestTopLevelMarkettrend(t *testing.T) {
	t.Parallel()
	d := SuggestTopLevel([]string{"gate-cli", "markettrend", "get-kline", "--symbol", "ETH"})
	if d == nil || !strings.Contains(d.Suggested, "info markettrend") {
		t.Fatalf("got %#v", d)
	}
}

func TestPrintDiagnosticJSONLine(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	PrintDiagnostic(&buf, &Diagnostic{
		ErrorType: "COMMAND_NOT_FOUND",
		Suggested: "gate-cli cex earn uni records",
	})
	out := buf.String()
	if !strings.Contains(out, "Hint:") {
		t.Fatalf("missing human hint: %q", out)
	}
}

func TestSuggestTopLevelInfoMessage(t *testing.T) {
	t.Parallel()
	d := SuggestTopLevel([]string{"gate-cli", "markettrend"})
	if d == nil || !strings.Contains(d.Message, "gate-cli info") {
		t.Fatalf("message=%q", d.Message)
	}
}

func TestSuggestFromErrorUnknownFlagDoesNotUseWrongTopLevel(t *testing.T) {
	t.Parallel()
	d := SuggestFromError([]string{"gate-cli", "cex", "earn", "uni", "records", "--bad"}, errUnknownFlag{}, nil)
	if d == nil {
		t.Fatal("expected diagnostic")
	}
	if d.Reason == "wrong_top_level" {
		t.Fatalf("unexpected top-level redirect: %#v", d)
	}
	if d.ErrorType != "INVALID_ARGS" {
		t.Fatalf("error_type=%q", d.ErrorType)
	}
}

type errUnknownFlag struct{}

func (errUnknownFlag) Error() string { return `unknown flag: --bad` }

func TestSuggestPathCorrectionNewsExplain(t *testing.T) {
	t.Parallel()
	d := suggestPathCorrection([]string{"gate-cli", "news", "explain-market-move", "--coin", "BTC"}, `unknown command "explain-market-move" for "gate-cli news"`)
	if d == nil || !strings.Contains(d.Suggested, "news events explain-market-move") {
		t.Fatalf("got %#v", d)
	}
}

func TestSuggestPathCorrectionNewsFeedExplain(t *testing.T) {
	t.Parallel()
	d := suggestPathCorrection([]string{"gate-cli", "news", "feed", "explain-market-move", "--coin", "BTC"}, `unknown command "explain-market-move" for "gate-cli news feed"`)
	if d == nil || !strings.Contains(d.Suggested, "news events explain-market-move") {
		t.Fatalf("got %#v", d)
	}
}

func TestSuggestAuthError(t *testing.T) {
	t.Parallel()
	d := suggestAuthError([]string{"gate-cli", "cex", "spot", "account", "list"}, "API key and secret required")
	if d == nil || d.ErrorType != "AUTH_ERROR" || d.Retryable {
		t.Fatalf("got %#v", d)
	}
}

func TestSuggestFlagCorrectionNewsSymbol(t *testing.T) {
	t.Parallel()
	got := suggestFlagCorrection([]string{"gate-cli", "news", "events", "explain-market-move", "--symbol", "BTC"})
	if !strings.Contains(got, "--coin") || strings.Contains(got, "--symbol") {
		t.Fatalf("got %q", got)
	}
}

func TestSuggestPathCorrectionNewsSearch(t *testing.T) {
	t.Parallel()
	d := suggestPathCorrection([]string{"gate-cli", "news", "search", "--coin", "BTC"}, `unknown command "search" for "gate-cli news"`)
	if d == nil || !strings.Contains(d.Suggested, "news feed search-news") {
		t.Fatalf("got %#v", d)
	}
}

func TestSuggestTopLevelCoinanalysis(t *testing.T) {
	t.Parallel()
	d := SuggestTopLevel([]string{"gate-cli", "coinanalysis", "--symbol", "BTC"})
	if d == nil || !strings.Contains(d.Suggested, "info coin get-coin-info") {
		t.Fatalf("got %#v", d)
	}
}

func TestAgentLeavesCount(t *testing.T) {
	t.Parallel()
	if len(AgentLeaves) != 31 {
		t.Fatalf("expected 31 info/news leaves, got %d", len(AgentLeaves))
	}
	for _, leaf := range AgentLeaves {
		if leaf.RequiredPrefix != "info" && leaf.RequiredPrefix != "news" {
			t.Fatalf("agent-leaves must be info/news only, got %q", leaf.RequiredPrefix)
		}
	}
}
