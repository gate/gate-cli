package news

import (
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/intelcmd"
	"github.com/gate/gate-cli/internal/mcpspec"
)

var mcpSpecCmd = &cobra.Command{
	Use:   "mcp-spec",
	Short: "Print embedded News MCP tools args/logic JSON (offline, for agents and LLMs)",
	Long: "Outputs the same document as specs/mcp/news-tools-args-and-logic.json bundled in the binary: " +
		"per-tool params, enums, default/max, and logic. No MCP network call; use with --format json or pretty. " +
		"When specs change, copy into internal/mcpspec/bundled/ so tests and embed stay in sync.",
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
