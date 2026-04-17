package toolargs

import (
	"testing"

	"github.com/gate/gate-cli/internal/toolschema"
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

func TestNormalizeFlagStringList_JSONAndComma(t *testing.T) {
	assert.Equal(t, []string{"rsi"}, normalizeFlagStringList([]string{`["rsi"]`}))
	assert.Equal(t, []string{"ema7", "ema30"}, normalizeFlagStringList([]string{`["ema7","ema30"]`}))
	assert.Equal(t, []string{"a", "b"}, normalizeFlagStringList([]string{"a,b"}))
	assert.Equal(t, []string{"x", "y"}, normalizeFlagStringList([]string{"x", "y"}))
}

func TestMergeFromCommand_ArrayJSONToken(t *testing.T) {
	cmd := &cobra.Command{Use: "call"}
	cmd.Flags().String("params", "", "")
	cmd.Flags().String("args-json", "", "")
	cmd.Flags().String("args-file", "", "")
	schema := map[string]interface{}{
		"properties": map[string]interface{}{
			"indicators": map[string]interface{}{
				"type":        "array",
				"description": "indicators",
				"items":       map[string]interface{}{"type": "string"},
			},
		},
	}
	toolschema.ApplyInputSchemaFlags(cmd, schema)
	require.NoError(t, cmd.Flags().Parse([]string{"--indicators", `["rsi","ema30"]`}))

	got, err := MergeFromCommand(cmd, MergeOptions{ReservedFlags: map[string]struct{}{
		"params": {}, "args-json": {}, "args-file": {},
	}})
	require.NoError(t, err)
	assert.Equal(t, []string{"rsi", "ema30"}, got["indicators"])
}

func TestMergeFromCommand_BooleanSpaceSeparatedTrue(t *testing.T) {
	cmd := &cobra.Command{Use: "call"}
	cmd.Flags().String("params", "", "")
	cmd.Flags().String("args-json", "", "")
	cmd.Flags().String("args-file", "", "")
	schema := map[string]interface{}{
		"properties": map[string]interface{}{
			"with_indicators": map[string]interface{}{
				"type":        "boolean",
				"description": "with_indicators",
			},
		},
	}
	toolschema.ApplyInputSchemaFlags(cmd, schema)
	require.NoError(t, cmd.Flags().Parse([]string{"--with-indicators", "true"}))

	got, err := MergeFromCommand(cmd, MergeOptions{ReservedFlags: map[string]struct{}{
		"params": {}, "args-json": {}, "args-file": {},
	}})
	require.NoError(t, err)
	assert.Equal(t, true, got["with_indicators"])
}
