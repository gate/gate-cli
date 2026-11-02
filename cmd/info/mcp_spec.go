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
		"The bundled document describes public inputs and behavior; use leaf help for the corresponding command.",
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
