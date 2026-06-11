package news

import (
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/intelcmd"
)

var describeCmd = &cobra.Command{
	Use:     "describe --name <tool-name-or-command>",
	Short:   "Describe one News capability",
	Example: "  gate-cli news describe --name \"news feed search-news\" --format json",
	RunE:    runNewsDescribe,
}

func init() {
	describeCmd.Flags().String("name", "", "News MCP tool name or CLI command (e.g. news feed search-news)")
	_ = describeCmd.MarkFlagRequired("name")
	Cmd.AddCommand(describeCmd)
}

func runNewsDescribe(cmd *cobra.Command, args []string) error {
	p := getPrinter(cmd)
	if p.IsTable() {
		return intelcmd.FailLeafUnsupportedTable(p, "news")
	}
	svc, err := newNewsService(cmd)
	if err != nil {
		return intelcmd.FailIntelClientInit(p, err, "news", "describe", "")
	}

	name, _ := cmd.Flags().GetString("name")
	name = intelcmd.ResolveMCPToolName("news", name)
	tool, httpResp, err := svc.DescribeTool(cmd.Context(), name)
	if err != nil {
		return intelcmd.FailDescribeTransport(p, err, httpResp, "news", name)
	}
	return intelcmd.RenderDescribeTool(p, "news", tool)
}
