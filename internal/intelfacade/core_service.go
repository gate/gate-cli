package intelfacade

import (
	"context"
	"net/http"

	"github.com/gate/gate-cli/internal/mcpclient"
)

type mcpToolClient interface {
	ListTools(ctx context.Context) ([]mcpclient.Tool, *http.Response, error)
	DescribeTool(ctx context.Context, name string) (*mcpclient.Tool, *http.Response, error)
	CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcpclient.CallResult, *http.Response, error)
}

// Service provides reusable facade operations for one backend.
type Service struct {
	client mcpToolClient
}

func newService(client mcpToolClient) *Service {
	return &Service{client: client}
}

func (s *Service) ListTools(ctx context.Context) ([]ToolSummary, *http.Response, error) {
	tools, resp, err := s.client.ListTools(ctx)
	if err != nil {
		return nil, resp, err
	}
	out := make([]ToolSummary, 0, len(tools))
	for _, t := range tools {
		out = append(out, toToolSummary(t))
	}
	return out, resp, nil
}

func (s *Service) DescribeTool(ctx context.Context, name string) (*ToolSummary, *http.Response, error) {
	tool, resp, err := s.client.DescribeTool(ctx, name)
	if err != nil {
		return nil, resp, err
	}
	if tool == nil {
		return nil, resp, nil
	}
	summary := toToolSummary(*tool)
	return &summary, resp, nil
}

func (s *Service) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcpclient.CallResult, *http.Response, error) {
	return s.client.CallTool(ctx, name, arguments)
}

func toToolSummary(t mcpclient.Tool) ToolSummary {
	return ToolSummary{
		Name:           t.Name,
		Description:    t.Description,
		HasInputSchema: t.InputSchema != nil,
		InputSchema:    t.InputSchema,
	}
}
