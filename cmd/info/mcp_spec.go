package info

import (
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/intelcmd"
	"github.com/gate/gate-cli/internal/mcpspec"
)

var mcpSpecCmd = &cobra.Command{
	Use:   "mcp-spec",
	Short: "Print embedded Info MCP inputs/spec JSON (offline, for agents and LLMs)",
	Long: "Prints the embedded Info MCP inputs/spec JSON shipped inside gate-cli (English description per tool, fields, logic). " +
		"No MCP network call; use --format json or pretty. Leaf -h reads description from this same embedded document. " +
		"Maintainers sync logic/fields from gate/mcp-server into internal/mcpspec/bundled/; routing descriptions are updated via scripts/patch-info-spec-descriptions.py (not from specs/mcp/, which is local QC only and not part of releases).",
	Args: cobra.NoArgs,
	RunE: runInfoMCPSpec,
}

func init() {
	Cmd.AddCommand(mcpSpecCmd)
}

func runInfoMCPSpec(cmd *cobra.Command, args []string) error {
	p := getPrinter(cmd)
	if p.IsTable() {
		return intelcmd.FailLeafUnsupportedTable(p, "info")
	}
	doc, err := mcpspec.InfoInputsLogic()
	if err != nil {
		return err
	}
	return p.Print(doc)
}
