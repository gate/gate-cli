//go:build agent

package agentcmd

import (
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/cmdhint"
	"github.com/gate/gate-cli/internal/cmdutil"
)

// NewLeavesCmd returns the agent-leaves discovery command.
func NewLeavesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent-leaves",
		Short: "List high-frequency info/news leaf commands for GateAI agents",
		Long:  "Prints info/news intent-to-command mappings (no CEX). Use agent-search --domain cex for trading leaves. Prefer over crawling --help from the root.",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := cmdutil.GetPrinter(cmd)
			catalog := cmdhint.BaselineMCPCatalog()
			out := map[string]interface{}{
				"count_curated": len(cmdhint.AgentLeaves),
				"leaves":        cmdhint.AgentLeaves,
				"count_mcp":     len(catalog),
				"mcp_catalog":   catalog,
			}
			if cmdhint.AgentModeEnabled() {
				out["scope"] = "info,news"
				out["recommended_flow"] = []string{
					"gate-cli agent-resolve --query <intent> --format json",
					"gate-cli agent-search --query <intent> --format json",
				}
			}
			return p.Print(out)
		},
	}
	return cmd
}
