//go:build agent

package cmdhint

import (
	"testing"

	"github.com/gate/gate-cli/internal/cmdindex"
)

func TestEnrichSearchMatchesShortcut(t *testing.T) {
	t.Parallel()
	got := EnrichSearchMatches([]cmdindex.Entry{{Path: "news +brief", Short: "brief"}})
	if len(got) != 1 || got[0].MatchSource != "shortcut" || !got[0].IsShortcut {
		t.Fatalf("got %#v", got)
	}
}

func TestEnrichSearchMatchesCurated(t *testing.T) {
	t.Parallel()
	got := EnrichSearchMatches([]cmdindex.Entry{{Path: "info markettrend get-kline"}})
	if len(got) != 1 || got[0].MatchSource != "curated" || got[0].CuratedIntent != "market_kline" {
		t.Fatalf("got %#v", got)
	}
}
