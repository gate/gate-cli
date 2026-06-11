package intelcmd

import (
	"testing"

	"github.com/gate/gate-cli/internal/intelfacade"
)

func TestCompactToolListPaths(t *testing.T) {
	t.Parallel()
	got := compactToolList("news", []intelfacade.ToolSummary{
		{Name: "news_feed_search_news"},
	})
	if len(got) != 1 || got[0]["path"] != "news feed search-news" {
		t.Fatalf("got=%v", got)
	}
	if _, ok := got[0]["name"]; ok {
		t.Fatalf("unexpected MCP wire name in compact list: %v", got)
	}
}

func TestToolNameToCLIPath(t *testing.T) {
	t.Parallel()
	if p := toolNameToCLIPath("info", "info_coin_get_coin_info"); p != "coin get-coin-info" {
		t.Fatalf("got %q", p)
	}
}
