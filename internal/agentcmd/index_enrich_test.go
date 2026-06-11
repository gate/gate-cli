//go:build agent

package agentcmd_test

import (
	"testing"

	"github.com/gate/gate-cli/cmd"
	"github.com/gate/gate-cli/internal/cmdhint"
	"github.com/gate/gate-cli/internal/cmdindex"
)

func TestAgentIndexLeavesHaveMatchSource(t *testing.T) {
	t.Parallel()
	entries := cmdindex.FilterInfoNewsOnly(cmdindex.CollectLeaves(cmd.Root()))
	enriched := cmdhint.EnrichSearchMatches(entries)
	if len(enriched) == 0 {
		t.Fatal("expected entries")
	}
	for _, m := range enriched {
		if m.MatchSource == "" {
			t.Fatalf("missing match_source: %#v", m)
		}
	}
	foundShortcut := false
	for _, m := range enriched {
		if m.IsShortcut {
			foundShortcut = true
			break
		}
	}
	if !foundShortcut {
		t.Fatal("expected at least one shortcut in info/news index")
	}
}
