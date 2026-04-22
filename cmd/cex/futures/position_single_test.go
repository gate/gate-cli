package futures

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestRoot builds a minimal cobra root mirroring cmd/root.go's persistent
// flags so runXxx handlers can invoke cmdutil.GetClient/GetPrinter/GetSettle.
// Environment is scrubbed so no real credentials leak in.
func newTestRoot(t *testing.T) *cobra.Command {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GATE_API_KEY", "")
	t.Setenv("GATE_API_SECRET", "")
	root := &cobra.Command{Use: "gate-cli"}
	root.PersistentFlags().String("format", "json", "")
	root.PersistentFlags().String("profile", "default", "")
	root.PersistentFlags().Bool("debug", false, "")
	root.PersistentFlags().Bool("verbose", false, "")
	root.PersistentFlags().String("api-key", "", "")
	root.PersistentFlags().String("api-secret", "", "")
	return root
}

// singleModeLeaf builds a standalone child command with the default flag set
// expected by the position single-mode runners (contract is always present).
func singleModeLeaf(use string, flagSetup func(*cobra.Command)) *cobra.Command {
	cmd := &cobra.Command{Use: use}
	cmd.Flags().String("contract", "BTC_USDT", "")
	cmd.Flags().String("settle", "usdt", "")
	if flagSetup != nil {
		flagSetup(cmd)
	}
	return cmd
}

func TestRunFuturesPositionGetSingle_RequiresAuth(t *testing.T) {
	root := newTestRoot(t)
	cmd := singleModeLeaf("get", nil)
	root.AddCommand(cmd)

	err := runFuturesPositionGetSingle(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

func TestRunFuturesUpdateSinglePositionMargin_RequiresAuth(t *testing.T) {
	root := newTestRoot(t)
	cmd := singleModeLeaf("update-margin", func(c *cobra.Command) {
		c.Flags().String("change", "10", "")
	})
	root.AddCommand(cmd)

	err := runFuturesUpdateSinglePositionMargin(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

func TestRunFuturesUpdateSinglePositionLeverage_RequiresAuth(t *testing.T) {
	root := newTestRoot(t)
	cmd := singleModeLeaf("update-leverage", func(c *cobra.Command) {
		c.Flags().String("leverage", "5", "")
		c.Flags().String("cross-leverage-limit", "", "")
	})
	root.AddCommand(cmd)

	err := runFuturesUpdateSinglePositionLeverage(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

func TestRunFuturesUpdateSinglePositionCrossMode_RequiresAuth(t *testing.T) {
	root := newTestRoot(t)
	cmd := singleModeLeaf("update-cross-mode", func(c *cobra.Command) {
		c.Flags().String("mode", "CROSS", "")
	})
	root.AddCommand(cmd)

	err := runFuturesUpdateSinglePositionCrossMode(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

func TestRunFuturesUpdateSinglePositionRiskLimit_RequiresAuth(t *testing.T) {
	root := newTestRoot(t)
	cmd := singleModeLeaf("update-risk-limit", func(c *cobra.Command) {
		c.Flags().String("risk-limit", "1000000", "")
	})
	root.AddCommand(cmd)

	err := runFuturesUpdateSinglePositionRiskLimit(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

// Dual-mode handlers follow the same RequireAuth pattern; covering them here
// protects against regressions in the shared auth gate.

func TestRunFuturesPositionGetDual_RequiresAuth(t *testing.T) {
	root := newTestRoot(t)
	cmd := singleModeLeaf("get-dual", nil)
	root.AddCommand(cmd)

	err := runFuturesPositionGetDual(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

func TestRunFuturesUpdateDualPositionLeverage_RequiresAuth(t *testing.T) {
	root := newTestRoot(t)
	cmd := singleModeLeaf("update-dual-leverage", func(c *cobra.Command) {
		c.Flags().String("leverage", "5", "")
	})
	root.AddCommand(cmd)

	err := runFuturesUpdateDualPositionLeverage(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}
