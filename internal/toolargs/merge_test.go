package toolargs

import (
	"os"
	"path/filepath"
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

func TestMergeFromCommand_StringArrayExplicitEmptyPreserved(t *testing.T) {
	cmd := &cobra.Command{Use: "call"}
	cmd.Flags().String("params", "", "")
	cmd.Flags().String("args-json", "", "")
	cmd.Flags().String("args-file", "", "")
	cmd.Flags().StringArray("symbols", nil, "")
	require.NoError(t, cmd.Flags().Set("symbols", ""))

	got, err := MergeFromCommand(cmd, MergeOptions{ReservedFlags: map[string]struct{}{
		"params": {}, "args-json": {}, "args-file": {},
	}})
	require.NoError(t, err)
	assert.Equal(t, []string{}, got["symbols"])
}

func TestMergeFromCommand_StringSliceExplicitEmptyPreserved(t *testing.T) {
	cmd := &cobra.Command{Use: "call"}
	cmd.Flags().String("params", "", "")
	cmd.Flags().String("args-json", "", "")
	cmd.Flags().String("args-file", "", "")
	cmd.Flags().StringSlice("tags", nil, "")
	require.NoError(t, cmd.Flags().Set("tags", ""))

	got, err := MergeFromCommand(cmd, MergeOptions{ReservedFlags: map[string]struct{}{
		"params": {}, "args-json": {}, "args-file": {},
	}})
	require.NoError(t, err)
	assert.Equal(t, []string{}, got["tags"])
}

func TestMergeFromCommand_ArgsFileExpandsHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, "args.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"query":"BTC"}`), 0o600))

	cmd := &cobra.Command{Use: "call"}
	cmd.Flags().String("params", "", "")
	cmd.Flags().String("args-json", "", "")
	cmd.Flags().String("args-file", "", "")
	require.NoError(t, cmd.Flags().Set("args-file", "~/args.json"))

	got, err := MergeFromCommand(cmd, MergeOptions{})
	require.NoError(t, err)
	assert.Equal(t, "BTC", got["query"])
}

func TestMergeFromCommand_ArgsFileRelativeUnderCwd(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	require.NoError(t, os.Chdir(dir))
	sub := filepath.Join(dir, "sub", "a.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(sub), 0o755))
	require.NoError(t, os.WriteFile(sub, []byte(`{"k":2}`), 0o600))

	cmd := &cobra.Command{Use: "call"}
	cmd.Flags().String("params", "", "")
	cmd.Flags().String("args-json", "", "")
	cmd.Flags().String("args-file", "", "")
	require.NoError(t, cmd.Flags().Set("args-file", filepath.Join("sub", "a.json")))

	got, err := MergeFromCommand(cmd, MergeOptions{})
	require.NoError(t, err)
	assert.Equal(t, float64(2), got["k"])
}

func TestMergeFromCommand_ArgsFileRejectsParentEscape(t *testing.T) {
	dir := t.TempDir()
	parent := filepath.Dir(dir)
	evil := filepath.Join(parent, "evil-args.json")
	require.NoError(t, os.WriteFile(evil, []byte(`{"bad":true}`), 0o600))
	oldwd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	require.NoError(t, os.Chdir(dir))

	cmd := &cobra.Command{Use: "call"}
	cmd.Flags().String("params", "", "")
	cmd.Flags().String("args-json", "", "")
	cmd.Flags().String("args-file", "", "")
	require.NoError(t, cmd.Flags().Set("args-file", filepath.Join("..", filepath.Base(evil))))

	_, mergeErr := MergeFromCommand(cmd, MergeOptions{})
	require.Error(t, mergeErr)
}
