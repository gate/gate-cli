package futures

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// positionCmd returns the `position` subcommand registered on futures.Cmd.
func findPositionCmd(t *testing.T) *cobra.Command {
	t.Helper()
	for _, c := range Cmd.Commands() {
		if c.Name() == "position" {
			return c
		}
	}
	t.Fatal("position subcommand not found on futures.Cmd")
	return nil
}

// subcommandByName walks a cobra parent and returns the matching subcommand.
func subcommandByName(parent *cobra.Command, name string) *cobra.Command {
	for _, c := range parent.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

func TestFuturesPositionFullSubcommandTree(t *testing.T) {
	// All commands expected after SDK v7.2.71 sync (dual rename + single-mode additions).
	want := map[string]bool{
		// Shared / read-only
		"list":           false,
		"list-timerange": false,
		"leverage":       false,
		"close-history":  false,
		"liquidates":     false,
		"adl":            false,
		// Dual-mode (renamed in route-β)
		"get-dual":               false,
		"update-dual-margin":     false,
		"update-dual-leverage":   false,
		"update-dual-cross-mode": false,
		"update-dual-risk-limit": false,
		// Contract-mode (unchanged)
		"update-contract-leverage": false,
		// One-way / single-mode (newly added)
		"get":               false,
		"update-margin":     false,
		"update-leverage":   false,
		"update-cross-mode": false,
		"update-risk-limit": false,
	}

	pos := findPositionCmd(t)
	for _, sub := range pos.Commands() {
		if _, ok := want[sub.Name()]; ok {
			want[sub.Name()] = true
		}
	}
	for name, ok := range want {
		assert.True(t, ok, "position should expose %q", name)
	}
}

func TestFuturesPositionDualCommandsRequireFlags(t *testing.T) {
	pos := findPositionCmd(t)

	// get-dual: --contract required
	getDual := subcommandByName(pos, "get-dual")
	require.NotNil(t, getDual, "get-dual should be registered")
	require.NotNil(t, getDual.Flag("contract"))
	assert.NotEmpty(t,
		getDual.Flag("contract").Annotations[cobraBashCompOneRequiredFlag],
		"get-dual --contract should be required")

	// update-dual-margin: --contract, --change, --dual-side required
	updDualMargin := subcommandByName(pos, "update-dual-margin")
	require.NotNil(t, updDualMargin)
	for _, name := range []string{"contract", "change"} {
		f := updDualMargin.Flag(name)
		require.NotNil(t, f, "update-dual-margin should have --%s", name)
		assert.NotEmpty(t, f.Annotations[cobraBashCompOneRequiredFlag],
			"update-dual-margin --%s should be required", name)
	}
	// dual-side exists but is not required (only required when in dual mode)
	require.NotNil(t, updDualMargin.Flag("dual-side"))

	// update-dual-leverage: --contract, --leverage required
	updDualLev := subcommandByName(pos, "update-dual-leverage")
	require.NotNil(t, updDualLev)
	for _, name := range []string{"contract", "leverage"} {
		f := updDualLev.Flag(name)
		require.NotNil(t, f)
		assert.NotEmpty(t, f.Annotations[cobraBashCompOneRequiredFlag])
	}

	// update-dual-cross-mode: --contract, --mode required
	updDualCross := subcommandByName(pos, "update-dual-cross-mode")
	require.NotNil(t, updDualCross)
	for _, name := range []string{"contract", "mode"} {
		f := updDualCross.Flag(name)
		require.NotNil(t, f)
		assert.NotEmpty(t, f.Annotations[cobraBashCompOneRequiredFlag])
	}

	// update-dual-risk-limit: --contract, --risk-limit required
	updDualRisk := subcommandByName(pos, "update-dual-risk-limit")
	require.NotNil(t, updDualRisk)
	for _, name := range []string{"contract", "risk-limit"} {
		f := updDualRisk.Flag(name)
		require.NotNil(t, f)
		assert.NotEmpty(t, f.Annotations[cobraBashCompOneRequiredFlag])
	}
}

func TestFuturesPositionSingleCommandsRequireFlags(t *testing.T) {
	pos := findPositionCmd(t)

	// get (single-mode): --contract required
	getSingle := subcommandByName(pos, "get")
	require.NotNil(t, getSingle, "get (single-mode) should be registered")
	f := getSingle.Flag("contract")
	require.NotNil(t, f)
	assert.NotEmpty(t, f.Annotations[cobraBashCompOneRequiredFlag])

	// update-margin: --contract, --change required
	updMargin := subcommandByName(pos, "update-margin")
	require.NotNil(t, updMargin)
	for _, name := range []string{"contract", "change"} {
		f := updMargin.Flag(name)
		require.NotNil(t, f)
		assert.NotEmpty(t, f.Annotations[cobraBashCompOneRequiredFlag])
	}

	// update-leverage: --contract, --leverage required; --cross-leverage-limit optional
	updLev := subcommandByName(pos, "update-leverage")
	require.NotNil(t, updLev)
	for _, name := range []string{"contract", "leverage"} {
		f := updLev.Flag(name)
		require.NotNil(t, f)
		assert.NotEmpty(t, f.Annotations[cobraBashCompOneRequiredFlag])
	}
	crossLim := updLev.Flag("cross-leverage-limit")
	require.NotNil(t, crossLim, "update-leverage should expose --cross-leverage-limit")
	assert.Empty(t, crossLim.Annotations[cobraBashCompOneRequiredFlag],
		"--cross-leverage-limit must remain optional")

	// update-cross-mode: --contract, --mode required
	updCross := subcommandByName(pos, "update-cross-mode")
	require.NotNil(t, updCross)
	for _, name := range []string{"contract", "mode"} {
		f := updCross.Flag(name)
		require.NotNil(t, f)
		assert.NotEmpty(t, f.Annotations[cobraBashCompOneRequiredFlag])
	}

	// update-risk-limit: --contract, --risk-limit required
	updRisk := subcommandByName(pos, "update-risk-limit")
	require.NotNil(t, updRisk)
	for _, name := range []string{"contract", "risk-limit"} {
		f := updRisk.Flag(name)
		require.NotNil(t, f)
		assert.NotEmpty(t, f.Annotations[cobraBashCompOneRequiredFlag])
	}
}

// Ensures route-β naming has been applied: the unprefixed `update-leverage`
// now belongs to single-mode (UpdatePositionLeverage), NOT the old dual-mode
// alias. We detect this via the Short description because RunE values are
// unexported.
func TestFuturesPositionRouteBetaRenaming(t *testing.T) {
	pos := findPositionCmd(t)

	// get is single-mode
	getCmd := subcommandByName(pos, "get")
	require.NotNil(t, getCmd)
	assert.Contains(t, getCmd.Short, "one-way", "get should be single/one-way mode")

	// get-dual is dual-mode
	getDual := subcommandByName(pos, "get-dual")
	require.NotNil(t, getDual)
	assert.Contains(t, getDual.Short, "dual-mode", "get-dual should be dual/hedge mode")

	// update-leverage is single-mode
	updLev := subcommandByName(pos, "update-leverage")
	require.NotNil(t, updLev)
	assert.Contains(t, updLev.Short, "one-way")

	// update-dual-leverage is dual-mode
	updDualLev := subcommandByName(pos, "update-dual-leverage")
	require.NotNil(t, updDualLev)
	assert.Contains(t, updDualLev.Short, "dual-mode")
}
