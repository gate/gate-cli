package configcmd

import (
	"github.com/gate/gate-cli/internal/config"
	"github.com/gate/gate-cli/internal/output"
)

// PrintFileCheck preserves the local report before returning a findings summary.
// This package does not import clients or initialize authentication to diagnose a
// configuration file. Print failure takes priority over the findings summary.
func PrintFileCheck(p *output.Printer, report config.FileCheck) error {
	if err := p.Print(report); err != nil {
		return err
	}
	if report.Status != "pass" {
		return &output.GateError{Status: 422, Label: "CONFIG_CHECK_FAILED", Message: "local config file check found an issue", SuggestedNextAction: report.Remediation}
	}
	return nil
}
