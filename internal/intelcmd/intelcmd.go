// Package intelcmd provides helpers shared only by gate-cli info and news (Intel MCP) commands.
// Do not use it from trading domains (spot, futures, etc.).
package intelcmd

import (
	"errors"

	"github.com/gate/gate-cli/internal/exitcode"
	"github.com/gate/gate-cli/internal/output"
)

// ErrSilenced is returned after PrintError so callers exit non-zero without cobra duplicating stderr.
var ErrSilenced = errors.New("intel command failed")

// FailAfterPrintError writes gateErr to stderr via the printer, then returns exit code 1.
// The command must use SilenceErrors=true so cobra does not print this error again.
func FailAfterPrintError(p *output.Printer, gateErr *output.GateError) error {
	if gateErr == nil {
		gateErr = &output.GateError{Status: 500, Label: "INTEL_ERROR", Message: "unknown error"}
	}
	p.PrintError(gateErr)
	return exitcode.New(1, ErrSilenced)
}

// FailUnsupportedTable prints the standard unsupported-format error and returns exit code 1.
func FailUnsupportedTable(p *output.Printer) error {
	return FailAfterPrintError(p, output.UnsupportedTableFormatError())
}

// FailLeafUnsupportedTable is for tool invoke/describe paths that are not tabular; backend is "info" or "news".
func FailLeafUnsupportedTable(p *output.Printer, backend string) error {
	base := output.UnsupportedTableFormatError()
	return FailAfterPrintError(p, &output.GateError{
		Status: base.Status,
		Label:  base.Label,
		Message: base.Message + " For a tabular tool index use `gate-cli " + backend +
			" list --format table`. Use `--format pretty` or `--format json` for tool results and `describe` output.",
	})
}
