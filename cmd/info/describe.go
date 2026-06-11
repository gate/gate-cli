package info

import (
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/intelcmd"
)

var describeCmd = &cobra.Command{
	Use:     "describe --name <tool-name-or-command>",
	Short:   "Describe one Info capability",
	Example: "  gate-cli info describe --name \"info coin get-coin-info\" --format json",
	RunE:    runInfoDescribe,
}

func init() {
	describeCmd.Flags().String("name", "", "Info MCP tool name or CLI command (e.g. info coin get-coin-info)")
	_ = describeCmd.MarkFlagRequired("name")
	Cmd.AddCommand(describeCmd)
}

func runInfoDescribe(cmd *cobra.Command, args []string) error {
	p := getPrinter(cmd)
	if p.IsTable() {
		return intelcmd.FailLeafUnsupportedTable(p, "info")
	}
	svc, err := newInfoService(cmd)
	if err != nil {
		return intelcmd.FailIntelClientInit(p, err, "info", "describe", "")
	}

	name, _ := cmd.Flags().GetString("name")
	name = intelcmd.ResolveMCPToolName("info", name)
	tool, httpResp, err := svc.DescribeTool(cmd.Context(), name)
	if err != nil {
		return intelcmd.FailDescribeTransport(p, err, httpResp, "info", name)
	}
	return intelcmd.RenderDescribeTool(p, "info", tool)
}
