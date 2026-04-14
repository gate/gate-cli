package info

import (
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/mcpclient"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available Info capabilities",
	RunE:  runInfoList,
}

func init() {
	Cmd.AddCommand(listCmd)
}

func runInfoList(cmd *cobra.Command, args []string) error {
	p := getPrinter(cmd)
	svc, err := newInfoService(cmd)
	if err != nil {
		p.PrintError(mcpclient.ParseError(err, nil, "POST", "info/list", ""))
		return nil
	}

	items, httpResp, err := svc.ListTools(cmd.Context())
	if err != nil {
		p.PrintError(mcpclient.ParseError(err, httpResp, "POST", "info/list", ""))
		return nil
	}

	if p.IsJSON() {
		return p.Print(items)
	}

	rows := make([][]string, 0, len(items))
	for _, item := range items {
		hasSchema := "no"
		if item.HasInputSchema {
			hasSchema = "yes"
		}
		rows = append(rows, []string{item.Name, item.Description, hasSchema})
	}
	return p.Table([]string{"Name", "Description", "HasInputSchema"}, rows)
}
