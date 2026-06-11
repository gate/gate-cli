package news

import (
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/intelcmd"
	"github.com/gate/gate-cli/internal/mcpspec"
)

var mcpSpecCmd = &cobra.Command{
	Use:   "mcp-spec",
	Short: "Print embedded News MCP tools args/logic JSON (offline, for agents and LLMs)",
	Long: "Prints the embedded News MCP tools args/logic JSON shipped inside gate-cli (English description per tool, params, logic). " +
		"No MCP network call; use with --format json or pretty. Leaf -h reads description from this same embedded document. " +
		"Maintainers sync from gate/mcp-server into internal/mcpspec/bundled/; do not treat specs/mcp/ as a release description source.",
	Args: cobra.NoArgs,
	RunE: runNewsMCPSpec,
}

func init() {
	Cmd.AddCommand(mcpSpecCmd)
}

func runNewsMCPSpec(cmd *cobra.Command, args []string) error {
	p := getPrinter(cmd)
	if p.IsTable() {
		return intelcmd.FailLeafUnsupportedTable(p, "news")
	}
	doc, err := mcpspec.NewsToolsArgs()
	if err != nil {
		return err
	}
	return p.Print(doc)
}
