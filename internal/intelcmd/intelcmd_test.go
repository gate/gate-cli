package intelcmd

import (
	"bytes"
	"errors"
	"net/http"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gate/gate-cli/internal/exitcode"
	"github.com/gate/gate-cli/internal/output"
)

func TestFailAfterPrintErrorReturnsExitCode1(t *testing.T) {
	var out, errOut bytes.Buffer
	p := output.NewWithStderr(&out, &errOut, output.FormatJSON)
	err := FailAfterPrintError(p, &output.GateError{Status: 502, Label: "INTEL_RESULT_ERROR", Message: "tool returned isError=true"})
	require.Error(t, err)
	var coded *exitcode.Error
	require.True(t, errors.As(err, &coded))
	assert.Equal(t, 1, coded.Code)
	assert.Contains(t, errOut.String(), `"label":"INTEL_RESULT_ERROR"`)
	assert.Empty(t, out.String())
}

func TestGateErrorForIntelToolIsErrorCopiesTraceID(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("x-gate-trace-id", "abc-123")
	ge := GateErrorForIntelToolIsError("info_coin_get_coin_info", resp)
	require.Equal(t, "abc-123", ge.TraceID)
}

func TestGateErrorForIntelToolIsErrorNoTraceWhenMissing(t *testing.T) {
	ge := GateErrorForIntelToolIsError("t", nil)
	assert.Empty(t, ge.TraceID)
	ge2 := GateErrorForIntelToolIsError("t", &http.Response{Header: http.Header{}})
	assert.Empty(t, ge2.TraceID)
}

func TestFailLeafUnsupportedTableMentionsListTable(t *testing.T) {
	var out, errOut bytes.Buffer
	p := output.NewWithStderr(&out, &errOut, output.FormatJSON)
	err := FailLeafUnsupportedTable(p, "info")
	require.Error(t, err)
	var coded *exitcode.Error
	require.True(t, errors.As(err, &coded))
	assert.Contains(t, errOut.String(), "list --format table")
	assert.Contains(t, errOut.String(), "gate-cli info")
}

func TestSilenceCommandTreeSetsFlagsOnChildren(t *testing.T) {
	root := &cobra.Command{Use: "intel"}
	child := &cobra.Command{Use: "list", Run: func(*cobra.Command, []string) {}}
	grand := &cobra.Command{Use: "x", Run: func(*cobra.Command, []string) {}}
	child.AddCommand(grand)
	root.AddCommand(child)
	SilenceCommandTree(root)
	assert.True(t, child.SilenceErrors)
	assert.True(t, child.SilenceUsage)
	assert.True(t, grand.SilenceErrors)
	assert.True(t, grand.SilenceUsage)
}

func TestBuildGroupedAliasesNilMakeAliasReturnsEmpty(t *testing.T) {
	out := BuildGroupedAliases(AliasBuildOptions{
		BackendPrefix: "info",
		BackendTitle:  "Info",
		ToolBaseline:  []string{"info_coin_get_coin_info"},
		MakeAlias:     nil,
	})
	assert.Nil(t, out)
}
