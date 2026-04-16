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

type fakeInfoService struct {
	describe *intelfacade.ToolSummary
	call     *mcpclient.CallResult
	err      error
}

func (f *fakeInfoService) ListTools(ctx context.Context) ([]intelfacade.ToolSummary, *http.Response, error) {
	return nil, nil, nil
}
func (f *fakeInfoService) DescribeTool(ctx context.Context, name string) (*intelfacade.ToolSummary, *http.Response, error) {
	return f.describe, nil, f.err
}
func (f *fakeInfoService) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcpclient.CallResult, *http.Response, error) {
	return f.call, nil, f.err
}

func TestRunInfoDescribeJSON(t *testing.T) {
	oldFactory, oldPrinter := newInfoService, getPrinter
	t.Cleanup(func() { newInfoService = oldFactory; getPrinter = oldPrinter })

	newInfoService = func(cmd *cobra.Command) (infoService, error) {
		return &fakeInfoService{describe: &intelfacade.ToolSummary{Name: "info_coin_get_coin_info"}}, nil
	}
	var out, errOut bytes.Buffer
	getPrinter = func(cmd *cobra.Command) *output.Printer {
		return output.NewWithStderr(&out, &errOut, output.FormatJSON)
	}

	cmd := &cobra.Command{Use: "describe"}
	cmd.Flags().String("name", "", "")
	_ = cmd.Flags().Set("name", "info_coin_get_coin_info")
	require.NoError(t, runInfoDescribe(cmd, nil))
	assert.Contains(t, out.String(), "info_coin_get_coin_info")
}

func TestRunInfoDescribePrettySections(t *testing.T) {
	oldFactory, oldPrinter := newInfoService, getPrinter
	t.Cleanup(func() { newInfoService = oldFactory; getPrinter = oldPrinter })

	newInfoService = func(cmd *cobra.Command) (infoService, error) {
		return &fakeInfoService{describe: &intelfacade.ToolSummary{
			Name:           "info_coin_get_coin_info",
			Description:    "Coin profile.",
			HasInputSchema: true,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"symbol": map[string]interface{}{"type": "string"},
				},
				"required": []interface{}{"symbol"},
			},
		}}, nil
	}
	var out, errOut bytes.Buffer
	getPrinter = func(cmd *cobra.Command) *output.Printer {
		return output.NewWithStderr(&out, &errOut, output.FormatPretty)
	}

	cmd := &cobra.Command{Use: "describe"}
	cmd.Flags().String("name", "", "")
	_ = cmd.Flags().Set("name", "info_coin_get_coin_info")
	require.NoError(t, runInfoDescribe(cmd, nil))
	assert.Contains(t, out.String(), "Overview")
	assert.Contains(t, out.String(), "Parameters")
	assert.Contains(t, out.String(), "symbol")
	assert.NotContains(t, out.String(), "input_schema")
	assert.Empty(t, errOut.String())
}
