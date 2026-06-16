package intelcmd

import (
	"bytes"
	"errors"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gate/gate-cli/internal/exitcode"
	"github.com/gate/gate-cli/internal/output"
)

func TestPrintFlagErrorToStderrJSON(t *testing.T) {
	root := &cobra.Command{Use: "gate-cli"}
	root.PersistentFlags().String("format", "json", "")
	leaf := &cobra.Command{Use: "describe"}
	root.AddCommand(leaf)

	var errOut bytes.Buffer
	leaf.SetErr(&errOut)
	err := printFlagErrorToStderr(leaf, errors.New(`required flag(s) "name" not set`))
	require.Error(t, err)
	out := errOut.String()
	assert.Contains(t, out, `"error"`)
	assert.Contains(t, out, `"label":"INVALID_ARGUMENTS"`)
	assert.Contains(t, out, "required flag")
}

func TestPrintFlagErrorToStderrPlain(t *testing.T) {
	root := &cobra.Command{Use: "gate-cli"}
	root.PersistentFlags().String("format", "pretty", "")
	leaf := &cobra.Command{Use: "describe"}
	root.AddCommand(leaf)

	var errOut bytes.Buffer
	leaf.SetErr(&errOut)
	err := printFlagErrorToStderr(leaf, errors.New(`required flag(s) "name" not set`))
	require.Error(t, err)
	out := errOut.String()
	assert.Contains(t, out, "Error [400")
	assert.NotContains(t, out, `"error"`)
}

func TestEmitExecuteErrorEnvelopeRequiredFlag(t *testing.T) {
	var errOut bytes.Buffer
	err := errors.New(`required flag(s) "name" not set`)
	EmitExecuteErrorEnvelope(&errOut, nil, []string{"gate-cli", "info", "describe", "--format", "json"}, err, err.Error())
	out := errOut.String()
	assert.Contains(t, out, `"error"`)
	assert.Contains(t, out, `"label":"INVALID_ARGUMENTS"`)
}

func TestEmitExecuteErrorEnvelopeSkipsSilenced(t *testing.T) {
	var errOut bytes.Buffer
	EmitExecuteErrorEnvelope(&errOut, nil, []string{"gate-cli", "info", "+coin-overview", "--format", "json"}, exitcode.New(1, ErrSilenced), "")
	assert.Empty(t, errOut.String())
}

func TestResolveOutputFormatFromArgv(t *testing.T) {
	assert.Equal(t, output.FormatJSON, resolveOutputFormatFromArgv(nil, []string{"gate-cli", "info", "describe", "--format", "json"}))
	assert.Equal(t, output.FormatJSON, resolveOutputFormatFromArgv(nil, []string{"gate-cli", "info", "describe", "--format=json"}))
	root := &cobra.Command{Use: "gate-cli"}
	root.PersistentFlags().String("format", "pretty", "")
	assert.Equal(t, output.FormatPretty, resolveOutputFormatFromArgv(root, []string{"gate-cli", "info", "list"}))
}

func TestResolveOutputFormatAgentEnv(t *testing.T) {
	t.Setenv("GATE_CLI_AGENT", "true")
	t.Cleanup(func() { _ = os.Unsetenv("GATE_CLI_AGENT") })
	root := &cobra.Command{Use: "gate-cli"}
	root.PersistentFlags().String("format", "pretty", "")
	assert.Equal(t, output.FormatJSON, resolveOutputFormatFromArgv(root, []string{"gate-cli", "info", "list"}))
}
