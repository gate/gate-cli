package configcmd

import (
	"github.com/gate/gate-cli/internal/cmdutil"
	"github.com/gate/gate-cli/internal/config"
	shared "github.com/gate/gate-cli/internal/configcmd"
	"github.com/gate/gate-cli/internal/output"
	"github.com/spf13/cobra"
)

// runCheck adapts an explicit command only. Keeping cmdutil in the command layer
// avoids a configcmd -> client -> oauth2 dependency cycle in config consumers.
func runCheck(cmd *cobra.Command, _ []string) error {
	p := cmdutil.GetPrinter(cmd)
	if p.IsTable() {
		return output.UnsupportedTableFormatError()
	}
	return shared.PrintFileCheck(p, config.CheckFile(config.DefaultConfigPath()))
}
