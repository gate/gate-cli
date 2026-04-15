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

func TestRunInfoCall_IsErrorPrintsStderrOnly(t *testing.T) {
	oldFactory, oldPrinter := newInfoService, getPrinter
	t.Cleanup(func() { newInfoService = oldFactory; getPrinter = oldPrinter })

	newInfoService = func(cmd *cobra.Command) (infoService, error) {
		return &fakeInfoCallService{result: &mcpclient.CallResult{
			IsError: true,
			Raw:     map[string]interface{}{"reason": "upstream_failed"},
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
	assert.Empty(t, out.String())
	assert.Contains(t, errOut.String(), `"error":`)
	assert.Contains(t, errOut.String(), `"label": "INTEL_RESULT_ERROR"`)
	assert.Contains(t, errOut.String(), `"tool_name": "info_coin_get_coin_info"`)
}

func TestRunInfoCall_IsErrorUnaffectedByMaxOutputBytes(t *testing.T) {
	oldFactory, oldPrinter := newInfoService, getPrinter
	t.Cleanup(func() { newInfoService = oldFactory; getPrinter = oldPrinter })

	newInfoService = func(cmd *cobra.Command) (infoService, error) {
		return &fakeInfoCallService{result: &mcpclient.CallResult{
			IsError: true,
			Raw:     map[string]interface{}{"reason": "upstream_failed"},
		}}, nil
	}

	var out, errOut bytes.Buffer
	getPrinter = func(cmd *cobra.Command) *output.Printer {
		return output.NewWithStderr(&out, &errOut, output.FormatJSON)
	}

	root := &cobra.Command{Use: "gate-cli"}
	root.PersistentFlags().Int64("max-output-bytes", 8, "")
	cmd := &cobra.Command{Use: "call"}
	cmd.Flags().String("params", "", "")
	cmd.Flags().String("args-json", "", "")
	cmd.Flags().String("args-file", "", "")
	root.AddCommand(cmd)

	require.NoError(t, runInfoCallByName(cmd, "info_coin_get_coin_info", map[string]struct{}{}))
	assert.Empty(t, out.String())
	assert.Contains(t, errOut.String(), `"error":`)
	assert.Contains(t, errOut.String(), `"label": "INTEL_RESULT_ERROR"`)
	assert.NotContains(t, errOut.String(), `"truncated"`)
}

func TestRunInfoCall_PrettyIsErrorPrintsReadableStderrOnly(t *testing.T) {
	oldFactory, oldPrinter := newInfoService, getPrinter
	t.Cleanup(func() { newInfoService = oldFactory; getPrinter = oldPrinter })

	newInfoService = func(cmd *cobra.Command) (infoService, error) {
		return &fakeInfoCallService{result: &mcpclient.CallResult{
			IsError: true,
		}}, nil
	}

	var out, errOut bytes.Buffer
	getPrinter = func(cmd *cobra.Command) *output.Printer {
		return output.NewWithStderr(&out, &errOut, output.FormatPretty)
	}
	cmd := &cobra.Command{Use: "call"}
	cmd.Flags().String("params", "", "")
	cmd.Flags().String("args-json", "", "")
	cmd.Flags().String("args-file", "", "")

	require.NoError(t, runInfoCallByName(cmd, "info_coin_get_coin_info", map[string]struct{}{}))
	assert.Empty(t, out.String())
	assert.Contains(t, errOut.String(), "Error [502 INTEL_RESULT_ERROR]: tool returned isError=true")
	assert.Contains(t, errOut.String(), "Tool: info_coin_get_coin_info")
}
