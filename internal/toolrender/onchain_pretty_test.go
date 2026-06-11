package toolrender

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gate/gate-cli/internal/mcpclient"
	"github.com/gate/gate-cli/internal/output"
)

func loadOnchainTestdata(t *testing.T, name string) map[string]interface{} {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	var data map[string]interface{}
	require.NoError(t, json.Unmarshal(b, &data))
	return data
}

func TestFormatAddressInfoPretty_AssetSummaryAndTokens(t *testing.T) {
	t.Parallel()
	data := map[string]interface{}{
		"address": "0xabc",
		"chain":   "ethereum",
		"asset_summary": map[string]interface{}{
			"total_usd_value":  100.5,
			"native_usd_value": 10.0,
			"token_usd_value":  90.5,
			"token_num":        2,
		},
		"token_balances": []interface{}{
			map[string]interface{}{"symbol": "LOW", "token_address": "0x1", "amount": "1", "value_usd": "1"},
			map[string]interface{}{"symbol": "HIGH", "token_address": "0x2", "amount": "2", "value_usd": "50"},
		},
		"multi_chain_token_balances": []interface{}{
			map[string]interface{}{"chain": "polygon", "token_symbol": "USDC", "amount": "3", "usd_value": "5", "percentage": "5"},
		},
	}
	out := formatAddressInfoPretty(data)
	assert.Contains(t, out, "Asset Summary")
	assert.Contains(t, out, "total_usd_value")
	assert.Contains(t, out, "Top Token Balances")
	assert.True(t, strings.Index(out, "HIGH") < strings.Index(out, "LOW"), "tokens should be sorted by value_usd desc")
	assert.Contains(t, out, "Multi-chain Assets")
	assert.Contains(t, out, "polygon")
}

func TestFormatAddressInfoPretty_SolanaTokenAccounts(t *testing.T) {
	t.Parallel()
	data := map[string]interface{}{
		"chain": "solana",
		"token_balances": []interface{}{
			map[string]interface{}{
				"token_account": "acct1",
				"mint_account":  "mint1",
				"token_address": "mint1",
			},
		},
	}
	out := formatAddressInfoPretty(data)
	assert.Contains(t, out, "Solana Token Accounts")
	assert.Contains(t, out, "acct1")
	assert.Contains(t, out, "mint1")
}

func TestFormatAddressTransactionsPretty_EmptyItemsNotMisleading(t *testing.T) {
	t.Parallel()
	out := formatAddressTransactionsPretty(map[string]interface{}{
		"address": "bc1q",
		"chain":   "bitcoin",
		"total":   5,
		"count":   0,
	})
	assert.Contains(t, out, "No transactions in this page")
	assert.NotContains(t, out, "暂无交易")
}

func TestFormatAddressInfoPretty_GoldenEVMFixture(t *testing.T) {
	t.Parallel()
	data := loadOnchainTestdata(t, "onchain_address_info_evm.json")
	out := formatAddressInfoPretty(data)
	assert.Contains(t, out, "Asset Summary")
	assert.Contains(t, out, "optimism")
	assert.Contains(t, out, "Top Token Balances")
	assert.Contains(t, out, "USDC")
	assert.Contains(t, out, "Multi-chain Assets")
	assert.True(t, strings.Index(out, "USDC") < strings.Index(out, "WETH"))
}

func TestFormatAddressInfoPretty_GoldenSolanaFixture(t *testing.T) {
	t.Parallel()
	data := loadOnchainTestdata(t, "onchain_address_info_solana.json")
	out := formatAddressInfoPretty(data)
	assert.Contains(t, out, "Solana Token Accounts")
	assert.Contains(t, out, "TokenAcct11111111111111111111111111111111")
}

func TestFormatAddressTransactionsPretty_GoldenBTCPartialPage(t *testing.T) {
	t.Parallel()
	out := formatAddressTransactionsPretty(map[string]interface{}{
		"address": "bc1qexample",
		"chain":   "bitcoin",
		"total":   5,
		"count":   0,
	})
	golden, err := os.ReadFile(filepath.Join("testdata", "onchain_address_transactions_btc.golden.txt"))
	require.NoError(t, err)
	assert.Equal(t, strings.TrimSpace(string(golden)), out)
}

func TestRenderCallResult_PrettyOnchainAddressInfo(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	p := output.NewWithStderr(&out, &errOut, output.FormatPretty)

	err := RenderCallResult(p, "info", toolInfoOnchainGetAddressInfo, &mcpclient.CallResult{
		StructuredContent: map[string]interface{}{
			"address": "0x1",
			"chain":   "optimism",
			"asset_summary": map[string]interface{}{
				"total_usd_value": 1,
			},
		},
	}, 0)
	require.NoError(t, err)
	body := out.String()
	assert.Contains(t, body, "Asset Summary")
	assert.Contains(t, body, "optimism")
	assert.NotContains(t, body, `"tool_name"`)
}
