package news

import (
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/intelfacade"
	"github.com/gate/gate-cli/internal/mcpclient"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available News capabilities",
	RunE:  runNewsList,
}

func init() {
	Cmd.AddCommand(listCmd)
}

func runNewsList(cmd *cobra.Command, args []string) error {
	p := getPrinter(cmd)
	svc, err := newNewsService(cmd)
	if err != nil {
		p.PrintError(mcpclient.ParseError(err, nil, "POST", "news/list", ""))
		return nil
	}

	items, httpResp, err := svc.ListTools(cmd.Context())
	if err != nil {
		p.PrintError(mcpclient.ParseError(err, httpResp, "POST", "news/list", ""))
		return nil
	}

	if p.IsJSON() {
		return p.Print(items)
	}

	if p.IsTable() {
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			params := "no"
			if item.HasInputSchema {
				params = "yes"
			}
			rows = append(rows, []string{item.Name, item.Description, params})
		}
		return p.Table([]string{"Name", "Description", "Accepts parameters"}, rows)
	}

	return p.WritePretty(intelfacade.ListCapabilitiesPrettyText(items))
}
