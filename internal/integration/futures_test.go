//go:build integration

package integration_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/gate/gate-cli/internal/integration"
)

func TestFuturesAccountGet(t *testing.T) {
	c := integration.LoadClient(t)

	account, httpResp, err := c.FuturesAPI.ListFuturesAccounts(c.Context(), "usdt")
	require.NoError(t, err)
	assert.Equal(t, 200, httpResp.StatusCode)
	assert.Equal(t, "usdt", account.Currency)

	t.Logf("futures account total: %s, available: %s", account.Total, account.Available)
}

func TestFuturesPositionList(t *testing.T) {
	c := integration.LoadClient(t)

	positions, httpResp, err := c.FuturesAPI.ListPositions(c.Context(), "usdt", nil)
	require.NoError(t, err)
	assert.Equal(t, 200, httpResp.StatusCode)

	t.Logf("open futures positions: %d", len(positions))
}

func TestFuturesOrderList(t *testing.T) {
	c := integration.LoadClient(t)

	opts := gateapi.ListFuturesOrdersOpts{}
	orders, httpResp, err := c.FuturesAPI.ListFuturesOrders(c.Context(), "usdt", "open", &opts)
	require.NoError(t, err)
	assert.Equal(t, 200, httpResp.StatusCode)

	t.Logf("open futures orders: %d", len(orders))
}
