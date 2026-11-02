package futures

import (
	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/stretchr/testify/require"
	"testing"
)

// Exercise execution-affecting combinations independently of network/auth state.
func TestValidateChaseCreate(t *testing.T) {
	base := gateapi.CreateChaseOrderReq{Contract: "BTC_USDT", Amount: "-0.000000000000000001", PriceLimit: "0"}
	require.NoError(t, validateChaseCreate("usdt", base))
	for _, change := range []func(*gateapi.CreateChaseOrderReq){
		func(v *gateapi.CreateChaseOrderReq) { v.Amount = "0" }, func(v *gateapi.CreateChaseOrderReq) { v.Amount = "1e5" },
		func(v *gateapi.CreateChaseOrderReq) { v.PriceLimit = "-1" }, func(v *gateapi.CreateChaseOrderReq) { v.PriceLimit = "" },
		func(v *gateapi.CreateChaseOrderReq) { v.OffsetLimit = "0" }, func(v *gateapi.CreateChaseOrderReq) { v.PriceLimit = "10"; v.OffsetLimit = "1" },
		func(v *gateapi.CreateChaseOrderReq) { v.Settle = "btc" }, func(v *gateapi.CreateChaseOrderReq) { v.Contract = "../BTC" },
		func(v *gateapi.CreateChaseOrderReq) { v.PriceType = 3 }, func(v *gateapi.CreateChaseOrderReq) { v.PriceType = 2 },
		func(v *gateapi.CreateChaseOrderReq) { v.PriceType = 2; v.PriceGapType = 1; v.PriceGapValue = "-1" },
		func(v *gateapi.CreateChaseOrderReq) { v.PriceGapValue = "1" }, func(v *gateapi.CreateChaseOrderReq) { v.PosMarginMode = "bad" },
		func(v *gateapi.CreateChaseOrderReq) { v.PositionMode = "bad" },
	} {
		v := base
		change(&v)
		require.Error(t, validateChaseCreate("usdt", v))
	}
	base.OffsetLimit = "1"
	base.PriceType = 2
	base.PriceGapType = 2
	base.PriceGapValue = "0.01"
	require.NoError(t, validateChaseCreate("usdt", base))
}
