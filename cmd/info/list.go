package info

import (
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/intelcmd"
	"github.com/gate/gate-cli/internal/intelfacade"
	"github.com/gate/gate-cli/internal/toolschema"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available Info capabilities",
	RunE:  runInfoList,
}

func init() {
	Cmd.AddCommand(listCmd)
}

var saveInfoSchemaCache = toolschema.SaveCache

func runInfoList(cmd *cobra.Command, args []string) error {
	p := getPrinter(cmd)
	svc, err := newInfoService(cmd)
	if err != nil {
		return intelcmd.FailIntelClientInit(p, err, "info", "list", "")
	}

	items, httpResp, err := svc.ListTools(cmd.Context())
	if err != nil {
		return intelcmd.FailListTransport(p, err, httpResp, "info")
	}
	_ = saveInfoSchemaCache("info", toInfoSchemaSummaries(items))

	return intelcmd.RenderToolList(p, items)
}

func toInfoSchemaSummaries(items []intelfacade.ToolSummary) []toolschema.ToolSummary {
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
