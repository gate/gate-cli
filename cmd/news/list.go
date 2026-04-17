package news

import (
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/intelcmd"
	"github.com/gate/gate-cli/internal/intelfacade"
	"github.com/gate/gate-cli/internal/toolschema"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available News capabilities",
	RunE:  runNewsList,
}

func init() {
	Cmd.AddCommand(listCmd)
}

var saveNewsSchemaCache = toolschema.SaveCache

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
	_ = saveNewsSchemaCache("news", toNewsSchemaSummaries(items))

	return intelcmd.RenderToolList(p, items)
}

func toNewsSchemaSummaries(items []intelfacade.ToolSummary) []toolschema.ToolSummary {
	out := make([]toolschema.ToolSummary, 0, len(items))
	for _, item := range items {
		out = append(out, toolschema.ToolSummary{
			Name:           item.Name,
			Description:    item.Description,
			HasInputSchema: item.HasInputSchema,
			InputSchema:    item.InputSchema,
		})
	}
	return out
}
