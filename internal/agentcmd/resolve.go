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

// NewResolveCmd returns the agent-resolve discovery command.
func NewResolveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent-resolve --query <text>",
		Short: "Resolve info/news intent to agent-leaves templates and leaf commands",
		Long:  "Layer-2 discovery: match curated agent-leaves first, then keyword search over info/news runnable leaves.",
		RunE:  runResolve,
	}
	cmd.Flags().String("query", "", "Intent phrase (e.g. brief BTC, latest events, token risk)")
	cmd.Flags().String("domain", "", "Limit leaf search: info or news (default: both)")
	cmd.Flags().Int("limit", 5, "Maximum leaf-search matches after curated leaves")
	_ = cmd.MarkFlagRequired("query")
	return cmd
}

func runResolve(cmd *cobra.Command, args []string) error {
	query, _ := cmd.Flags().GetString("query")
	domain, _ := cmd.Flags().GetString("domain")
	limit, _ := cmd.Flags().GetInt("limit")
	query = strings.TrimSpace(query)
	if query == "" {
		return fmt.Errorf("missing required flag: query")
	}
	resolved := cmdhint.ResolveAgentIntent(query, 3, 3, domain)
	entries := DiscoveryEntries(cmd.Root(), domain)
	searchHits := cmdindex.Search(entries, query, limit)
	commands := make([]string, 0, len(searchHits))
	for _, h := range searchHits {
		commands = append(commands, cmdindex.CLICommandLine(h.Path))
	}
	p := cmdutil.GetPrinter(cmd)
	out := map[string]interface{}{
		"query":           query,
		"resolved_leaves": resolved,
		"search_matches":  searchHits,
		"commands":        commands,
	}
	if d := strings.TrimSpace(domain); d != "" {
		out["domain"] = d
	}
	return p.Print(out)
}
