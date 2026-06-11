//go:build agent

package agentcmd_test

import (
	"testing"

	"github.com/gate/gate-cli/cmd"
	"github.com/gate/gate-cli/internal/agentcmd"
	"github.com/gate/gate-cli/internal/cmdindex"
)

func TestDiscoveryEntriesAgentScope(t *testing.T) {
	t.Setenv("GATE_CLI_AGENT", "1")
	got := agentcmd.DiscoveryEntries(cmd.Root(), "")
	for _, e := range got {
		parts := cmdindex.FilterInfoNewsOnly([]cmdindex.Entry{e})
		if len(parts) != 1 {
			t.Fatalf("entry outside info/news: %s", e.Path)
		}
	}
}
