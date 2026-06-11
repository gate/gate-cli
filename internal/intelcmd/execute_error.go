package intelcmd

import (
	"errors"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/output"
)

// EmitExecuteErrorEnvelope prints {"error":…} for cobra execution failures when the user
// requested json/pretty output. Skips when Intel already printed via FailAfterPrintError.
func EmitExecuteErrorEnvelope(w io.Writer, root *cobra.Command, argv []string, err error, message string) {
	if w == nil || err == nil {
		return
	}
	if errors.Is(err, ErrSilenced) {
		return
	}
	format := resolveOutputFormatFromArgv(root, argv)
	if format != output.FormatJSON && format != output.FormatPretty {
		return
	}
	msg := strings.TrimSpace(message)
	if msg == "" {
		msg = strings.TrimSpace(err.Error())
	}
	if msg == "" {
		return
	}
	p := output.NewWithStderr(io.Discard, w, format)
	ge := output.InvalidArgsError(msg)
	output.FillAgentErrorConvergence(ge)
	p.PrintError(ge)
}

func resolveOutputFormatFromArgv(root *cobra.Command, argv []string) output.Format {
	if root != nil {
		if raw, err := root.PersistentFlags().GetString("format"); err == nil && strings.TrimSpace(raw) != "" {
			return output.ParseFormat(raw)
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
	if strings.TrimSpace(os.Getenv("GATE_CLI_AGENT")) == "1" || strings.TrimSpace(os.Getenv("GATE_AI_AGENT")) == "1" {
		return output.FormatJSON
	}
	return output.FormatPretty
}
