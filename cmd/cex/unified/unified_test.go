package unified

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnifiedCommandStructure(t *testing.T) {
	// Verify all 6 subcommand groups are registered
	subs := Cmd.Commands()
	names := make(map[string]bool, len(subs))
	for _, c := range subs {
		names[c.Use] = true
	}

	expected := []string{"account", "mode", "query", "loan", "risk", "config"}
	for _, name := range expected {
		assert.True(t, names[name], "missing subcommand group: %s", name)
	}
}

func TestAccountSubcommands(t *testing.T) {
	subs := accountCmd.Commands()
	names := make([]string, len(subs))
	for i, c := range subs {
		names[i] = c.Use
	}
	require.Contains(t, names, "get")
	require.Contains(t, names, "currencies")
}

func TestModeSubcommands(t *testing.T) {
	subs := modeCmd.Commands()
	names := make([]string, len(subs))
	for i, c := range subs {
		names[i] = c.Use
	}
	require.Contains(t, names, "get")
	require.Contains(t, names, "set")
}

func TestQuerySubcommands(t *testing.T) {
	subs := queryCmd.Commands()
	names := make([]string, len(subs))
	for i, c := range subs {
		names[i] = c.Use
	}
	require.Contains(t, names, "borrowable")
	require.Contains(t, names, "borrowable-list")
	require.Contains(t, names, "transferable")
	require.Contains(t, names, "transferables")
	require.Contains(t, names, "estimate-rate")
}

func TestLoanSubcommands(t *testing.T) {
	subs := loanCmd.Commands()
	names := make([]string, len(subs))
	for i, c := range subs {
		names[i] = c.Use
	}
	require.Contains(t, names, "list")
	require.Contains(t, names, "create")
	require.Contains(t, names, "records")
	require.Contains(t, names, "interest")
	require.Contains(t, names, "history-rate")
}

func TestRiskSubcommands(t *testing.T) {
	subs := riskCmd.Commands()
	names := make([]string, len(subs))
	for i, c := range subs {
		names[i] = c.Use
	}
	require.Contains(t, names, "units")
	require.Contains(t, names, "calculate")
}

func TestConfigSubcommands(t *testing.T) {
	subs := configCmd.Commands()
	names := make([]string, len(subs))
	for i, c := range subs {
		names[i] = c.Use
	}
	require.Contains(t, names, "collateral")
	require.Contains(t, names, "discount-tiers")
	require.Contains(t, names, "loan-tiers")
	require.Contains(t, names, "leverage-config")
	require.Contains(t, names, "leverage-get")
	require.Contains(t, names, "leverage-set")
}
