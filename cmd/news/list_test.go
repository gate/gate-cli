package news

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gate/gate-cli/internal/intelfacade"
	"github.com/gate/gate-cli/internal/mcpclient"
	"github.com/gate/gate-cli/internal/output"
)

type fakeNewsLister struct {
	items []intelfacade.ToolSummary
	resp  *http.Response
	err   error
}

func (f *fakeNewsLister) ListTools(ctx context.Context) ([]intelfacade.ToolSummary, *http.Response, error) {
	return f.items, f.resp, f.err
}

func (f *fakeNewsLister) DescribeTool(ctx context.Context, name string) (*intelfacade.ToolSummary, *http.Response, error) {
	return nil, f.resp, errors.New("not implemented")
}

func (f *fakeNewsLister) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcpclient.CallResult, *http.Response, error) {
	return nil, f.resp, errors.New("not implemented")
}

func newTestCmd(format string) *cobra.Command {
	root := &cobra.Command{Use: "gate-cli"}
	root.PersistentFlags().String("format", format, "Output format")
	root.PersistentFlags().Bool("debug", false, "Debug")
	root.AddCommand(Cmd)
	list := listCmd
	list.SetContext(context.Background())
	return list
}

func TestRunNewsListJSON(t *testing.T) {
	oldFactory := newNewsService
	oldPrinter := getPrinter
	t.Cleanup(func() { newNewsService = oldFactory })
	t.Cleanup(func() { getPrinter = oldPrinter })

	newNewsService = func(cmd *cobra.Command) (newsService, error) {
		return &fakeNewsLister{
			items: []intelfacade.ToolSummary{
				{Name: "news_feed_search_news", Description: "search", HasInputSchema: true},
			},
		}, nil
	}

	cmd := newTestCmd("json")
	var out bytes.Buffer
	var errOut bytes.Buffer
	getPrinter = func(cmd *cobra.Command) *output.Printer {
		return output.NewWithStderr(&out, &errOut, output.FormatJSON)
	}

	err := runNewsList(cmd, nil)
	require.NoError(t, err)
	assert.Contains(t, out.String(), "news_feed_search_news")
	assert.Empty(t, errOut.String())
}

func TestRunNewsListErrorGoesStderr(t *testing.T) {
	oldFactory := newNewsService
	oldPrinter := getPrinter
	t.Cleanup(func() { newNewsService = oldFactory })
	t.Cleanup(func() { getPrinter = oldPrinter })

	newNewsService = func(cmd *cobra.Command) (newsService, error) {
		return &fakeNewsLister{
			err: errors.New("list failed"),
		}, nil
	}

	cmd := newTestCmd("json")
	var out bytes.Buffer
	var errOut bytes.Buffer
	getPrinter = func(cmd *cobra.Command) *output.Printer {
		return output.NewWithStderr(&out, &errOut, output.FormatJSON)
	}

	err := runNewsList(cmd, nil)
	require.NoError(t, err)
	assert.Empty(t, out.String())
	assert.Contains(t, errOut.String(), `"error"`)
	assert.Contains(t, errOut.String(), "list failed")
}
