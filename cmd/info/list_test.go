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

	"github.com/gate/gate-cli/internal/exitcode"
	"github.com/gate/gate-cli/internal/intelfacade"
	"github.com/gate/gate-cli/internal/mcpclient"
	"github.com/gate/gate-cli/internal/output"
	"github.com/gate/gate-cli/internal/toolschema"
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
	root.PersistentFlags().Bool("verbose", false, "Verbose")
	root.AddCommand(Cmd)
	list := listCmd
	list.SetContext(context.Background())
	return list
}

func TestRunInfoListJSON(t *testing.T) {
	oldFactory := newInfoService
	oldPrinter := getPrinter
	oldSaveCache := saveInfoSchemaCache
	t.Cleanup(func() { newInfoService = oldFactory })
	t.Cleanup(func() { getPrinter = oldPrinter })
	t.Cleanup(func() { saveInfoSchemaCache = oldSaveCache })

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
	saveInfoSchemaCache = func(backend string, items []toolschema.ToolSummary) error {
		return nil
	}

	err := runInfoList(cmd, nil)
	require.NoError(t, err)
	assert.Contains(t, out.String(), "info_coin_get_coin_info")
	assert.Empty(t, errOut.String())
}

func TestRunInfoListSaveCacheFailureIgnored(t *testing.T) {
	oldFactory := newInfoService
	oldPrinter := getPrinter
	oldSaveCache := saveInfoSchemaCache
	t.Cleanup(func() { newInfoService = oldFactory })
	t.Cleanup(func() { getPrinter = oldPrinter })
	t.Cleanup(func() { saveInfoSchemaCache = oldSaveCache })

	newInfoService = func(cmd *cobra.Command) (infoService, error) {
		return &fakeInfoLister{
			items: []intelfacade.ToolSummary{
				{Name: "info_coin_get_coin_info", Description: "coin info", HasInputSchema: true},
			},
		}, nil
	}
	saveInfoSchemaCache = func(backend string, items []toolschema.ToolSummary) error {
		return errors.New("cache write failed")
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
	oldSaveCache := saveInfoSchemaCache
	t.Cleanup(func() { newInfoService = oldFactory })
	t.Cleanup(func() { getPrinter = oldPrinter })
	t.Cleanup(func() { saveInfoSchemaCache = oldSaveCache })

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
	saveInfoSchemaCache = func(backend string, items []toolschema.ToolSummary) error {
		return nil
	}

	err := runInfoList(cmd, nil)
	require.Error(t, err)
	var coded *exitcode.Error
	require.True(t, errors.As(err, &coded))
	assert.Equal(t, 1, coded.Code)
	assert.Empty(t, out.String())
	assert.Contains(t, errOut.String(), `"error"`)
	assert.Contains(t, errOut.String(), "list failed")
}

func TestRunInfoListPrettySegmented(t *testing.T) {
	oldFactory := newInfoService
	oldPrinter := getPrinter
	oldSaveCache := saveInfoSchemaCache
	t.Cleanup(func() { newInfoService = oldFactory })
	t.Cleanup(func() { getPrinter = oldPrinter })
	t.Cleanup(func() { saveInfoSchemaCache = oldSaveCache })

	newInfoService = func(cmd *cobra.Command) (infoService, error) {
		return &fakeInfoLister{
			items: []intelfacade.ToolSummary{
				{Name: "info_coin_get_coin_info", Description: "Coin", HasInputSchema: true},
			},
		}, nil
	}

	cmd := newTestCmd("pretty")
	var out bytes.Buffer
	var errOut bytes.Buffer
	getPrinter = func(cmd *cobra.Command) *output.Printer {
		return output.NewWithStderr(&out, &errOut, output.FormatPretty)
	}
	saveInfoSchemaCache = func(backend string, items []toolschema.ToolSummary) error {
		return nil
	}

	require.NoError(t, runInfoList(cmd, nil))
	assert.Contains(t, out.String(), "Capabilities")
	assert.Contains(t, out.String(), "info_coin_get_coin_info")
	assert.Contains(t, out.String(), "Accepts parameters: yes")
	assert.NotContains(t, out.String(), "HasInputSchema")
	assert.Empty(t, errOut.String())
}

func TestRunInfoListTableColumns(t *testing.T) {
	oldFactory := newInfoService
	oldPrinter := getPrinter
	oldSaveCache := saveInfoSchemaCache
	t.Cleanup(func() { newInfoService = oldFactory })
	t.Cleanup(func() { getPrinter = oldPrinter })
	t.Cleanup(func() { saveInfoSchemaCache = oldSaveCache })

	newInfoService = func(cmd *cobra.Command) (infoService, error) {
		return &fakeInfoLister{
			items: []intelfacade.ToolSummary{
				{Name: "info_coin_get_coin_info", Description: "Coin", HasInputSchema: false},
			},
		}, nil
	}

	cmd := newTestCmd("table")
	var out bytes.Buffer
	var errOut bytes.Buffer
	getPrinter = func(cmd *cobra.Command) *output.Printer {
		return output.NewWithStderr(&out, &errOut, output.FormatTable)
	}
	saveInfoSchemaCache = func(backend string, items []toolschema.ToolSummary) error {
		return nil
	}

	require.NoError(t, runInfoList(cmd, nil))
	assert.Contains(t, out.String(), "Accepts parameters")
	assert.Contains(t, out.String(), "info_coin_get_coin_info")
	assert.Empty(t, errOut.String())
}
