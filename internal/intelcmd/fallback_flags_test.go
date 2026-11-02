package intelcmd

import (
	"testing"

	"github.com/gate/gate-cli/internal/toolargs"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func attachRootIntelPersistentFlags(root *cobra.Command) {
	root.PersistentFlags().String("format", "pretty", "")
	root.PersistentFlags().String("profile", "default", "")
	root.PersistentFlags().Bool("debug", false, "")
	root.PersistentFlags().Bool("verbose", false, "")
	root.PersistentFlags().Int64("max-output-bytes", 0, "")
	root.PersistentFlags().String("api-key", "", "")
	root.PersistentFlags().String("api-secret", "", "")
}

func TestReservedMCPJSONFallbackFlags_ExcludesRootGlobalsFromLeafMerge(t *testing.T) {
	root := &cobra.Command{Use: "gate-cli"}
	attachRootIntelPersistentFlags(root)

	leaf := &cobra.Command{Use: "get-market-snapshot"}
	leaf.Flags().String("symbol", "", "")
	leaf.Flags().String("source", "", "")
	leaf.Flags().String("scope", "", "")
	AddFallbackArgFlags(leaf)
	root.AddCommand(leaf)

	var merged map[string]interface{}
	var mergeErr error
	leaf.RunE = func(cmd *cobra.Command, args []string) error {
		merged, mergeErr = toolargs.MergeFromCommand(cmd, toolargs.MergeOptions{
			ReservedFlags: ReservedMCPJSONFallbackFlags(),
		})
		return mergeErr
	}

	root.SetArgs([]string{
		"get-market-snapshot",
		"--symbol", "AKE",
		"--source", "spot",
		"--scope", "full",
		"--format", "json",
		"--verbose",
		"--debug",
		"--profile", "default",
		"--max-output-bytes", "1024",
		"--api-key", "not-an-intel-bearer",
		"--api-secret", "not-used-for-mcp",
	})
	require.NoError(t, root.Execute())
	require.NoError(t, mergeErr)
	require.NotNil(t, merged)

	assert.Equal(t, map[string]interface{}{
		"symbol": "AKE",
		"source": "spot",
		"scope":  "full",
	}, merged)

	for _, leaked := range []string{
		"format", "verbose", "debug", "profile", "max_output_bytes", "max-output-bytes",
		"api_key", "api-key", "api_secret", "api-secret", "help",
		"params", "args_json", "args-json", "args_file", "args-file",
	} {
		assert.NotContains(t, merged, leaked, "CLI-only flag %q must not appear in MCP args", leaked)
	}
}

func TestReservedMCPJSONFallbackFlags_DoesNotStripExplicitParamsJSONKeys(t *testing.T) {
	// Reserved applies to cobra flags only. Explicit JSON fallback may still carry a key
	// named format if the user put it there intentionally (MCP will reject if invalid).
	cmd := &cobra.Command{Use: "leaf"}
	AddFallbackArgFlags(cmd)
	cmd.Flags().String("symbol", "", "")
	require.NoError(t, cmd.Flags().Set("params", `{"symbol":"AKE","format":"json"}`))
	require.NoError(t, cmd.Flags().Set("symbol", "BTC"))

	got, err := toolargs.MergeFromCommand(cmd, toolargs.MergeOptions{
		ReservedFlags: ReservedMCPJSONFallbackFlags(),
	})
	require.NoError(t, err)
	assert.Equal(t, "BTC", got["symbol"])
	assert.Equal(t, "json", got["format"])
}

func TestReservedInvokeFlags_IncludesNameAndGlobals(t *testing.T) {
	m := ReservedInvokeFlags()
	for _, name := range []string{
		"name", "params", "args-json", "args-file",
		"format", "verbose", "debug", "profile",
		"max-output-bytes", "api-key", "api-secret", "help",
	} {
		_, ok := m[name]
		assert.True(t, ok, "missing reserved flag %q", name)
	}
}

func TestReservedCLIGlobalFlags_StableSet(t *testing.T) {
	got := ReservedCLIGlobalFlags()
	assert.Equal(t, len(rootCLIOnlyFlagNames), len(got))
	for _, name := range rootCLIOnlyFlagNames {
		_, ok := got[name]
		assert.True(t, ok, name)
	}
}
