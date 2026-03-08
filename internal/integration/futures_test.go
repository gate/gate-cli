//go:build integration

package integration_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/antihax/optional"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/gate/gate-cli/internal/integration"
)

const (
	settle   = "usdt"
	contract = "BTC_USDT"
)

func TestFuturesAccountGet(t *testing.T) {
	c := integration.LoadClient(t)

	account, httpResp, err := c.FuturesAPI.ListFuturesAccounts(c.Context(), settle)
	require.NoError(t, err)
	assert.Equal(t, 200, httpResp.StatusCode)
	assert.True(t, strings.EqualFold("usdt", account.Currency), "expected currency usdt, got %s", account.Currency)

	t.Logf("futures account total: %s, available: %s, dual_mode: %v",
		account.Total, account.Available, account.InDualMode)
}

func TestFuturesPositionList(t *testing.T) {
	c := integration.LoadClient(t)

	positions, httpResp, err := c.FuturesAPI.ListPositions(c.Context(), settle, nil)
	require.NoError(t, err)
	assert.Equal(t, 200, httpResp.StatusCode)

	t.Logf("open futures positions: %d", len(positions))
}

func TestFuturesOrderList(t *testing.T) {
	c := integration.LoadClient(t)

	opts := gateapi.ListFuturesOrdersOpts{}
	orders, httpResp, err := c.FuturesAPI.ListFuturesOrders(c.Context(), settle, "open", &opts)
	require.NoError(t, err)
	assert.Equal(t, 200, httpResp.StatusCode)

	t.Logf("open futures orders: %d", len(orders))
}

// TestFuturesOrderCreateAndCancel places a far-OTM limit long at $1000 and cancels it.
// 5 contracts ≈ 5 USDT notional — safely within 10 U budget, above 1 U minimum.
func TestFuturesOrderCreateAndCancel(t *testing.T) {
	c := integration.LoadClient(t)

	// Limit long at $1000 — far below any realistic BTC price, will never fill.
	order := gateapi.FuturesOrder{
		Contract: contract,
		Size:     "5",
		Price:    "1000",
		Tif:      "gtc",
	}
	created, httpResp, err := c.FuturesAPI.CreateFuturesOrder(c.Context(), settle, order, nil)
	require.NoError(t, err)
	assert.Equal(t, 201, httpResp.StatusCode)
	assert.Equal(t, "open", created.Status)
	t.Logf("created futures order: id=%d size=%s price=%s status=%s",
		created.Id, created.Size, created.Price, created.Status)

	// Safety net: cancel if the test fails before reaching the cancel step.
	t.Cleanup(func() {
		if created.Status == "open" {
			c.FuturesAPI.CancelFuturesOrder(c.Context(), settle, fmt.Sprintf("%d", created.Id), nil) //nolint:errcheck
		}
	})

	// Cancel the order.
	orderId := fmt.Sprintf("%d", created.Id)
	cancelled, httpResp, err := c.FuturesAPI.CancelFuturesOrder(c.Context(), settle, orderId, nil)
	require.NoError(t, err)
	assert.Equal(t, 200, httpResp.StatusCode)
	assert.Equal(t, "finished", cancelled.Status)
	assert.Equal(t, "cancelled", cancelled.FinishAs)
	t.Logf("cancelled futures order: id=%d finish_as=%s", cancelled.Id, cancelled.FinishAs)
}

// TestFuturesLeverageUpdate changes leverage via the client helper that handles
// both single and dual mode transparently. Verifies the change then restores.
func TestFuturesLeverageUpdate(t *testing.T) {
	c := integration.LoadClient(t)

	// Find an open position to know which contract to use.
	// (UpdatePositionLeverage returns an empty array when no position exists.)
	positions, httpResp, err := c.FuturesAPI.ListPositions(c.Context(), settle, nil)
	require.NoError(t, err)
	assert.Equal(t, 200, httpResp.StatusCode)

	var target *gateapi.Position
	for i := range positions {
		if positions[i].Size != "0" && positions[i].Size != "" {
			target = &positions[i]
			break
		}
	}
	if target == nil {
		t.Skip("no open futures positions to test leverage update")
	}
	t.Logf("using contract=%s leverage=%s cross_leverage_limit=%s (dual_mode=%v)",
		target.Contract, target.Leverage, target.CrossLeverageLimit, c.IsDualMode(settle))

	// Pick a new cross leverage limit that differs from the current one.
	originalCrossLimit := target.CrossLeverageLimit
	if originalCrossLimit == "" {
		originalCrossLimit = "10"
	}
	newCrossLimit := "20"
	if originalCrossLimit == "20" {
		newCrossLimit = "10"
	}

	// Use the client wrapper — routes to dual or single mode endpoint automatically.
	opts := &gateapi.UpdatePositionLeverageOpts{
		CrossLeverageLimit: optional.NewString(newCrossLimit),
	}
	updated, httpResp, err := c.UpdateFuturesPositionLeverage(settle, target.Contract, "0", opts)
	require.NoError(t, err)
	assert.Equal(t, 200, httpResp.StatusCode)
	require.NotEmpty(t, updated)
	assert.Equal(t, newCrossLimit, updated[0].CrossLeverageLimit)
	t.Logf("cross leverage limit updated: %s → %s", originalCrossLimit, updated[0].CrossLeverageLimit)

	// Restore.
	restoreOpts := &gateapi.UpdatePositionLeverageOpts{
		CrossLeverageLimit: optional.NewString(originalCrossLimit),
	}
	restored, _, err := c.UpdateFuturesPositionLeverage(settle, target.Contract, "0", restoreOpts)
	require.NoError(t, err)
	require.NotEmpty(t, restored)
	assert.Equal(t, originalCrossLimit, restored[0].CrossLeverageLimit)
	t.Logf("cross leverage limit restored to %s", restored[0].CrossLeverageLimit)
}

// TestFuturesGetPosition verifies GetFuturesPosition works in both single and dual mode.
func TestFuturesGetPosition(t *testing.T) {
	c := integration.LoadClient(t)

	// Find a contract with a non-zero position.
	all, _, err := c.FuturesAPI.ListPositions(c.Context(), settle, nil)
	require.NoError(t, err)
	targetContract := ""
	for _, p := range all {
		if p.Size != "0" && p.Size != "" {
			targetContract = p.Contract
			break
		}
	}
	if targetContract == "" {
		t.Skip("no open futures positions to test GetFuturesPosition")
	}

	// Use the transparent client method.
	positions, httpResp, err := c.GetFuturesPosition(settle, targetContract)
	require.NoError(t, err)
	assert.Equal(t, 200, httpResp.StatusCode)
	require.NotEmpty(t, positions)

	for _, p := range positions {
		t.Logf("position: contract=%s mode=%s size=%s entry=%s leverage=%s",
			p.Contract, p.Mode, p.Size, p.EntryPrice, p.Leverage)
	}
}

// TestFuturesPositionLifecycle tests the full cycle:
//
//	open 5 contracts → add 5 more → reduce by 5 → close remaining 5.
//
// All orders use market execution (Price="0", Tif="ioc").
// Total exposure: 10 contracts × $1 each ≈ 10 USDT notional at any point.
func TestFuturesPositionLifecycle(t *testing.T) {
	c := integration.LoadClient(t)

	// Track how many long contracts we have opened so cleanup is precise.
	opened := int64(0)

	// Safety cleanup: sell whatever we opened (reduce-only, so it can never overshoot).
	t.Cleanup(func() {
		if opened <= 0 {
			return
		}
		t.Logf("cleanup: closing %d remaining contracts", opened)
		closeOrder := gateapi.FuturesOrder{
			Contract:   contract,
			Size:       fmt.Sprintf("-%d", opened),
			Price:      "0",
			Tif:        "ioc",
			ReduceOnly: true,
		}
		c.FuturesAPI.CreateFuturesOrder(c.Context(), settle, closeOrder, nil) //nolint:errcheck
	})

	buyOrder := gateapi.FuturesOrder{
		Contract: contract,
		Size:     "5",
		Price:    "0",
		Tif:      "ioc",
	}

	// ── 1. Open position: long 5 contracts ──────────────────────────────────
	buy1, httpResp, err := c.FuturesAPI.CreateFuturesOrder(c.Context(), settle, buyOrder, nil)
	require.NoError(t, err)
	assert.Equal(t, 201, httpResp.StatusCode)
	filled1, _ := strconv.ParseInt(buy1.Size, 10, 64)
	if filled1 < 0 {
		filled1 = -filled1
	}
	opened += filled1
	t.Logf("开仓: order id=%d size=%s fill_price=%s", buy1.Id, buy1.Size, buy1.FillPrice)

	// Verify position via the transparent client method.
	posList, _, err := c.GetFuturesPosition(settle, contract)
	require.NoError(t, err)
	var longPos *gateapi.Position
	for i := range posList {
		sz, _ := strconv.ParseInt(posList[i].Size, 10, 64)
		if sz > 0 {
			longPos = &posList[i]
			break
		}
	}
	require.NotNil(t, longPos, "expected long position for %s after opening", contract)
	t.Logf("position after 开仓: size=%s mode=%s leverage=%s entry=%s",
		longPos.Size, longPos.Mode, longPos.Leverage, longPos.EntryPrice)

	// ── 2. Add to position: long 5 more contracts (加仓) ────────────────────
	buy2, httpResp, err := c.FuturesAPI.CreateFuturesOrder(c.Context(), settle, buyOrder, nil)
	require.NoError(t, err)
	assert.Equal(t, 201, httpResp.StatusCode)
	filled2, _ := strconv.ParseInt(buy2.Size, 10, 64)
	if filled2 < 0 {
		filled2 = -filled2
	}
	opened += filled2
	t.Logf("加仓: order id=%d size=%s fill_price=%s (total opened=%d)", buy2.Id, buy2.Size, buy2.FillPrice, opened)

	// ── 3. Reduce position: sell 5 contracts (减仓, reduce-only) ────────────
	reduceOrder := gateapi.FuturesOrder{
		Contract:   contract,
		Size:       "-5",
		Price:      "0",
		Tif:        "ioc",
		ReduceOnly: true,
	}
	red, httpResp, err := c.FuturesAPI.CreateFuturesOrder(c.Context(), settle, reduceOrder, nil)
	require.NoError(t, err)
	assert.Equal(t, 201, httpResp.StatusCode)
	reducedAbs, _ := strconv.ParseInt(red.Size, 10, 64)
	if reducedAbs < 0 {
		reducedAbs = -reducedAbs
	}
	opened -= reducedAbs
	t.Logf("减仓: order id=%d size=%s fill_price=%s (remaining opened=%d)", red.Id, red.Size, red.FillPrice, opened)

	// ── 4. Close remaining position (平仓, reduce-only) ─────────────────────
	closeOrder := gateapi.FuturesOrder{
		Contract:   contract,
		Size:       fmt.Sprintf("-%d", opened),
		Price:      "0",
		Tif:        "ioc",
		ReduceOnly: true,
	}
	closed, httpResp, err := c.FuturesAPI.CreateFuturesOrder(c.Context(), settle, closeOrder, nil)
	require.NoError(t, err)
	assert.Equal(t, 201, httpResp.StatusCode)
	t.Logf("平仓: order id=%d size=%s fill_price=%s", closed.Id, closed.Size, closed.FillPrice)

	// Mark as fully closed so cleanup is a no-op.
	opened = 0

	// Final check via transparent position method.
	finalList, _, err := c.GetFuturesPosition(settle, contract)
	require.NoError(t, err)
	for _, p := range finalList {
		sz, _ := strconv.ParseInt(p.Size, 10, 64)
		if sz > 0 {
			t.Logf("final long position size=%s mode=%s", p.Size, p.Mode)
		}
	}
}
