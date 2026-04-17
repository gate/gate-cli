package news

import (
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/intelcmd"
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
		return intelcmd.FailIntelClientInit(p, err, "news", "list", "")
	}

	items, httpResp, err := svc.ListTools(cmd.Context())
	if err != nil {
		return intelcmd.FailListTransport(p, err, httpResp, "news")
	}

	return intelcmd.RenderToolList(p, items)
}
