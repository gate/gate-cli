//go:build agent

package agentcmd_test

import (
	"os"
	"strings"
	"testing"

	"github.com/gate/gate-cli/cmd"
	"github.com/gate/gate-cli/internal/cmdhint"
	"github.com/gate/gate-cli/internal/toolargs"
)

// PRD adversarial cases: wrong top-level, help crawl, wrong news path, oversized kline.

func TestAdversarialWrongTopLevelEarn(t *testing.T) {
	t.Parallel()
	d := cmdhint.SuggestTopLevel([]string{"gate-cli", "earn", "simple-earn", "check"})
	if d == nil || d.Reason != "wrong_top_level" {
		t.Fatalf("got %#v", d)
	}
	if !strings.Contains(d.Suggested, "cex earn") {
		t.Fatalf("suggested=%q", d.Suggested)
	}
}

func TestAdversarialWrongTopLevelMarkettrend(t *testing.T) {
	t.Parallel()
	d := cmdhint.SuggestTopLevel([]string{"gate-cli", "markettrend"})
	if d == nil || !strings.Contains(d.Suggested, "info markettrend") {
		t.Fatalf("got %#v", d)
	}
}

func TestAdversarialNewsExplainWrongGroup(t *testing.T) {
	t.Parallel()
	d := cmdhint.SuggestFromError([]string{"gate-cli", "news", "feed", "explain-market-move", "--coin", "EDX"}, errAdversarialUnknownCmd{}, cmd.Root())
	if d == nil || d.Suggested == "" {
		t.Fatal("expected path correction")
	}
	if !strings.Contains(d.Suggested, "news events explain-market-move") {
		t.Fatalf("suggested=%q", d.Suggested)
	}
}

func TestAdversarialParentHelpBlockedInAgentMode(t *testing.T) {
	t.Setenv("GATE_CLI_AGENT", "1")
	t.Cleanup(func() { _ = os.Unsetenv("GATE_CLI_AGENT") })
	if !cmdhint.ShouldBlockParentHelp([]string{"gate-cli", "news", "--help"}) {
		t.Fatal("expected parent help block")
	}
	if cmdhint.ShouldBlockParentHelp([]string{"gate-cli", "news", "feed", "search-news", "--help"}) {
		t.Fatal("leaf help should be allowed")
	}
}

func TestAdversarialKlineSize5000Rejected(t *testing.T) {
	t.Parallel()
	err := toolargs.ValidateForTool("info_markettrend_get_kline", map[string]interface{}{"size": 5000})
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("err=%v", err)
	}
}

type errAdversarialUnknownCmd struct{}

func (errAdversarialUnknownCmd) Error() string {
	return `unknown command "explain-market-move" for "gate-cli news feed"`
}

type errAdversarialEventDetail struct{}

func (errAdversarialEventDetail) Error() string {
	return `unknown command "get-event-detail" for "gate-cli news"`
}

func TestAdversarialRootHelpBlocked(t *testing.T) {
	t.Setenv("GATE_CLI_AGENT", "1")
	t.Cleanup(func() { _ = os.Unsetenv("GATE_CLI_AGENT") })
	if !cmdhint.ShouldBlockParentHelp([]string{"gate-cli", "--help"}) {
		t.Fatal("expected root --help block")
	}
}

func TestAdversarialOnchainAddressRequired(t *testing.T) {
	t.Parallel()
	if err := toolargs.ValidateForTool("info_onchain_get_address_info", map[string]interface{}{}); err == nil {
		t.Fatal("expected address required")
	}
}

func TestAdversarialMacroIndicatorRequired(t *testing.T) {
	t.Parallel()
	if err := toolargs.ValidateForTool("info_macro_get_macro_indicator", map[string]interface{}{}); err == nil {
		t.Fatal("expected indicator required")
	}
}

func TestAdversarialPathNewsEventDetail(t *testing.T) {
	t.Parallel()
	d := cmdhint.SuggestFromError([]string{"gate-cli", "news", "get-event-detail", "--event-id", "x"}, errAdversarialEventDetail{}, cmd.Root())
	if d == nil || !strings.Contains(d.Suggested, "news events get-event-detail") {
		t.Fatalf("got %#v", d)
	}
}

func TestAdversarialCoinInfoRequiresSymbol(t *testing.T) {
	t.Parallel()
	err := toolargs.ValidateForTool("info_coin_get_coin_info", map[string]interface{}{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestAdversarialAgentValidateDiscoveryOK(t *testing.T) {
	report := cmdhint.ValidateAgentDiscovery(cmd.Root())
	if !report.OK {
		t.Fatalf("discovery validation failed: %+v", report)
	}
}
