//go:build agent

package agentcmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/cmdhint"
	"github.com/gate/gate-cli/internal/cmdindex"
	"github.com/gate/gate-cli/internal/cmdutil"
)

// NewIndexCmd returns the agent-index discovery command.
func NewIndexCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent-index",
		Short: "Export all runnable gate-cli leaf commands for offline agent indexing",
		Long:  "Full leaf catalog (path + short help). Pair with agent-search for keyword lookup.",
		RunE:  runIndex,
	}
	cmd.Flags().String("domain", "", "Limit catalog to domain: cex, info, news, config (aliases: trading, intel)")
	return cmd
}

func runIndex(cmd *cobra.Command, args []string) error {
	domain, _ := cmd.Flags().GetString("domain")
	leaves := DiscoveryEntries(cmd.Root(), domain)
	commands := make([]string, 0, len(leaves))
	for _, e := range leaves {
		commands = append(commands, cmdindex.CLICommandLine(e.Path))
	}
	p := cmdutil.GetPrinter(cmd)
	out := map[string]interface{}{
		"count":    len(leaves),
		"leaves":   cmdhint.EnrichSearchMatches(leaves),
		"commands": commands,
	}
	if cmdhint.AgentModeEnabled() {
		out["agent_resolve_hint"] = cmdhint.AgentResolveHint("")
	}
	if d := strings.TrimSpace(domain); d != "" {
		out["domain"] = d
	}
	return p.Print(out)
}
