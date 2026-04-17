//go:build integration

package integration_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gate/gate-cli/internal/integration"
	gateapi "github.com/gate/gateapi-go/v7"
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

// TestSpotOrderCreateAndCancel places a far-OTM limit order and cancels it.
//
// The test first inspects spot account balances to pick a usable currency and side:
//   - USDT balance → limit buy BTC at $5,000 (far below market, ≈5 USDT notional)
//   - BTC balance  → limit sell BTC at $999,000 (far above market, ≈0.001 BTC needed)
//
// If no usable balance is found the test is skipped.
func TestSpotOrderCreateAndCancel(t *testing.T) {
	c := integration.LoadClient(t)

	// Discover which currencies have a non-zero available balance.
	accounts, httpResp, err := c.SpotAPI.ListSpotAccounts(c.Context(), nil)
	require.NoError(t, err)
	assert.Equal(t, 200, httpResp.StatusCode)

	type orderSpec struct {
		pair, side, amount, price string
	}
	var spec *orderSpec
	for _, acc := range accounts {
		avail, _ := strconv.ParseFloat(acc.Available, 64)
		switch acc.Currency {
		case "USDT":
			// Need at least 5 USDT: 0.001 BTC × $5,000 = $5 (above 1 USDT minimum).
			if avail >= 5 {
				spec = &orderSpec{"BTC_USDT", "buy", "0.001", "5000"}
			}
		case "BTC":
			// Need at least 0.001 BTC. Sell at $999,000 — far above market, never fills.
			if avail >= 0.001 && spec == nil {
				spec = &orderSpec{"BTC_USDT", "sell", "0.001", "999000"}
			}
		}
	}
	if spec == nil {
		t.Skip("no suitable spot balance (≥5 USDT or ≥0.001 BTC) found — skipping order test")
	}
	t.Logf("using pair=%s side=%s amount=%s price=%s", spec.pair, spec.side, spec.amount, spec.price)

	order := gateapi.Order{
		CurrencyPair: spec.pair,
		Side:         spec.side,
		Type:         "limit",
		Amount:       spec.amount,
		Price:        spec.price,
		TimeInForce:  "gtc",
	}
	created, httpResp, err := c.SpotAPI.CreateOrder(c.Context(), order, nil)
	require.NoError(t, err)
	assert.Equal(t, 201, httpResp.StatusCode)
	assert.Equal(t, "open", created.Status)
	t.Logf("created spot order: id=%s side=%s amount=%s price=%s status=%s",
		created.Id, created.Side, created.Amount, created.Price, created.Status)

	// Safety net: cancel if the test fails before reaching the cancel step.
	t.Cleanup(func() {
		if created.Status == "open" {
			c.SpotAPI.CancelOrder(c.Context(), created.Id, spec.pair, nil) //nolint:errcheck
		}
	})

	// Cancel the order.
	cancelled, httpResp, err := c.SpotAPI.CancelOrder(c.Context(), created.Id, spec.pair, nil)
	require.NoError(t, err)
	assert.Equal(t, 200, httpResp.StatusCode)
	assert.Equal(t, "cancelled", cancelled.Status)
	t.Logf("cancelled spot order: id=%s finish_as=%s", cancelled.Id, cancelled.FinishAs)
}
