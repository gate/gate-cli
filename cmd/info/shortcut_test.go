package info

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gate/gate-cli/internal/intelfacade"
	"github.com/gate/gate-cli/internal/mcpclient"
	"github.com/gate/gate-cli/internal/output"
)

// tokenRiskRecordingService records tools/call order for +token-risk contract tests.
type tokenRiskRecordingService struct {
	calls        []string
	failSecurity bool
	failCoin     bool
	nativeCoin   bool
}

func (s *tokenRiskRecordingService) ListTools(ctx context.Context) ([]intelfacade.ToolSummary, *http.Response, error) {
	return nil, nil, nil
}
func (s *tokenRiskRecordingService) DescribeTool(ctx context.Context, name string) (*intelfacade.ToolSummary, *http.Response, error) {
	return &intelfacade.ToolSummary{Name: name}, nil, nil
}
func (s *tokenRiskRecordingService) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcpclient.CallResult, *http.Response, error) {
	s.calls = append(s.calls, name)
	switch name {
	case "info_compliance_check_token_security":
		if s.failSecurity {
			return nil, nil, errors.New("security check failed")
		}
		return &mcpclient.CallResult{StructuredContent: map[string]interface{}{"risk_level": "low"}}, nil, nil
	case "info_coin_get_coin_info":
		if s.failCoin {
			return nil, nil, errors.New("coin lookup failed")
		}
		if s.nativeCoin {
			return &mcpclient.CallResult{StructuredContent: map[string]interface{}{
				"query": "BTC",
				"items": []interface{}{
					map[string]interface{}{
						"symbol":           "BTC",
						"contract_address": "",
						"chain":            []interface{}{"botanix"},
					},
				},
			}}, nil, nil
		}
		content := map[string]interface{}{"tool": name}
		q, _ := arguments["query"].(string)
		if strings.TrimSpace(q) == "" {
			q, _ = arguments["symbol"].(string)
		}
		if strings.TrimSpace(q) != "" {
			content["canonical_contract"] = "0xresolved"
			content["canonical_chain"] = "eth"
		}
		return &mcpclient.CallResult{StructuredContent: content}, nil, nil
	default:
		return &mcpclient.CallResult{StructuredContent: map[string]interface{}{"tool": name}}, nil, nil
	}
}

func runTokenRiskShortcut(t *testing.T, svc infoService, setFlags func(*cobra.Command)) (stdout, stderr string, err error) {
	t.Helper()
	oldFactory, oldPrinter := newInfoService, getPrinter
	t.Cleanup(func() { newInfoService = oldFactory; getPrinter = oldPrinter })
	newInfoService = func(cmd *cobra.Command) (infoService, error) { return svc, nil }

	var out, errOut bytes.Buffer
	getPrinter = func(cmd *cobra.Command) *output.Printer {
		return output.NewWithStderr(&out, &errOut, output.FormatJSON)
	}
	cmd := newInfoTokenRiskCmd()
	setFlags(cmd)
	err = cmd.RunE(cmd, nil)
	return out.String(), errOut.String(), err
}

type fakeInfoShortcutService struct{}

func (f *fakeInfoShortcutService) ListTools(ctx context.Context) ([]intelfacade.ToolSummary, *http.Response, error) {
	return nil, nil, nil
}
func (f *fakeInfoShortcutService) DescribeTool(ctx context.Context, name string) (*intelfacade.ToolSummary, *http.Response, error) {
	return &intelfacade.ToolSummary{Name: name}, nil, nil
}
func (f *fakeInfoShortcutService) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcpclient.CallResult, *http.Response, error) {
	return &mcpclient.CallResult{StructuredContent: map[string]interface{}{"tool": name}}, nil, nil
}

func TestInfoShortcutCoinOverviewMissingSymbolJSON(t *testing.T) {
	oldFactory, oldPrinter := newInfoService, getPrinter
	t.Cleanup(func() { newInfoService = oldFactory; getPrinter = oldPrinter })
	newInfoService = func(cmd *cobra.Command) (infoService, error) { return &fakeInfoShortcutService{}, nil }

	var out, errOut bytes.Buffer
	getPrinter = func(cmd *cobra.Command) *output.Printer {
		return output.NewWithStderr(&out, &errOut, output.FormatJSON)
	}

	cmd := newInfoCoinOverviewCmd()
	err := cmd.RunE(cmd, nil)
	require.Error(t, err)
	assert.Empty(t, out.String())
	assert.Contains(t, errOut.String(), `"error"`)
	assert.Contains(t, errOut.String(), `"label":"INVALID_ARGUMENTS"`)
	assert.Contains(t, errOut.String(), "missing required flag: symbol")
}

func TestInfoShortcutCoinOverviewOmitsScopeOnGetCoinInfo(t *testing.T) {
	svc := &coinOverviewRecordingService{}
	oldFactory, oldPrinter := newInfoService, getPrinter
	t.Cleanup(func() { newInfoService = oldFactory; getPrinter = oldPrinter })
	newInfoService = func(cmd *cobra.Command) (infoService, error) { return svc, nil }

	var out, errOut bytes.Buffer
	getPrinter = func(cmd *cobra.Command) *output.Printer {
		return output.NewWithStderr(&out, &errOut, output.FormatJSON)
	}

	cmd := newInfoCoinOverviewCmd()
	require.NoError(t, cmd.Flags().Set("symbol", "BTC"))
	require.NoError(t, cmd.RunE(cmd, nil))
	require.NotEmpty(t, svc.coinArgs)
	assert.Equal(t, "BTC", svc.coinArgs["query"])
	assert.Equal(t, "symbol", svc.coinArgs["query_type"])
	if _, ok := svc.coinArgs["scope"]; ok {
		t.Fatalf("expected get-coin-info without scope, got %#v", svc.coinArgs)
	}
	if _, ok := svc.coinArgs["symbol"]; ok {
		t.Fatalf("expected query not symbol, got %#v", svc.coinArgs)
	}
}

type coinOverviewRecordingService struct {
	coinArgs map[string]interface{}
}

func (s *coinOverviewRecordingService) ListTools(ctx context.Context) ([]intelfacade.ToolSummary, *http.Response, error) {
	return nil, nil, nil
}
func (s *coinOverviewRecordingService) DescribeTool(ctx context.Context, name string) (*intelfacade.ToolSummary, *http.Response, error) {
	return &intelfacade.ToolSummary{Name: name}, nil, nil
}
func (s *coinOverviewRecordingService) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcpclient.CallResult, *http.Response, error) {
	if name == "info_coin_get_coin_info" {
		s.coinArgs = arguments
	}
	return &mcpclient.CallResult{StructuredContent: map[string]interface{}{"tool": name}}, nil, nil
}

func TestInfoShortcutCoinCompareUsesQueryForCoinInfo(t *testing.T) {
	svc := &coinCompareRecordingService{}
	oldFactory, oldPrinter := newInfoService, getPrinter
	t.Cleanup(func() { newInfoService = oldFactory; getPrinter = oldPrinter })
	newInfoService = func(cmd *cobra.Command) (infoService, error) { return svc, nil }

	var out, errOut bytes.Buffer
	getPrinter = func(cmd *cobra.Command) *output.Printer {
		return output.NewWithStderr(&out, &errOut, output.FormatJSON)
	}

	cmd := newInfoCoinCompareCmd()
	require.NoError(t, cmd.Flags().Set("symbols", "BTC,ETH"))
	require.NoError(t, cmd.RunE(cmd, nil))
	require.NotEmpty(t, svc.coinArgs)
	assert.Equal(t, "BTC", svc.coinArgs["query"])
	assert.Equal(t, "symbol", svc.coinArgs["query_type"])
}

type coinCompareRecordingService struct {
	coinArgs map[string]interface{}
}

func (s *coinCompareRecordingService) ListTools(ctx context.Context) ([]intelfacade.ToolSummary, *http.Response, error) {
	return nil, nil, nil
}
func (s *coinCompareRecordingService) DescribeTool(ctx context.Context, name string) (*intelfacade.ToolSummary, *http.Response, error) {
	return &intelfacade.ToolSummary{Name: name}, nil, nil
}
func (s *coinCompareRecordingService) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcpclient.CallResult, *http.Response, error) {
	if name == "info_coin_get_coin_info" && s.coinArgs == nil {
		s.coinArgs = arguments
	}
	return &mcpclient.CallResult{StructuredContent: map[string]interface{}{"tool": name}}, nil, nil
}

func TestInfoShortcutCoinOverview(t *testing.T) {
	oldFactory, oldPrinter := newInfoService, getPrinter
	t.Cleanup(func() { newInfoService = oldFactory; getPrinter = oldPrinter })
	newInfoService = func(cmd *cobra.Command) (infoService, error) { return &fakeInfoShortcutService{}, nil }

	var out, errOut bytes.Buffer
	getPrinter = func(cmd *cobra.Command) *output.Printer {
		return output.NewWithStderr(&out, &errOut, output.FormatJSON)
	}

	cmd := newInfoCoinOverviewCmd()
	require.NoError(t, cmd.Flags().Set("symbol", "BTC"))
	require.NoError(t, cmd.RunE(cmd, nil))
	assert.Contains(t, out.String(), `"summary"`)
	assert.Contains(t, out.String(), `"basic_info"`)
	assert.Empty(t, errOut.String())
}

func TestInfoShortcutTokenRiskNeedsResolvedCanonical(t *testing.T) {
	stdout, stderr, err := runTokenRiskShortcut(t, &fakeInfoShortcutService{}, func(cmd *cobra.Command) {
		require.NoError(t, cmd.Flags().Set("symbol", "BTC"))
	})
	require.Error(t, err)
	assert.Empty(t, stdout)
	assert.Contains(t, stderr, `"error"`)
	assert.Contains(t, stderr, "failed to resolve canonical contract or chain from symbol")
}

func TestInfoShortcutTokenRiskNativeCoinMessage(t *testing.T) {
	svc := &tokenRiskRecordingService{nativeCoin: true}
	stdout, stderr, err := runTokenRiskShortcut(t, svc, func(cmd *cobra.Command) {
		require.NoError(t, cmd.Flags().Set("symbol", "BTC"))
	})
	require.Error(t, err)
	assert.Empty(t, stdout)
	assert.Contains(t, stderr, "native coin has no contract address for token security")
}

func TestInfoShortcutTokenRiskAddressBranchCallOrder(t *testing.T) {
	svc := &tokenRiskRecordingService{}
	stdout, stderr, err := runTokenRiskShortcut(t, svc, func(cmd *cobra.Command) {
		require.NoError(t, cmd.Flags().Set("address", "0xabc"))
		require.NoError(t, cmd.Flags().Set("chain", "eth"))
	})
	require.NoError(t, err)
	assert.Empty(t, stderr)
	require.Equal(t, []string{
		"info_compliance_check_token_security",
		"info_coin_get_coin_info",
	}, svc.calls)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &payload))
	assert.Equal(t, "low", payload["risk_level"].(map[string]interface{})["risk_level"])
	assert.NotNil(t, payload["token_identity"])
	assert.NotContains(t, payload, "partial")
}

func TestInfoShortcutTokenRiskAddressBranchSecurityFailsBeforeCoinLookup(t *testing.T) {
	svc := &tokenRiskRecordingService{failSecurity: true}
	stdout, stderr, err := runTokenRiskShortcut(t, svc, func(cmd *cobra.Command) {
		require.NoError(t, cmd.Flags().Set("address", "0xabc"))
		require.NoError(t, cmd.Flags().Set("chain", "eth"))
	})
	require.Error(t, err)
	assert.Empty(t, stdout)
	assert.Contains(t, stderr, `"error"`)
	require.Equal(t, []string{"info_compliance_check_token_security"}, svc.calls)
}

func TestInfoShortcutTokenRiskAddressBranchPartialWhenCoinMissing(t *testing.T) {
	svc := &tokenRiskRecordingService{failCoin: true}
	stdout, stderr, err := runTokenRiskShortcut(t, svc, func(cmd *cobra.Command) {
		require.NoError(t, cmd.Flags().Set("address", "0xabc"))
		require.NoError(t, cmd.Flags().Set("chain", "eth"))
	})
	require.NoError(t, err)
	assert.Empty(t, stderr)
	require.Equal(t, []string{
		"info_compliance_check_token_security",
		"info_coin_get_coin_info",
	}, svc.calls)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &payload))
	assert.Equal(t, true, payload["partial"])
	missing, ok := payload["missing_sections"].([]interface{})
	require.True(t, ok)
	require.Len(t, missing, 1)
	assert.Equal(t, "token_identity", missing[0])
}

func TestInfoShortcutTokenRiskSymbolAndAddressMutualExclusion(t *testing.T) {
	stdout, stderr, err := runTokenRiskShortcut(t, &fakeInfoShortcutService{}, func(cmd *cobra.Command) {
		require.NoError(t, cmd.Flags().Set("symbol", "USDT"))
		require.NoError(t, cmd.Flags().Set("address", "0xabc"))
		require.NoError(t, cmd.Flags().Set("chain", "eth"))
	})
	require.Error(t, err)
	assert.Empty(t, stdout)
	assert.Contains(t, stderr, `"error"`)
	assert.Contains(t, stderr, "provide either symbol or address, not both")
}

func TestInfoShortcutTokenOnchainSymbolUsesQueryNotScopeFull(t *testing.T) {
	svc := &tokenOnchainRecordingService{}
	oldFactory, oldPrinter := newInfoService, getPrinter
	t.Cleanup(func() { newInfoService = oldFactory; getPrinter = oldPrinter })
	newInfoService = func(cmd *cobra.Command) (infoService, error) { return svc, nil }

	var out, errOut bytes.Buffer
	getPrinter = func(cmd *cobra.Command) *output.Printer {
		return output.NewWithStderr(&out, &errOut, output.FormatJSON)
	}

	cmd := newInfoTokenOnchainCmd()
	require.NoError(t, cmd.Flags().Set("symbol", "USDT"))
	require.NoError(t, cmd.RunE(cmd, nil))
	require.NotEmpty(t, svc.coinArgs)
	assert.Equal(t, "USDT", svc.coinArgs["query"])
	assert.Equal(t, "symbol", svc.coinArgs["query_type"])
	if _, ok := svc.coinArgs["scope"]; ok {
		t.Fatalf("expected get-coin-info without scope=full, got %#v", svc.coinArgs)
	}
	if _, ok := svc.coinArgs["symbol"]; ok {
		t.Fatalf("expected query not symbol, got %#v", svc.coinArgs)
	}
}

type tokenOnchainRecordingService struct {
	coinArgs map[string]interface{}
}

func (s *tokenOnchainRecordingService) ListTools(ctx context.Context) ([]intelfacade.ToolSummary, *http.Response, error) {
	return nil, nil, nil
}
func (s *tokenOnchainRecordingService) DescribeTool(ctx context.Context, name string) (*intelfacade.ToolSummary, *http.Response, error) {
	return &intelfacade.ToolSummary{Name: name}, nil, nil
}
func (s *tokenOnchainRecordingService) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcpclient.CallResult, *http.Response, error) {
	if name == "info_coin_get_coin_info" {
		s.coinArgs = arguments
		return &mcpclient.CallResult{StructuredContent: map[string]interface{}{
			"canonical_contract": "0xusdt",
			"canonical_chain":    "eth",
		}}, nil, nil
	}
	return &mcpclient.CallResult{StructuredContent: map[string]interface{}{"tool": name}}, nil, nil
}

func TestInfoShortcutTokenRiskSymbolBranchCallOrder(t *testing.T) {
	svc := &tokenRiskRecordingService{}
	stdout, stderr, err := runTokenRiskShortcut(t, svc, func(cmd *cobra.Command) {
		require.NoError(t, cmd.Flags().Set("symbol", "BTC"))
	})
	require.NoError(t, err)
	assert.Empty(t, stderr)
	require.Equal(t, []string{
		"info_coin_get_coin_info",
		"info_compliance_check_token_security",
	}, svc.calls)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(stdout), &payload))
	identity, ok := payload["token_identity"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "0xresolved", identity["canonical_contract"])
	assert.NotContains(t, payload, "partial")
}
