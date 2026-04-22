package launch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLaunchCommandStructure(t *testing.T) {
	subCmds := make(map[string]bool)
	for _, c := range Cmd.Commands() {
		subCmds[c.Name()] = true
	}

	expected := []string{
		"projects", "pledge", "redeem", "pledge-records", "reward-records",
		// Added alongside SDK v7.2.71 sync:
		"candy-drop", "hodler",
	}
	for _, name := range expected {
		assert.True(t, subCmds[name], "missing subcommand: %s", name)
	}
}

func TestPledgeRequiresJSON(t *testing.T) {
	for _, c := range Cmd.Commands() {
		if c.Name() == "pledge" {
			f := c.Flag("json")
			require.NotNil(t, f, "pledge should have --json flag")
			return
		}
	}
	t.Fatal("pledge subcommand not found")
}

func TestRedeemRequiresJSON(t *testing.T) {
	for _, c := range Cmd.Commands() {
		if c.Name() == "redeem" {
			f := c.Flag("json")
			require.NotNil(t, f, "redeem should have --json flag")
			return
		}
	}
	t.Fatal("redeem subcommand not found")
}
