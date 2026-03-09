package futures

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gateapi "github.com/gate/gateapi-go/v7"
)

// --- positionIsShort ---

func TestPositionIsShort_Long(t *testing.T) {
	positions := []gateapi.Position{{Contract: "BTC_USDT", Size: "10"}}
	isShort, err := positionIsShort(positions)
	require.NoError(t, err)
	assert.False(t, isShort)
}

func TestPositionIsShort_Short(t *testing.T) {
	positions := []gateapi.Position{{Contract: "BTC_USDT", Size: "-10"}}
	isShort, err := positionIsShort(positions)
	require.NoError(t, err)
	assert.True(t, isShort)
}

func TestPositionIsShort_ZeroSize(t *testing.T) {
	positions := []gateapi.Position{{Contract: "BTC_USDT", Size: "0"}}
	_, err := positionIsShort(positions)
	require.Error(t, err)
}

func TestPositionIsShort_Empty(t *testing.T) {
	_, err := positionIsShort(nil)
	require.Error(t, err)
}

func TestPositionIsShort_SkipsZeroPicksNonZero(t *testing.T) {
	// Dual-mode: first entry is zero (closed side), second is open.
	positions := []gateapi.Position{
		{Contract: "BTC_USDT", Mode: "dual_long", Size: "0"},
		{Contract: "BTC_USDT", Mode: "dual_short", Size: "-5"},
	}
	isShort, err := positionIsShort(positions)
	require.NoError(t, err)
	assert.True(t, isShort)
}

// --- applyDirectionSign (P1-1) ---

func TestApplyDirectionSign_AddLong(t *testing.T) {
	// Adding to a long position: keep positive size, no reduce-only.
	size, reduceOnly := applyDirectionSign("add", "5", false)
	assert.Equal(t, "5", size)
	assert.False(t, reduceOnly)
}

func TestApplyDirectionSign_AddShort(t *testing.T) {
	// Adding to a short position: negate the size.
	size, reduceOnly := applyDirectionSign("add", "5", true)
	assert.Equal(t, "-5", size)
	assert.False(t, reduceOnly)
}

func TestApplyDirectionSign_RemoveLong(t *testing.T) {
	// Reducing a long: sell (negative size) + reduce-only.
	size, reduceOnly := applyDirectionSign("remove", "5", false)
	assert.Equal(t, "-5", size)
	assert.True(t, reduceOnly)
}

func TestApplyDirectionSign_RemoveShort(t *testing.T) {
	// Reducing a short: buy (positive size) + reduce-only.
	size, reduceOnly := applyDirectionSign("remove", "5", true)
	assert.Equal(t, "5", size)
	assert.True(t, reduceOnly)
}

func TestApplyDirectionSign_RemoveShortAlreadyNegated(t *testing.T) {
	// If user mistakenly passes "-5" for remove, strip the minus for shorts.
	size, reduceOnly := applyDirectionSign("remove", "-5", true)
	assert.Equal(t, "5", size)
	assert.True(t, reduceOnly)
}

// --- applyFuturesTif ---

func TestApplyFuturesTif_MarketOrder(t *testing.T) {
	order := gateapi.FuturesOrder{}
	applyFuturesTif(&order, "")
	assert.Equal(t, "0", order.Price)
	assert.Equal(t, "ioc", order.Tif)
}

func TestApplyFuturesTif_LimitOrder(t *testing.T) {
	order := gateapi.FuturesOrder{}
	applyFuturesTif(&order, "80000")
	assert.Equal(t, "80000", order.Price)
	assert.Equal(t, "gtc", order.Tif)
}

// --- closePartialSingleSize (P1-2) ---

func TestClosePartialSingleSize_Long(t *testing.T) {
	// Closing part of a long: sell (negative size).
	size := closePartialSingleSize("5", false)
	assert.Equal(t, "-5", size)
}

func TestClosePartialSingleSize_Short(t *testing.T) {
	// Closing part of a short: buy (positive size).
	size := closePartialSingleSize("5", true)
	assert.Equal(t, "5", size)
}
