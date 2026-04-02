package margin

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarginCommandStructure(t *testing.T) {
	// Verify all subcommand groups are registered on Cmd
	subCmds := make(map[string]bool)
	for _, c := range Cmd.Commands() {
		subCmds[c.Name()] = true
	}

	expected := []string{"account", "auto-repay", "tier", "leverage", "cross", "uni"}
	for _, name := range expected {
		assert.True(t, subCmds[name], "missing subcommand group: %s", name)
	}
}

func TestAccountSubcommands(t *testing.T) {
	subCmds := make(map[string]bool)
	for _, c := range accountCmd.Commands() {
		subCmds[c.Name()] = true
	}

	expected := []string{"list", "book", "funding", "transferable", "user-info"}
	for _, name := range expected {
		assert.True(t, subCmds[name], "missing account subcommand: %s", name)
	}
}

func TestAutoRepaySubcommands(t *testing.T) {
	subCmds := make(map[string]bool)
	for _, c := range autoRepayCmd.Commands() {
		subCmds[c.Name()] = true
	}

	expected := []string{"get", "set"}
	for _, name := range expected {
		assert.True(t, subCmds[name], "missing auto-repay subcommand: %s", name)
	}
}

func TestTierSubcommands(t *testing.T) {
	subCmds := make(map[string]bool)
	for _, c := range tierCmd.Commands() {
		subCmds[c.Name()] = true
	}

	expected := []string{"user", "market"}
	for _, name := range expected {
		assert.True(t, subCmds[name], "missing tier subcommand: %s", name)
	}
}

func TestUniSubcommands(t *testing.T) {
	subCmds := make(map[string]bool)
	for _, c := range uniCmd.Commands() {
		subCmds[c.Name()] = true
	}

	expected := []string{"pairs", "pair", "estimate-rate", "loans", "lend",
		"loan-records", "interest-records", "borrowable"}
	for _, name := range expected {
		assert.True(t, subCmds[name], "missing uni subcommand: %s", name)
	}
}

func TestAutoRepaySetRequiresStatusFlag(t *testing.T) {
	// Find the "set" subcommand under auto-repay
	var setCmd *cobra.Command
	for _, c := range autoRepayCmd.Commands() {
		if c.Name() == "set" {
			setCmd = c
			break
		}
	}
	require.NotNil(t, setCmd)

	f := setCmd.Flag("status")
	require.NotNil(t, f, "set command should have --status flag")
}

func TestLeverageSetRequiredFlags(t *testing.T) {
	var setCmd *cobra.Command
	for _, c := range leverageCmd.Commands() {
		if c.Name() == "set" {
			setCmd = c
			break
		}
	}
	require.NotNil(t, setCmd)

	pairFlag := setCmd.Flag("pair")
	require.NotNil(t, pairFlag, "set command should have --pair flag")

	levFlag := setCmd.Flag("leverage")
	require.NotNil(t, levFlag, "set command should have --leverage flag")
}
