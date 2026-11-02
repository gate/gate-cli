package intelcmd

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/output"
	"github.com/stretchr/testify/require"
)

// Unknown group paths must be explicit in every format, including legacy runnable
// groups whose own cobra.NoArgs validation returns an ordinary error.
func TestExecuteUnknownCommandAllFormats(t *testing.T) {
	failure := cobra.NoArgs(&cobra.Command{Use: "group"}, []string{"nosuch"})
	for _, format := range []string{"json", "pretty", "table"} {
		var stderr bytes.Buffer
		EmitExecuteErrorEnvelope(&stderr, nil, []string{"--format", format}, failure, failure.Error())
		require.Contains(t, stderr.String(), `unknown command`)
		require.Contains(t, stderr.String(), `nosuch`)
	}
	var stderr bytes.Buffer
	EmitExecuteErrorEnvelope(&stderr, nil, []string{"--format", "table"}, fmt.Errorf("ordinary non-command failure"), "")
	require.Empty(t, stderr.String(), "preserve the existing policy for other plain errors")
}

// TestExecuteAPIErrorAllFormats 保证格式切换不丢失状态、追踪信息和结果未知时的恢复建议。
func TestExecuteAPIErrorAllFormats(t *testing.T) {
	for _, format := range []string{"json", "pretty", "table"} {
		for _, status := range []int{401, 403, 503} {
			t.Run(fmt.Sprintf("%s/%d", format, status), func(t *testing.T) {
				var stderr bytes.Buffer
				failure := &output.GateError{Status: status, Label: "SYNTHETIC_FAILURE", Message: "synthetic API failure", TraceID: "synthetic-trace", SuggestedNextAction: "Check order status before resubmitting"}
				EmitExecuteErrorEnvelope(&stderr, nil, []string{"--format", format}, fmt.Errorf("command: %w", failure), "")
				for _, expected := range []string{fmt.Sprint(status), failure.Label, failure.Message, failure.TraceID, failure.SuggestedNextAction} {
					require.Contains(t, stderr.String(), expected)
				}
			})
		}
	}
}
