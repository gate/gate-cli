package cmd

import (
	"github.com/gate/gate-cli/internal/output"
	"github.com/gate/gate-cli/internal/version"
)

// printVersionDetails governs only the opt-in renderer; shared Print and the
// legacy version Run retain their existing semantics.
func printVersionDetails(p *output.Printer) error {
	if p.IsTable() {
		return output.UnsupportedTableFormatError()
	}
	return p.Print(version.Details())
}
