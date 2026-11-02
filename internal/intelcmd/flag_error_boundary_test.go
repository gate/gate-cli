package intelcmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/gate/gate-cli/internal/output"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// The flag hook and Execute share stderr ownership. Preserve the original
// error for hints/errors.Is while preventing a second copy of the envelope.
func TestFlagErrorPrintedOnceAtExecuteBoundary(t *testing.T) {
	for _, format := range []string{"json", "pretty", "table"} {
		t.Run(format, func(t *testing.T) {
			root := &cobra.Command{Use: "gate-cli"}
			root.PersistentFlags().String("format", format, "")
			var stderr bytes.Buffer
			root.SetErr(&stderr)
			cause := errors.New("unknown flag: --bad-fixture")
			err := printFlagErrorToStderr(root, cause)
			require.ErrorIs(t, err, cause)
			require.Equal(t, cause.Error(), err.Error())
			EmitExecuteErrorEnvelope(&stderr, root, []string{"gate-cli", "--format", format}, err, err.Error())
			require.Equal(t, 1, bytes.Count(stderr.Bytes(), []byte(cause.Error())), "flag error printed twice: %s", stderr.String())
			require.ErrorIs(t, err, ErrSilenced)
		})
	}
}

// Preserve programmatic root defaults as well as parsed-value precedence.
// These helper guards cannot be instantiated as a nil root in a CLI process.
func TestErrorFormatFallbackBoundaries(t *testing.T) {
	t.Setenv("GATE_CLI_AGENT", "")
	t.Setenv("GATE_AI_AGENT", "")
	require.Equal(t, output.FormatPretty, resolveOutputFormat(nil))
	require.Equal(t, output.FormatPretty, resolveOutputFormatFromArgv(nil, nil))
	require.Equal(t, output.FormatPretty, resolveOutputFormatFromArgv(&cobra.Command{}, nil))
	for _, c := range []struct {
		initial        string
		clear, changed bool
		want           output.Format
	}{
		{"json", false, false, output.FormatJSON},
		{"table", false, false, output.FormatTable},
		{"", false, false, output.FormatPretty},
		{"json", true, false, output.FormatJSON},
		{"pretty", false, true, output.FormatJSON},
	} {
		root := &cobra.Command{}
		root.PersistentFlags().String("format", c.initial, "")
		if c.clear {
			require.NoError(t, root.PersistentFlags().Lookup("format").Value.Set(""))
		}
		var argv []string
		if c.changed {
			require.NoError(t, root.PersistentFlags().Set("format", "json"))
			argv = []string{"gate-cli", "--format", "pretty"}
		}
		require.Equal(t, c.want, resolveOutputFormatFromArgv(root, argv))
	}
}

// An unknown/ill-typed flag can stop pflag before the later --format is parsed.
// The sole error owner must still honor the user's raw argv and Agent default.
func TestFlagErrorHonorsUnparsedFormat(t *testing.T) {
	for _, tc := range []struct {
		name  string
		args  []string
		agent bool
	}{
		{"unknown-before-format", []string{"leaf", "--bad-fixture", "--format", "json"}, false},
		{"equals-format", []string{"leaf", "--bad-fixture", "--format=json"}, false},
		{"invalid-scalar-before-format", []string{"leaf", "--count", "bad", "--format", "json"}, false},
		{"agent-default", []string{"leaf", "--bad-fixture"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GATE_CLI_AGENT", "")
			t.Setenv("GATE_AI_AGENT", "")
			if tc.agent {
				t.Setenv("GATE_CLI_AGENT", "true")
			}
			original := os.Args
			os.Args = append([]string{"gate-cli"}, tc.args...)
			t.Cleanup(func() { os.Args = original })
			root := &cobra.Command{Use: "gate-cli", SilenceErrors: true, SilenceUsage: true}
			root.PersistentFlags().String("format", "pretty", "")
			leaf := &cobra.Command{Use: "leaf", Run: func(*cobra.Command, []string) { t.Error("bad flags reached business") }}
			leaf.Flags().Int("count", 0, "")
			root.AddCommand(leaf)
			InstallFlagErrorHook(root)
			var stderr bytes.Buffer
			root.SetErr(&stderr)
			root.SetArgs(tc.args)
			err := root.Execute()
			require.Error(t, err)
			EmitExecuteErrorEnvelope(&stderr, root, os.Args, err, err.Error())
			var envelope map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(stderr.Bytes(), &envelope), "unparsed --format lost: %s", stderr.String())
			require.NotEmpty(t, envelope["error"])
		})
	}
}

func TestFlagErrorNotPrintedWithoutCommandOrError(t *testing.T) {
	cause := errors.New("fixture")
	require.Same(t, cause, printFlagErrorToStderr(nil, cause))
	root := &cobra.Command{Use: "gate-cli"}
	var stderr bytes.Buffer
	root.SetErr(&stderr)
	require.NoError(t, printFlagErrorToStderr(root, nil))
	require.Empty(t, stderr.String())
}
