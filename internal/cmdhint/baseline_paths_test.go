//go:build agent

package cmdhint

import (
	"strings"
	"testing"
)

func TestMCPToolToCLICommand(t *testing.T) {
	t.Parallel()
	if got := MCPToolToCLICommand("news_feed_search_news"); got != "gate-cli news feed search-news --format json" {
		t.Fatalf("got %q", got)
	}
	if got := MCPToolToCLICommand("info_markettrend_get_kline"); got != "gate-cli info markettrend get-kline --format json" {
		t.Fatalf("got %q", got)
	}
	if got := MCPToolToCLICommand("info_platformmetrics_get_chain_activity"); got != "gate-cli info platformmetrics get-chain-activity --format json" {
		t.Fatalf("got %q", got)
	}
}

func TestBaselineToolLeafIntentUsesCLIPath(t *testing.T) {
	t.Parallel()
	leaf := BaselineToolLeaf("info_coin_get_coin_info")
	if leaf.Intent != "coin-get-coin-info" {
		t.Fatalf("intent=%q", leaf.Intent)
	}
	if strings.Contains(leaf.Intent, "_") {
		t.Fatalf("intent must not contain MCP wire underscores: %q", leaf.Intent)
	}
	if !strings.Contains(leaf.Command, "info coin get-coin-info") {
		t.Fatalf("command=%q", leaf.Command)
	}
}

func TestBaselineMCPCatalogCount(t *testing.T) {
	t.Parallel()
	if len(BaselineMCPCatalog()) != 46 {
		t.Fatalf("want 46 baseline tools, got %d", len(BaselineMCPCatalog()))
	}
}
