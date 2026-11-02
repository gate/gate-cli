package spot

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/antihax/optional"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/spf13/cobra"
)

// Empty account preserves existing server-side selection, including unified accounts.
func addSpotAccountFlag(cmd *cobra.Command) {
	cmd.Flags().String("account", "", "Order account: spot, margin, unified or cross_margin; omitted uses server defaults")
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error { return validateSpotAccount(spotAccountValue(cmd)) }
}
func spotAccountValue(cmd *cobra.Command) string { v, _ := cmd.Flags().GetString("account"); return v }
func spotAccountOption(cmd *cobra.Command) optional.String {
	v := spotAccountValue(cmd)
	if v == "" {
		return optional.String{}
	}
	return optional.NewString(v)
}
func validateSpotAccount(v string) error {
	switch v {
	case "", "spot", "margin", "unified", "cross_margin":
		return nil
	default:
		return fmt.Errorf("--account must be spot, margin, unified or cross_margin")
	}
}
func addSpotProtectionFlags(cmd *cobra.Command) {
	cmd.Flags().String("stop-profit-json", "", "Take profit: {trigger_price,order_price}; on amend {} cancels, null/omitted keeps unchanged")
	cmd.Flags().String("stop-loss-json", "", "Stop loss: {trigger_price,order_price}; on amend {} cancels, null/omitted keeps unchanged")
}

// The SDK distinguishes null (unchanged) from {} (cancel) on amendments.
func readSpotProtection(cmd *cobra.Command, name string, amend bool) (*gateapi.SpotOrderStopProfit, error) {
	if !cmd.Flags().Changed(name) {
		return nil, nil
	}
	raw, _ := cmd.Flags().GetString(name)
	if strings.TrimSpace(raw) == "null" {
		return nil, nil
	}
	var v gateapi.SpotOrderStopProfit
	if err := cmdutil.DecodeTradingObject("--"+name, raw, &v); err != nil {
		return nil, err
	}
	// Only a literal empty object requests cancellation; null/empty members must
	// never collapse into {} and silently remove existing protection.
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		return nil, fmt.Errorf("invalid --%s: %w", name, err)
	}
	if len(fields) == 0 {
		if amend {
			return &v, nil
		}
		return nil, fmt.Errorf("--%s cannot be empty on create", name)
	}
	for field, rawValue := range fields {
		var value string
		if string(rawValue) == "null" {
			return nil, fmt.Errorf("--%s %s cannot be null", name, field)
		}
		if err := json.Unmarshal(rawValue, &value); err != nil || value == "" {
			return nil, fmt.Errorf("--%s %s must be a non-empty decimal string", name, field)
		}
	}
	if _, err := cmdutil.TradingDecimal("--"+name+" trigger_price", v.TriggerPrice, true); err != nil {
		return nil, err
	}
	// Preserve the decimal string; the server applies product-specific execution rules.
	if v.OrderPrice != "" {
		price, err := cmdutil.TradingDecimal("--"+name+" order_price", v.OrderPrice, false)
		if err != nil {
			return nil, err
		}
		if price.Sign() < 0 {
			return nil, fmt.Errorf("--%s order_price must be non-negative", name)
		}
	}
	return &v, nil
}
func applySpotCreateFlags(cmd *cobra.Command, order *gateapi.Order) error {
	account := spotAccountValue(cmd)
	if err := validateSpotAccount(account); err != nil {
		return err
	}
	borrow, _ := cmd.Flags().GetBool("auto-borrow")
	repay, _ := cmd.Flags().GetBool("auto-repay")
	if borrow && account != "margin" && account != "unified" && account != "cross_margin" {
		return fmt.Errorf("--auto-borrow requires an explicit margin, unified or cross_margin account")
	}
	if repay && account != "unified" && account != "cross_margin" {
		return fmt.Errorf("--auto-repay requires an explicit unified or cross_margin account; isolated margin uses its account-level repayment setting")
	}
	tp, err := readSpotProtection(cmd, "stop-profit-json", false)
	if err != nil {
		return err
	}
	sl, err := readSpotProtection(cmd, "stop-loss-json", false)
	if err != nil {
		return err
	}
	if tp != nil || sl != nil {
		if order.Type != "limit" {
			return fmt.Errorf("attached take profit and stop loss require a limit order")
		}
		price, err := cmdutil.TradingDecimal("--price", order.Price, true)
		if err != nil {
			return err
		}
		for _, v := range []struct {
			protection *gateapi.SpotOrderStopProfit
			profit     bool
		}{{tp, true}, {sl, false}} {
			if v.protection == nil {
				continue
			}
			trigger, err := cmdutil.TradingDecimal("trigger_price", v.protection.TriggerPrice, true)
			if err != nil {
				return err
			}
			above := (order.Side == "buy") == v.profit
			if (above && trigger.Cmp(price) <= 0) || (!above && trigger.Cmp(price) >= 0) {
				return fmt.Errorf("protection trigger_price is on the wrong side of the limit price")
			}
		}
	}
	order.Account = account
	order.AutoBorrow = borrow
	order.AutoRepay = repay
	order.StopProfit = tp
	if sl != nil {
		order.StopLoss = &gateapi.SpotOrderStopLoss{TriggerPrice: sl.TriggerPrice, OrderPrice: sl.OrderPrice}
	}
	return nil
}
func applySpotPatchProtection(cmd *cobra.Command, patch *gateapi.OrderPatch) error {
	tp, err := readSpotProtection(cmd, "stop-profit-json", true)
	if err != nil {
		return err
	}
	sl, err := readSpotProtection(cmd, "stop-loss-json", true)
	if err != nil {
		return err
	}
	patch.StopProfit = tp
	if sl != nil {
		patch.StopLoss = &gateapi.SpotOrderStopLoss{TriggerPrice: sl.TriggerPrice, OrderPrice: sl.OrderPrice}
	}
	// Side and existing price are authoritative at the server; do not validate from stale reads.
	return nil
}
