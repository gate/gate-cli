package info

import (
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/intelcmd"
	"github.com/gate/gate-cli/internal/mcpspec"
)

var mcpSpecCmd = &cobra.Command{
	Use:   "mcp-spec",
	Short: "Print embedded Info MCP inputs/spec JSON (offline, for agents and LLMs)",
	Long: "Outputs the same document as specs/mcp/info-mcp-tools-inputs-logic.json bundled in the binary: " +
		"tool names, fields, enums, default/max bounds, and logic text. No MCP network call; use with --format json or pretty. " +
		"When specs change, copy into internal/mcpspec/bundled/ so tests and embed stay in sync.",
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
