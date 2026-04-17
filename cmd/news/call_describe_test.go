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

type fakeNewsService struct {
	describe *intelfacade.ToolSummary
	call     *mcpclient.CallResult
	err      error
}

func (f *fakeNewsService) ListTools(ctx context.Context) ([]intelfacade.ToolSummary, *http.Response, error) {
	return nil, nil, nil
}
func (f *fakeNewsService) DescribeTool(ctx context.Context, name string) (*intelfacade.ToolSummary, *http.Response, error) {
	return f.describe, nil, f.err
}
func (f *fakeNewsService) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcpclient.CallResult, *http.Response, error) {
	return f.call, nil, f.err
}

func TestRunNewsDescribeJSON(t *testing.T) {
	oldFactory, oldPrinter := newNewsService, getPrinter
	t.Cleanup(func() { newNewsService = oldFactory; getPrinter = oldPrinter })

	newNewsService = func(cmd *cobra.Command) (newsService, error) {
		return &fakeNewsService{describe: &intelfacade.ToolSummary{Name: "news_feed_search_news"}}, nil
	}
	var out, errOut bytes.Buffer
	getPrinter = func(cmd *cobra.Command) *output.Printer {
		return output.NewWithStderr(&out, &errOut, output.FormatJSON)
	}

	cmd := &cobra.Command{Use: "describe"}
	cmd.Flags().String("name", "", "")
	_ = cmd.Flags().Set("name", "news_feed_search_news")
	require.NoError(t, runNewsDescribe(cmd, nil))
	assert.Contains(t, out.String(), "news_feed_search_news")
}

func TestRunNewsDescribePrettySections(t *testing.T) {
	oldFactory, oldPrinter := newNewsService, getPrinter
	t.Cleanup(func() { newNewsService = oldFactory; getPrinter = oldPrinter })

	newNewsService = func(cmd *cobra.Command) (newsService, error) {
		return &fakeNewsService{describe: &intelfacade.ToolSummary{
			Name:           "news_feed_search_news",
			Description:    "Search news.",
			HasInputSchema: true,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"coin": map[string]interface{}{"type": "string"},
				},
				"required": []interface{}{"coin"},
			},
		}}, nil
	}
	var out, errOut bytes.Buffer
	getPrinter = func(cmd *cobra.Command) *output.Printer {
		return output.NewWithStderr(&out, &errOut, output.FormatPretty)
	}

	cmd := &cobra.Command{Use: "describe"}
	cmd.Flags().String("name", "", "")
	_ = cmd.Flags().Set("name", "news_feed_search_news")
	require.NoError(t, runNewsDescribe(cmd, nil))
	assert.Contains(t, out.String(), "Overview")
	assert.Contains(t, out.String(), "Parameters")
	assert.Contains(t, out.String(), "coin")
	assert.NotContains(t, out.String(), "input_schema")
	assert.Empty(t, errOut.String())
}
