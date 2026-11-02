//go:build agent

package agentcmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/cmdhint"
	"github.com/gate/gate-cli/internal/cmdindex"
	"github.com/gate/gate-cli/internal/cmdutil"
	"github.com/gate/gate-cli/internal/output"
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
	cmd.Flags().Bool("details", false, "Explicit CEX parameter and action declarations; requires --domain cex")
	cmd.Flags().String("command", "", "Canonical CEX leaf path; only with --details")
	return cmd
}

func runIndex(cmd *cobra.Command, args []string) error {
	domain, _ := cmd.Flags().GetString("domain")
	details, _ := cmd.Flags().GetBool("details")
	selector, _ := cmd.Flags().GetString("command")
	if details {
		if domain != "cex" {
			return fmt.Errorf("--details requires explicit --domain cex")
		}
		p := cmdutil.GetPrinter(cmd)
		if p.IsTable() {
			return output.UnsupportedTableFormatError()
		}
		commands, err := cexDetails(cmd.Root(), selector)
		if err != nil {
			return err
		}
		return p.Print(struct {
			SchemaVersion int             `json:"schema_version"`
			Commands      []commandDetail `json:"commands"`
		}{1, commands})
	}
	if selector != "" {
		return fmt.Errorf("--command requires --details")
	}

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
