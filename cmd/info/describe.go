package info

import (
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/intelcmd"
	"github.com/gate/gate-cli/internal/intelfacade"
)

var describeCmd = &cobra.Command{
	Use:   "describe --name <tool-name>",
	Short: "Describe one Info capability",
	RunE:  runInfoDescribe,
}

func init() {
	describeCmd.Flags().String("name", "", "Info tool name")
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
	tool, httpResp, err := svc.DescribeTool(cmd.Context(), name)
	if err != nil {
		return intelcmd.FailDescribeTransport(p, err, httpResp, "info", name)
	}
	if p.IsJSON() {
		return p.Print(tool)
	}
	return p.WritePretty(intelfacade.DescribePrettyText(tool))
}
