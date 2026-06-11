//go:build agent

package cmdhint

import "testing"

func TestShouldBlockParentHelp(t *testing.T) {
	t.Parallel()
	if !ShouldBlockParentHelp([]string{"gate-cli", "cex", "--help"}) {
		t.Fatal("expected block on cex --help")
	}
	if ShouldBlockParentHelp([]string{"gate-cli", "cex", "earn", "uni", "records", "--help"}) {
		t.Fatal("expected allow on leaf --help")
	}
	if !ShouldBlockParentHelp([]string{"gate-cli", "--help"}) {
		t.Fatal("expected block on root --help")
	}
	if ShouldBlockParentHelp([]string{"gate-cli", "news", "feed", "search-news", "--help"}) {
		t.Fatal("expected allow on info/news 3-segment leaf --help")
	}
	if !ShouldBlockParentHelp([]string{"gate-cli", "cex", "spot", "market", "--help"}) {
		t.Fatal("expected block on cex 3-segment group --help")
	}
	if ShouldBlockParentHelp([]string{"gate-cli", "info", "+coin-overview", "--help"}) {
		t.Fatal("expected allow on info/news shortcut --help")
	}
	if ShouldBlockParentHelp([]string{"gate-cli", "news", "list", "--help"}) {
		t.Fatal("expected allow on news list --help")
	}
	if ShouldBlockParentHelp([]string{"gate-cli", "info", "describe", "-h"}) {
		t.Fatal("expected allow on info describe --help")
	}
}
