package intelfacade

import (
	"context"
	"net/http"

	"github.com/gate/gate-cli/internal/mcpclient"
)

// ToolSummary is the CLI-friendly projection of News capability metadata.
type ToolSummary struct {
	Name           string      `json:"name"`
	Description    string      `json:"description,omitempty"`
	HasInputSchema bool        `json:"has_input_schema"`
	InputSchema    interface{} `json:"input_schema,omitempty"`
}

type newsToolLister interface {
	ListTools(ctx context.Context) ([]mcpclient.Tool, *http.Response, error)
	DescribeTool(ctx context.Context, name string) (*mcpclient.Tool, *http.Response, error)
	CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcpclient.CallResult, *http.Response, error)
}

// NewsService hides transport/protocol details from command handlers.
type NewsService struct {
	*Service
}

// NewNewsService creates a News facade service.
func NewNewsService(client newsToolLister) *NewsService {
	return &NewsService{Service: newService(client)}
}
