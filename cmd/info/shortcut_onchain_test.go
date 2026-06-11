package info

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gate/gate-cli/internal/output"
)

func runOnchainShortcut(t *testing.T, cmd *cobra.Command, setFlags func(*cobra.Command)) (stdout, stderr string, err error) {
	t.Helper()
	oldFactory, oldPrinter := newInfoService, getPrinter
	t.Cleanup(func() { newInfoService = oldFactory; getPrinter = oldPrinter })
	newInfoService = func(cmd *cobra.Command) (infoService, error) { return &fakeInfoShortcutService{}, nil }

	var out, errOut bytes.Buffer
	getPrinter = func(cmd *cobra.Command) *output.Printer {
		return output.NewWithStderr(&out, &errOut, output.FormatJSON)
	}
	setFlags(cmd)
	err = cmd.RunE(cmd, nil)
	return out.String(), errOut.String(), err
}

func TestInfoShortcutAddressTrackerMissingAddress(t *testing.T) {
	cmd := newInfoAddressTrackerCmd()
	_, stderr, err := runOnchainShortcut(t, cmd, func(c *cobra.Command) {
		require.NoError(t, c.Flags().Set("chain", "eth"))
	})
	require.Error(t, err)
	assert.Contains(t, stderr, "missing required flag: address")
}

func TestInfoShortcutAddressTrackerDegradedFundFlow(t *testing.T) {
	cmd := newInfoAddressTrackerCmd()
	stdout, _, err := runOnchainShortcut(t, cmd, func(c *cobra.Command) {
		require.NoError(t, c.Flags().Set("address", "0xabc"))
		require.NoError(t, c.Flags().Set("chain", "eth"))
	})
	require.NoError(t, err)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &payload))
	assert.Equal(t, true, payload["fund_flow_unavailable"])
	assert.Contains(t, payload["coverage_note"], "trace_fund_flow")
}

func TestInfoShortcutTokenOnchainMissingInput(t *testing.T) {
	cmd := newInfoTokenOnchainCmd()
	_, stderr, err := runOnchainShortcut(t, cmd, func(c *cobra.Command) {})
	require.Error(t, err)
	assert.Contains(t, stderr, "either symbol or address is required")
}

func TestInfoShortcutTokenOnchainDegradedSmartMoney(t *testing.T) {
	cmd := newInfoTokenOnchainCmd()
	stdout, _, err := runOnchainShortcut(t, cmd, func(c *cobra.Command) {
		require.NoError(t, c.Flags().Set("symbol", "USDT"))
	})
	require.NoError(t, err)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &payload))
	assert.Equal(t, true, payload["smart_money_unavailable"])
	assert.Contains(t, payload["coverage_note"], "get_smart_money")
}
