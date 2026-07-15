//go:build agent

package agentcmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/cmdhint"
	"github.com/gate/gate-cli/internal/cmdutil"
)

// NewValidateCmd returns the agent-validate CI helper command.
func NewValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent-validate",
		Short: "Validate agent mcp_catalog paths against the live info/news cobra tree",
		Long:  "Reports baseline MCP tools (50) and info/news shortcuts (10) whose CLI paths are missing from the runnable cobra tree. Use in CI after adding MCP tools or shortcuts.",
		RunE: func(cmd *cobra.Command, args []string) error {
			report := cmdhint.ValidateAgentDiscovery(cmd.Root())
			p := cmdutil.GetPrinter(cmd)
			if err := p.Print(report); err != nil {
				return err
			}
			if !report.OK {
				return fmt.Errorf("%d agent discovery path(s) not found in cobra tree", report.TotalCount)
			}
			return nil
		},
	}
	return cmd
}
