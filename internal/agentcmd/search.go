//go:build agent

package agentcmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/cmdhint"
	"github.com/gate/gate-cli/internal/cmdindex"
	"github.com/gate/gate-cli/internal/cmdutil"
)

// NewSearchCmd returns the agent-search discovery command.
func NewSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent-search --query <text>",
		Short: "Search runnable gate-cli leaf commands (agent discovery)",
		Long:  "Keyword search over the command tree. Prefer over root or group --help when GATE_CLI_AGENT=1.",
		RunE:  runSearch,
	}
	cmd.Flags().String("query", "", "Keywords (e.g. coin overview, market kline, explain market move)")
	cmd.Flags().String("domain", "", "Limit to domain: cex, info, news, config (aliases: trading, intel)")
	cmd.Flags().Int("limit", 10, "Maximum matches")
	_ = cmd.MarkFlagRequired("query")
	return cmd
}

func runSearch(cmd *cobra.Command, args []string) error {
	query, _ := cmd.Flags().GetString("query")
	domain, _ := cmd.Flags().GetString("domain")
	limit, _ := cmd.Flags().GetInt("limit")
	query = strings.TrimSpace(query)
	if query == "" {
		return fmt.Errorf("missing required flag: query")
	}
	entries := DiscoveryEntries(cmd.Root(), domain)
	hits := cmdindex.Search(entries, query, limit)
	matches := cmdhint.EnrichSearchMatches(hits)
	commands := make([]string, 0, len(hits))
	for _, h := range hits {
		commands = append(commands, cmdindex.CLICommandLine(h.Path))
	}
	p := cmdutil.GetPrinter(cmd)
	out := map[string]interface{}{
		"query":    query,
		"matches":  matches,
		"commands": commands,
	}
	if cmdhint.AgentModeEnabled() {
		out["agent_resolve_hint"] = cmdhint.AgentResolveHint(query)
	}
	if d := strings.TrimSpace(domain); d != "" {
		out["domain"] = d
	}
	return p.Print(out)
}
