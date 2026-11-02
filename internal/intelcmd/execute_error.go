package intelcmd

import (
	"errors"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/agentfeature"
	"github.com/gate/gate-cli/internal/exitcode"
	"github.com/gate/gate-cli/internal/output"
)

// EmitExecuteErrorEnvelope prints {"error":…} for cobra execution failures when the user
// requested json/pretty output, and API/unknown-command/missing-auth failures in every format.
// Skips when Intel already printed via FailAfterPrintError.
func EmitExecuteErrorEnvelope(w io.Writer, root *cobra.Command, argv []string, err error, message string) {
	if w == nil || err == nil {
		return
	}
	if errors.Is(err, ErrSilenced) {
		return
	}
	format := resolveOutputFormatFromArgv(root, argv)
	p := output.NewWithStderr(io.Discard, w, format)
	// API错误不受成功结果格式限制；table同样需要状态、追踪与结果未知的恢复建议。
	var apiError *output.GateError
	if errors.As(err, &apiError) {
		p.PrintError(apiError)
		return
	}
	// Legacy runnable groups return Cobra's plain NoArgs error. Table output
	// still needs the actual unknown-command reason before optional hints.
	// A missing local credential carries exit code 4 but no GateError DTO.
	// Keep the existing error/exit semantics and make the message visible in
	// table mode too; already-printed and typed API errors returned above.
	var coded *exitcode.Error
	missingAuth := errors.As(err, &coded) && coded.Code == 4
	if format != output.FormatJSON && format != output.FormatPretty && !strings.HasPrefix(err.Error(), "unknown command ") && !missingAuth {
		return
	}
	msg := strings.TrimSpace(message)
	if msg == "" {
		msg = strings.TrimSpace(err.Error())
	}
	if msg == "" {
		return
	}
	ge := output.InvalidArgsError(msg)
	if missingAuth && format == output.FormatTable {
		// This newly visible path is a local auth failure, not a flag error.
		// Keep legacy JSON/pretty envelopes and the original exit code intact.
		ge = &output.GateError{Status: 401, Label: "AUTH_REQUIRED", Message: msg, SuggestedNextAction: "run gate-cli cex auth login or configure GATE_API_KEY/GATE_API_SECRET before trying again"}
	}
	output.FillAgentErrorConvergence(ge)
	p.PrintError(ge)
}

func resolveOutputFormatFromArgv(root *cobra.Command, argv []string) output.Format {
	if root != nil {
		if f := root.PersistentFlags().Lookup("format"); f != nil && f.Changed {
			return output.ParseFormat(f.Value.String())
		}
	}
	for i := 0; i < len(argv); i++ {
		a := argv[i]
		if a == "--format" && i+1 < len(argv) {
			return output.ParseFormat(argv[i+1])
		}
		if strings.HasPrefix(a, "--format=") {
			return output.ParseFormat(strings.TrimPrefix(a, "--format="))
		}
	}
	if agentfeature.RuntimeEnvEnabled() {
		return output.FormatJSON
	}
	// A caller can supply a non-pretty default without marking the flag Changed
	// (unit commands and embedded roots do). Retain it when argv has no override.
	if root != nil {
		if f := root.PersistentFlags().Lookup("format"); f != nil {
			raw := strings.TrimSpace(f.Value.String())
			if raw == "" {
				raw = strings.TrimSpace(f.DefValue)
			}
			if raw != "" {
				return output.ParseFormat(raw)
			}
		}
	}
	return output.FormatPretty
}
