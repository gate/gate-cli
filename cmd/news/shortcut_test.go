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

type fakeNewsShortcutService struct{}

func (f *fakeNewsShortcutService) ListTools(ctx context.Context) ([]intelfacade.ToolSummary, *http.Response, error) {
	return nil, nil, nil
}
func (f *fakeNewsShortcutService) DescribeTool(ctx context.Context, name string) (*intelfacade.ToolSummary, *http.Response, error) {
	return &intelfacade.ToolSummary{Name: name}, nil, nil
}
func (f *fakeNewsShortcutService) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcpclient.CallResult, *http.Response, error) {
	if name == "news_events_get_latest_events" {
		return &mcpclient.CallResult{
			StructuredContent: map[string]interface{}{
				"items": []interface{}{map[string]interface{}{"event_id": "evt-1"}},
			},
		}, nil, nil
	}
	return &mcpclient.CallResult{StructuredContent: map[string]interface{}{"tool": name}}, nil, nil
}

func TestNewsShortcutEventExplain(t *testing.T) {
	oldFactory, oldPrinter := newNewsService, getPrinter
	t.Cleanup(func() { newNewsService = oldFactory; getPrinter = oldPrinter })
	newNewsService = func(cmd *cobra.Command) (newsService, error) { return &fakeNewsShortcutService{}, nil }

	var out, errOut bytes.Buffer
	getPrinter = func(cmd *cobra.Command) *output.Printer {
		return output.NewWithStderr(&out, &errOut, output.FormatJSON)
	}

	cmd := newNewsEventExplainCmd()
	require.NoError(t, cmd.Flags().Set("coin", "BTC"))
	require.NoError(t, cmd.RunE(cmd, nil))
	assert.Contains(t, out.String(), `"event_summary"`)
	assert.Contains(t, out.String(), `"source_coverage"`)
	assert.Empty(t, errOut.String())
}

func TestNewsShortcutBriefAllowsOneOfPrimarySources(t *testing.T) {
	oldFactory, oldPrinter := newNewsService, getPrinter
	t.Cleanup(func() { newNewsService = oldFactory; getPrinter = oldPrinter })
	newNewsService = func(cmd *cobra.Command) (newsService, error) { return &fakeNewsShortcutService{}, nil }

	var out, errOut bytes.Buffer
	getPrinter = func(cmd *cobra.Command) *output.Printer {
		return output.NewWithStderr(&out, &errOut, output.FormatJSON)
	}

	cmd := newNewsBriefCmd()
	require.NoError(t, cmd.Flags().Set("coin", "BTC"))
	require.NoError(t, cmd.RunE(cmd, nil))
	assert.Contains(t, out.String(), `"missing_sections"`)
	assert.Empty(t, errOut.String())
}
