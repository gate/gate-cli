package info

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

type fakeInfoCallService struct {
	result *mcpclient.CallResult
}

func (f *fakeInfoCallService) ListTools(ctx context.Context) ([]intelfacade.ToolSummary, *http.Response, error) {
	return nil, nil, nil
}
func (f *fakeInfoCallService) DescribeTool(ctx context.Context, name string) (*intelfacade.ToolSummary, *http.Response, error) {
	return &intelfacade.ToolSummary{Name: name}, nil, nil
}
func (f *fakeInfoCallService) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcpclient.CallResult, *http.Response, error) {
	return f.result, nil, nil
}

func TestRunInfoCall_JSONEnvelope(t *testing.T) {
	oldFactory, oldPrinter := newInfoService, getPrinter
	t.Cleanup(func() { newInfoService = oldFactory; getPrinter = oldPrinter })

	newInfoService = func(cmd *cobra.Command) (infoService, error) {
		return &fakeInfoCallService{result: &mcpclient.CallResult{
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
	require.NoError(t, runInfoCallByName(cmd, "info_coin_get_coin_info", map[string]struct{}{}))
	assert.Contains(t, out.String(), `"status": "success"`)
	assert.Contains(t, out.String(), `"tool_name": "info_coin_get_coin_info"`)
	assert.Contains(t, out.String(), `"is_error": false`)
	assert.Contains(t, out.String(), `"data_source": "content"`)
	assert.Contains(t, out.String(), `"data":`)
	assert.Empty(t, errOut.String())
}
