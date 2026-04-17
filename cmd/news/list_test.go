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

	"github.com/gate/gate-cli/internal/exitcode"
	"github.com/gate/gate-cli/internal/intelfacade"
	"github.com/gate/gate-cli/internal/mcpclient"
	"github.com/gate/gate-cli/internal/output"
	"github.com/gate/gate-cli/internal/toolschema"
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
	root.PersistentFlags().Bool("verbose", false, "Verbose")
	root.AddCommand(Cmd)
	list := listCmd
	list.SetContext(context.Background())
	return list
}

func TestRunNewsListJSON(t *testing.T) {
	oldFactory := newNewsService
	oldPrinter := getPrinter
	oldSaveCache := saveNewsSchemaCache
	t.Cleanup(func() { newNewsService = oldFactory })
	t.Cleanup(func() { getPrinter = oldPrinter })
	t.Cleanup(func() { saveNewsSchemaCache = oldSaveCache })

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
	saveNewsSchemaCache = func(backend string, items []toolschema.ToolSummary) error {
		return nil
	}

	err := runNewsList(cmd, nil)
	require.NoError(t, err)
	assert.Contains(t, out.String(), "news_feed_search_news")
	assert.Empty(t, errOut.String())
}

func TestRunNewsListSaveCacheFailureIgnored(t *testing.T) {
	oldFactory := newNewsService
	oldPrinter := getPrinter
	oldSaveCache := saveNewsSchemaCache
	t.Cleanup(func() { newNewsService = oldFactory })
	t.Cleanup(func() { getPrinter = oldPrinter })
	t.Cleanup(func() { saveNewsSchemaCache = oldSaveCache })

	newNewsService = func(cmd *cobra.Command) (newsService, error) {
		return &fakeNewsLister{
			items: []intelfacade.ToolSummary{
				{Name: "news_feed_search_news", Description: "search", HasInputSchema: true},
			},
		}, nil
	}
	saveNewsSchemaCache = func(backend string, items []toolschema.ToolSummary) error {
		return errors.New("cache write failed")
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
	oldSaveCache := saveNewsSchemaCache
	t.Cleanup(func() { newNewsService = oldFactory })
	t.Cleanup(func() { getPrinter = oldPrinter })
	t.Cleanup(func() { saveNewsSchemaCache = oldSaveCache })

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
	saveNewsSchemaCache = func(backend string, items []toolschema.ToolSummary) error {
		return nil
	}

	err := runNewsList(cmd, nil)
	require.Error(t, err)
	var coded *exitcode.Error
	require.True(t, errors.As(err, &coded))
	assert.Equal(t, 1, coded.Code)
	assert.Empty(t, out.String())
	assert.Contains(t, errOut.String(), `"error"`)
	assert.Contains(t, errOut.String(), "list failed")
}

func TestRunNewsListPrettySegmented(t *testing.T) {
	oldFactory := newNewsService
	oldPrinter := getPrinter
	oldSaveCache := saveNewsSchemaCache
	t.Cleanup(func() { newNewsService = oldFactory })
	t.Cleanup(func() { getPrinter = oldPrinter })
	t.Cleanup(func() { saveNewsSchemaCache = oldSaveCache })

	newNewsService = func(cmd *cobra.Command) (newsService, error) {
		return &fakeNewsLister{
			items: []intelfacade.ToolSummary{
				{Name: "news_feed_search_news", Description: "Search", HasInputSchema: true},
			},
		}, nil
	}

	cmd := newTestCmd("pretty")
	var out bytes.Buffer
	var errOut bytes.Buffer
	getPrinter = func(cmd *cobra.Command) *output.Printer {
		return output.NewWithStderr(&out, &errOut, output.FormatPretty)
	}
	saveNewsSchemaCache = func(backend string, items []toolschema.ToolSummary) error {
		return nil
	}

	require.NoError(t, runNewsList(cmd, nil))
	assert.Contains(t, out.String(), "Capabilities")
	assert.Contains(t, out.String(), "news_feed_search_news")
	assert.NotContains(t, out.String(), "HasInputSchema")
	assert.Empty(t, errOut.String())
}

func TestRunNewsListTableColumns(t *testing.T) {
	oldFactory := newNewsService
	oldPrinter := getPrinter
	oldSaveCache := saveNewsSchemaCache
	t.Cleanup(func() { newNewsService = oldFactory })
	t.Cleanup(func() { getPrinter = oldPrinter })
	t.Cleanup(func() { saveNewsSchemaCache = oldSaveCache })

	newNewsService = func(cmd *cobra.Command) (newsService, error) {
		return &fakeNewsLister{
			items: []intelfacade.ToolSummary{
				{Name: "news_feed_search_news", Description: "Search", HasInputSchema: false},
			},
		}, nil
	}

	cmd := newTestCmd("table")
	var out bytes.Buffer
	var errOut bytes.Buffer
	getPrinter = func(cmd *cobra.Command) *output.Printer {
		return output.NewWithStderr(&out, &errOut, output.FormatTable)
	}
	saveNewsSchemaCache = func(backend string, items []toolschema.ToolSummary) error {
		return nil
	}

	require.NoError(t, runNewsList(cmd, nil))
	assert.Contains(t, out.String(), "Accepts parameters")
	assert.Contains(t, out.String(), "news_feed_search_news")
	assert.Empty(t, errOut.String())
}
