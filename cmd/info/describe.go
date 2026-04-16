package info

import (
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/intelfacade"
	"github.com/gate/gate-cli/internal/mcpclient"
	"github.com/gate/gate-cli/internal/output"
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
		p.PrintError(output.UnsupportedTableFormatError())
		return nil
	}
	svc, err := newInfoService(cmd)
	if err != nil {
		p.PrintError(mcpclient.ParseError(err, nil, "POST", "info/describe", ""))
		return nil
	}

	name, _ := cmd.Flags().GetString("name")
	tool, httpResp, err := svc.DescribeTool(cmd.Context(), name)
	if err != nil {
		p.PrintError(mcpclient.ParseError(err, httpResp, "POST", "info/describe", name))
		return nil
	}
	if p.IsJSON() {
		return p.Print(tool)
	}
	return p.WritePretty(intelfacade.DescribePrettyText(tool))
}
