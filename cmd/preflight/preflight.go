package preflight

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/cmdutil"
	"github.com/gate/gate-cli/internal/exitcode"
	"github.com/gate/gate-cli/internal/migration"
	"github.com/gate/gate-cli/internal/output"
	"github.com/gate/gate-cli/internal/version"
)

// Cmd is the preflight command.
var Cmd = &cobra.Command{
	Use:           "preflight",
	Short:         "Run CLI-first preflight check for Gate info/news",
	RunE:          runPreflight,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	Cmd.Flags().Bool("fallback-enabled", true, "Enable MCP fallback route when CLI is not installed")
}

func runPreflight(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	if p.IsTable() {
		p.PrintError(output.UnsupportedTableFormatError())
		return exitcode.New(exitcode.RenderOrInternal, errors.New("unsupported format"))
	}
	fallbackEnabled, _ := cmd.Flags().GetBool("fallback-enabled")
	result := migration.BuildPreflight(migration.PreflightOptions{
		FallbackEnabled: fallbackEnabled,
		Version:         version.Version,
	})
	if err := p.Print(result); err != nil {
		return exitcode.New(exitcode.RenderOrInternal, err)
	}
	if result.Route == "BLOCK" {
		return exitcode.New(exitcode.Failure, errors.New("preflight blocked"))
	}
	return nil
}
