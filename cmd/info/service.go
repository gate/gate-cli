package info

import (
	"context"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/cmdutil"
	"github.com/gate/gate-cli/internal/intelfacade"
	"github.com/gate/gate-cli/internal/mcpclient"
	"github.com/gate/gate-cli/internal/toolconfig"
)

type infoService interface {
	ListTools(ctx context.Context) ([]intelfacade.ToolSummary, *http.Response, error)
	DescribeTool(ctx context.Context, name string) (*intelfacade.ToolSummary, *http.Response, error)
	CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcpclient.CallResult, *http.Response, error)
}

var newInfoService = func(cmd *cobra.Command) (infoService, error) {
	endpoint, err := toolconfig.Resolve(toolconfig.ResolveOptions{Backend: "info"})
	if err != nil {
		return nil, err
	}
	diag, tag := cmdutil.IntelMCPTransportDiag(cmd)
	client := mcpclient.New(endpoint, mcpclient.WithTransportDiag(diag, tag))
	return intelfacade.NewInfoService(client), nil
}

var getPrinter = cmdutil.GetPrinter
