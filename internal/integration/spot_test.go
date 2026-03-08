//go:build integration

package integration_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gate/gate-cli/internal/integration"
)

func TestSpotAccountList(t *testing.T) {
	c := integration.LoadClient(t)

	accounts, httpResp, err := c.SpotAPI.ListSpotAccounts(c.Context(), nil)
	require.NoError(t, err)
	assert.Equal(t, 200, httpResp.StatusCode)
	assert.NotEmpty(t, accounts, "expected at least one currency in spot account")

	t.Logf("spot accounts returned: %d currencies", len(accounts))
}

func TestSpotOrderList(t *testing.T) {
	c := integration.LoadClient(t)

	orders, httpResp, err := c.SpotAPI.ListOrders(c.Context(), "BTC_USDT", "open", nil)
	require.NoError(t, err)
	assert.Equal(t, 200, httpResp.StatusCode)

	t.Logf("open spot orders for BTC_USDT: %d", len(orders))
}

func TestSpotMarketTicker(t *testing.T) {
	c := integration.LoadClient(t)

	tickers, httpResp, err := c.SpotAPI.ListTickers(c.Context(), nil)
	require.NoError(t, err)
	assert.Equal(t, 200, httpResp.StatusCode)
	assert.NotEmpty(t, tickers)

	t.Logf("spot tickers returned: %d pairs", len(tickers))
}
