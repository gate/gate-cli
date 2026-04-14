package toolargs

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeFromCommand_OverlayFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "call"}
	cmd.Flags().String("params", `{"query":"ETH","limit":5}`, "")
	cmd.Flags().String("args-json", "", "")
	cmd.Flags().String("args-file", "", "")
	cmd.Flags().String("query", "", "")
	cmd.Flags().Int("limit", 10, "")
	require.NoError(t, cmd.Flags().Set("params", `{"query":"ETH","limit":5}`))
	require.NoError(t, cmd.Flags().Set("query", "BTC"))
	require.NoError(t, cmd.Flags().Set("limit", "20"))

	got, err := MergeFromCommand(cmd, MergeOptions{ReservedFlags: map[string]struct{}{
		"params": {}, "args-json": {}, "args-file": {},
	}})
	require.NoError(t, err)
	assert.Equal(t, "BTC", got["query"])
	assert.Equal(t, int64(20), got["limit"])
}

func TestMergeFromCommand_MutualExclusiveBase(t *testing.T) {
	cmd := &cobra.Command{Use: "call"}
	cmd.Flags().String("params", "", "")
	cmd.Flags().String("args-json", "", "")
	cmd.Flags().String("args-file", "", "")
	require.NoError(t, cmd.Flags().Set("params", `{"a":1}`))
	require.NoError(t, cmd.Flags().Set("args-json", `{"b":2}`))
	_, err := MergeFromCommand(cmd, MergeOptions{})
	require.Error(t, err)
}
