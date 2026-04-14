package news

import (
	"context"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/cmdutil"
	"github.com/gate/gate-cli/internal/intelfacade"
	"github.com/gate/gate-cli/internal/mcpclient"
	"github.com/gate/gate-cli/internal/toolconfig"
)

type newsService interface {
	ListTools(ctx context.Context) ([]intelfacade.ToolSummary, *http.Response, error)
	DescribeTool(ctx context.Context, name string) (*intelfacade.ToolSummary, *http.Response, error)
	CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcpclient.CallResult, *http.Response, error)
}

var newNewsService = func(cmd *cobra.Command) (newsService, error) {
	debug, _ := cmd.Root().PersistentFlags().GetBool("debug")
	endpoint, err := toolconfig.Resolve(toolconfig.ResolveOptions{Backend: "news"})
	if err != nil {
		return nil, err
	}
	client := mcpclient.New(endpoint, mcpclient.WithDebug(debug))
	return intelfacade.NewNewsService(client), nil
}

var getPrinter = cmdutil.GetPrinter
