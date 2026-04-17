package intelfacade

import (
	"context"
	"net/http"

	"github.com/gate/gate-cli/internal/mcpclient"
)

type infoToolLister interface {
	ListTools(ctx context.Context) ([]mcpclient.Tool, *http.Response, error)
	DescribeTool(ctx context.Context, name string) (*mcpclient.Tool, *http.Response, error)
	CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcpclient.CallResult, *http.Response, error)
}

// InfoService hides transport/protocol details from info command handlers.
type InfoService struct {
	*Service
}

// NewInfoService creates an Info facade service.
func NewInfoService(client infoToolLister) *InfoService {
	return &InfoService{Service: newService(client)}
}
