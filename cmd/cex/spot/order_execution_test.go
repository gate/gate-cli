package spot

import (
	"encoding/json"
	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"testing"
)

func executionFlags(t *testing.T, flags ...string) *cobra.Command {
	t.Helper()
	c := &cobra.Command{Use: "test"}
	addSpotAccountFlag(c)
	addSpotProtectionFlags(c)
	c.Flags().Bool("auto-borrow", false, "")
	c.Flags().Bool("auto-repay", false, "")
	require.NoError(t, c.ParseFlags(flags))
	return c
}
func TestSpotAccountBorrowRepay(t *testing.T) {
	for _, account := range []string{"", "spot", "margin", "unified", "cross_margin"} {
		require.NoError(t, validateSpotAccount(account))
	}
	require.Error(t, validateSpotAccount("other"))
	for _, tc := range []struct {
		flags []string
		ok    bool
	}{
		{nil, true}, {[]string{"--account", "margin", "--auto-borrow"}, true},
		{[]string{"--account", "unified", "--auto-borrow", "--auto-repay"}, true},
		{[]string{"--account", "cross_margin", "--auto-borrow", "--auto-repay"}, true},
		{[]string{"--auto-borrow"}, false}, {[]string{"--auto-repay"}, false},
		{[]string{"--account", "spot", "--auto-borrow"}, false}, {[]string{"--account", "margin", "--auto-repay"}, false},
	} {
		order := gateapi.Order{}
		c := executionFlags(t, tc.flags...)
		require.Equal(t, tc.ok, applySpotCreateFlags(c, &order) == nil)
		if tc.ok {
			require.Equal(t, spotAccountValue(c), order.Account)
		}
	}
	c := executionFlags(t)
	require.False(t, spotAccountOption(c).IsSet())
	c = executionFlags(t, "--account", "margin")
	require.Equal(t, "margin", spotAccountOption(c).Value())
}
func TestSpotProtectionCreate(t *testing.T) {
	for _, tc := range []struct {
		side, kind, price, raw string
		ok                     bool
	}{
		{"buy", "stop-profit-json", "100", `{"trigger_price":"100.000000000000000001","order_price":"101"}`, true},
		{"buy", "stop-loss-json", "100", `{"trigger_price":"99"}`, true},
		{"sell", "stop-profit-json", "100", `{"trigger_price":"99"}`, true},
		{"sell", "stop-loss-json", "100", `{"trigger_price":"101"}`, true},
		{"buy", "stop-profit-json", "100", `{"trigger_price":"100"}`, false},
		{"buy", "stop-loss-json", "100", `{"trigger_price":"101"}`, false},
		{"sell", "stop-profit-json", "100", `{"trigger_price":"101"}`, false},
		{"sell", "stop-loss-json", "100", `{"trigger_price":"99"}`, false},
		{"buy", "stop-profit-json", "", `{"trigger_price":"101"}`, false},
		{"buy", "stop-profit-json", "100", `{}`, false}, {"buy", "stop-profit-json", "100", `{"trigger_price":"NaN"}`, false},
		{"buy", "stop-profit-json", "100", `{"order_price":"100"}`, false},
		{"buy", "stop-profit-json", "100", `{"trigger_price":"101","order_price":"-1"}`, false},
		{"buy", "stop-profit-json", "100", `{"triger_price":"101"}`, false},
	} {
		order := gateapi.Order{Side: tc.side, Type: "limit", Price: tc.price}
		if tc.price == "" {
			order.Type = "market"
		}
		require.Equal(t, tc.ok, applySpotCreateFlags(executionFlags(t, "--"+tc.kind, tc.raw), &order) == nil, tc)
	}
}
func TestSpotProtectionAmendTriState(t *testing.T) {
	for _, tc := range []struct {
		flags        []string
		profit, loss string
	}{
		{nil, "null", "null"}, {[]string{"--stop-profit-json", "null"}, "null", "null"},
		{[]string{"--stop-profit-json", "{}"}, "{}", "null"},
		{[]string{"--stop-loss-json", "{}"}, "null", "{}"},
		{[]string{"--stop-profit-json", `{"trigger_price":"101"}`, "--stop-loss-json", `{"trigger_price":"99"}`}, `{"trigger_price":"101"}`, `{"trigger_price":"99"}`},
	} {
		var patch gateapi.OrderPatch
		require.NoError(t, applySpotPatchProtection(executionFlags(t, tc.flags...), &patch))
		raw, err := json.Marshal(patch)
		require.NoError(t, err)
		var wire map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(raw, &wire))
		require.JSONEq(t, tc.profit, string(wire["stop_profit"]))
		require.JSONEq(t, tc.loss, string(wire["stop_loss"]))
	}
	for _, raw := range []string{`{"trigger_price":null}`, `{"trigger_price":""}`, `{"order_price":""}`, `{"order_price":null}`, `{"trigger_price":"99","order_price":null}`} {
		var patch gateapi.OrderPatch
		require.Error(t, applySpotPatchProtection(executionFlags(t, "--stop-loss-json", raw), &patch), raw)
	}
	var patch gateapi.OrderPatch
	require.Error(t, applySpotPatchProtection(executionFlags(t, "--stop-loss-json", `{"trigger_price":"-1"}`), &patch))
}
