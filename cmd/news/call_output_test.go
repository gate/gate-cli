package news

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gate/gate-cli/internal/intelfacade"
	"github.com/gate/gate-cli/internal/mcpclient"
	"github.com/gate/gate-cli/internal/output"
)

type fakeNewsCallService struct {
	result *mcpclient.CallResult
}

func (f *fakeNewsCallService) ListTools(ctx context.Context) ([]intelfacade.ToolSummary, *http.Response, error) {
	return nil, nil, nil
}
func (f *fakeNewsCallService) DescribeTool(ctx context.Context, name string) (*intelfacade.ToolSummary, *http.Response, error) {
	return &intelfacade.ToolSummary{Name: name}, nil, nil
}
func (f *fakeNewsCallService) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcpclient.CallResult, *http.Response, error) {
	return f.result, nil, nil
}

func TestRunNewsCall_JSONEnvelope(t *testing.T) {
	oldFactory, oldPrinter := newNewsService, getPrinter
	t.Cleanup(func() { newNewsService = oldFactory; getPrinter = oldPrinter })

	newNewsService = func(cmd *cobra.Command) (newsService, error) {
		return &fakeNewsCallService{result: &mcpclient.CallResult{
			Content: []mcpclient.ContentItem{{"type": "text", "text": `{"ok":true}`}},
		}}, nil
	}

	var out, errOut bytes.Buffer
	getPrinter = func(cmd *cobra.Command) *output.Printer {
		return output.NewWithStderr(&out, &errOut, output.FormatJSON)
	}
	cmd := &cobra.Command{Use: "call"}
	cmd.Flags().String("params", "", "")
	cmd.Flags().String("args-json", "", "")
	cmd.Flags().String("args-file", "", "")
	require.NoError(t, runNewsCallByName(cmd, "news_feed_search_news", map[string]struct{}{}))
	assert.Contains(t, out.String(), `"status": "success"`)
	assert.Contains(t, out.String(), `"tool_name": "news_feed_search_news"`)
	assert.Contains(t, out.String(), `"is_error": false`)
	assert.Contains(t, out.String(), `"data_source": "content"`)
	assert.Contains(t, out.String(), `"data":`)
	assert.Empty(t, errOut.String())
}
