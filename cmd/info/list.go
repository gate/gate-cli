package info

import (
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/intelcmd"
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
		return intelcmd.FailIntelClientInit(p, err, "info", "list", "")
	}

	items, httpResp, err := svc.ListTools(cmd.Context())
	if err != nil {
		return intelcmd.FailListTransport(p, err, httpResp, "info")
	}

	return intelcmd.RenderToolList(p, items)
}
