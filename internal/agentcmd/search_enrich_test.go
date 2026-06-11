//go:build agent

package agentcmd_test

import (
	"testing"

	"github.com/gate/gate-cli/cmd"
	"github.com/gate/gate-cli/internal/cmdhint"
	"github.com/gate/gate-cli/internal/cmdindex"
)

func TestAgentSearchEnrichMatchesBriefShortcut(t *testing.T) {
	t.Parallel()
	entries := cmdindex.FilterByDomain(cmdindex.CollectLeaves(cmd.Root()), "news")
	hits := cmdindex.Search(entries, "brief", 3)
	matches := cmdhint.EnrichSearchMatches(hits)
	if len(matches) == 0 {
		t.Fatal("expected hits")
	}
	found := false
	for _, m := range matches {
		if m.IsShortcut && m.MatchSource == "shortcut" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected shortcut match in %#v", matches)
	}
}
