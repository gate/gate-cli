package info

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

type fakeInfoLister struct {
	items []intelfacade.ToolSummary
	resp  *http.Response
	err   error
}

func (f *fakeInfoLister) ListTools(ctx context.Context) ([]intelfacade.ToolSummary, *http.Response, error) {
	return f.items, f.resp, f.err
}

func (f *fakeInfoLister) DescribeTool(ctx context.Context, name string) (*intelfacade.ToolSummary, *http.Response, error) {
	return nil, f.resp, errors.New("not implemented")
}

func (f *fakeInfoLister) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcpclient.CallResult, *http.Response, error) {
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

func TestRunInfoListJSON(t *testing.T) {
	oldFactory := newInfoService
	oldPrinter := getPrinter
	t.Cleanup(func() { newInfoService = oldFactory })
	t.Cleanup(func() { getPrinter = oldPrinter })

	newInfoService = func(cmd *cobra.Command) (infoService, error) {
		return &fakeInfoLister{
			items: []intelfacade.ToolSummary{
				{Name: "info_coin_get_coin_info", Description: "coin info", HasInputSchema: true},
			},
		}, nil
	}

	cmd := newTestCmd("json")
	var out bytes.Buffer
	var errOut bytes.Buffer
	getPrinter = func(cmd *cobra.Command) *output.Printer {
		return output.NewWithStderr(&out, &errOut, output.FormatJSON)
	}

	err := runInfoList(cmd, nil)
	require.NoError(t, err)
	assert.Contains(t, out.String(), "info_coin_get_coin_info")
	assert.Empty(t, errOut.String())
}

func TestRunInfoListErrorGoesStderr(t *testing.T) {
	oldFactory := newInfoService
	oldPrinter := getPrinter
	t.Cleanup(func() { newInfoService = oldFactory })
	t.Cleanup(func() { getPrinter = oldPrinter })

	newInfoService = func(cmd *cobra.Command) (infoService, error) {
		return &fakeInfoLister{
			err: errors.New("list failed"),
		}, nil
	}

	cmd := newTestCmd("json")
	var out bytes.Buffer
	var errOut bytes.Buffer
	getPrinter = func(cmd *cobra.Command) *output.Printer {
		return output.NewWithStderr(&out, &errOut, output.FormatJSON)
	}

	err := runInfoList(cmd, nil)
	require.NoError(t, err)
	assert.Empty(t, out.String())
	assert.Contains(t, errOut.String(), `"error"`)
	assert.Contains(t, errOut.String(), "list failed")
}
