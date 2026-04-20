package spot

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- buildSpotOrder (P1-3) ---

func TestBuildSpotOrder_MarketBuyWithQuote(t *testing.T) {
	order, err := buildSpotOrder("buy", "BTC_USDT", "", "", "10")
	require.NoError(t, err)
	assert.Equal(t, "market", order.Type)
	assert.Equal(t, "10", order.Amount) // quote amount passed through
	assert.Equal(t, "buy", order.Side)
}

func TestBuildSpotOrder_MarketBuyAmountRejected(t *testing.T) {
	// --amount must not be used for market buy (it would be misinterpreted as base currency).
	_, err := buildSpotOrder("buy", "BTC_USDT", "0.001", "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--quote")
}

func TestBuildSpotOrder_MarketBuyMissingQuote(t *testing.T) {
	// Market buy without --quote should error.
	_, err := buildSpotOrder("buy", "BTC_USDT", "", "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--quote")
}

func TestBuildSpotOrder_MarketBuyAmountAndQuoteBothSet(t *testing.T) {
	// If --amount is set alongside --quote, --amount wins the rejection check.
	_, err := buildSpotOrder("buy", "BTC_USDT", "0.001", "", "10")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--quote")
}

func TestBuildSpotOrder_LimitBuy(t *testing.T) {
	order, err := buildSpotOrder("buy", "BTC_USDT", "0.001", "50000", "")
	require.NoError(t, err)
	assert.Equal(t, "limit", order.Type)
	assert.Equal(t, "0.001", order.Amount)
	assert.Equal(t, "50000", order.Price)
}

func TestBuildSpotOrder_LimitBuyMissingAmount(t *testing.T) {
	_, err := buildSpotOrder("buy", "BTC_USDT", "", "50000", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--amount")
}

func TestBuildSpotOrder_MarketSell(t *testing.T) {
	order, err := buildSpotOrder("sell", "BTC_USDT", "0.001", "", "")
	require.NoError(t, err)
	assert.Equal(t, "market", order.Type)
	assert.Equal(t, "0.001", order.Amount)
}

func TestBuildSpotOrder_MarketSellMissingAmount(t *testing.T) {
	_, err := buildSpotOrder("sell", "BTC_USDT", "", "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--amount")
}

func TestBuildSpotOrder_LimitSell(t *testing.T) {
	order, err := buildSpotOrder("sell", "BTC_USDT", "0.001", "999000", "")
	require.NoError(t, err)
	assert.Equal(t, "limit", order.Type)
	assert.Equal(t, "0.001", order.Amount)
	assert.Equal(t, "999000", order.Price)
}

func TestBuildSpotOrder_MarketBuyTifIsIoc(t *testing.T) {
	order, err := buildSpotOrder("buy", "BTC_USDT", "", "", "10")
	require.NoError(t, err)
	assert.Equal(t, "ioc", order.TimeInForce)
}

func TestBuildSpotOrder_MarketSellTifIsIoc(t *testing.T) {
	order, err := buildSpotOrder("sell", "BTC_USDT", "0.001", "", "")
	require.NoError(t, err)
	assert.Equal(t, "ioc", order.TimeInForce)
}

func TestBuildSpotOrder_LimitOrderNoTif(t *testing.T) {
	// Limit orders do not set time_in_force; the API defaults to gtc.
	order, err := buildSpotOrder("buy", "BTC_USDT", "0.001", "80000", "")
	require.NoError(t, err)
	assert.Equal(t, "", order.TimeInForce)
}
