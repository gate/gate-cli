package stock

import (
	"fmt"

	"github.com/antihax/optional"
	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/spf13/cobra"
)

func init() {
	group := &cobra.Command{Use: "order", Short: "Stock orders", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() }}
	list := &cobra.Command{Use: "list", Short: "List open orders", RunE: runOrders}
	list.Flags().String("symbol", "", "Filter by symbol")
	group.AddCommand(list)
	history := &cobra.Command{Use: "history", Short: "List historical orders", RunE: runOrderHistory}
	history.Flags().String("symbol", "", "Filter by symbol")
	history.Flags().String("order-ids", "", "At most 20 comma-separated IDs")
	history.Flags().Int32("begin", 0, "Begin Unix seconds (int32)")
	history.Flags().Int32("end", 0, "End Unix seconds (int32)")
	history.Flags().Int32("side", 0, "Side: 1=sell, 2=buy")
	addPagination(history)
	group.AddCommand(history)
	create := &cobra.Command{Use: "create", Short: "Create one stock order (never automatically retried)", RunE: runCreateOrder}
	for _, key := range []string{"symbol", "volume", "client-order-id"} {
		create.Flags().String(key, "", key+" (required)")
		if err := create.MarkFlagRequired(key); err != nil {
			panic(err)
		} // 该flag必须先声明，失败是注册错误。
	}
	create.Flags().Int32("side", 0, "Side: 1=sell, 2=buy (required)")
	if err := create.MarkFlagRequired("side"); err != nil {
		panic(err)
	} // 该flag必须先声明，失败是注册错误。
	create.Flags().String("price-type", "market", "market or limit")
	create.Flags().String("trading-session", "regular", "market=regular, limit=all")
	create.Flags().String("time-in-force", "day", "Time in force: day")
	create.Flags().String("price", "", "Positive limit price")
	group.AddCommand(create)
	update := &cobra.Command{Use: "update", Short: "Update order quantity and price", RunE: runUpdateOrder}
	update.Flags().Int64("id", 0, "Order ID (required)")
	update.Flags().String("volume", "", "New positive volume (required)")
	update.Flags().String("price", "", "New positive price (required)")
	for _, key := range []string{"id", "volume", "price"} {
		if err := update.MarkFlagRequired(key); err != nil {
			panic(err)
		} // 该flag必须先声明，失败是注册错误。
	}
	group.AddCommand(update)
	cancel := &cobra.Command{Use: "cancel", Short: "Cancel one order", RunE: runCancelOrder}
	cancel.Flags().Int64("id", 0, "Order ID (required)")
	if err := cancel.MarkFlagRequired("id"); err != nil {
		panic(err)
	} // 该flag必须先声明，失败是注册错误。
	group.AddCommand(cancel, &cobra.Command{Use: "cancel-all", Short: "Cancel all open stock orders", RunE: runCancelAll})
	Cmd.AddCommand(group)
}
func runOrders(cmd *cobra.Command, _ []string) error {
	if err := symbol(flagString(cmd, "symbol"), false); err != nil {
		return err
	}
	c, err := sdkClient(cmd, true)
	if err != nil {
		return err
	}
	result, response, err := c.StockAPI.QueryStockOrderList(c.Context(), &gateapi.QueryStockOrderListOpts{Symbol: optionalString(cmd, "symbol")})
	return finish(cmd, result, response, err, "GET", "/stock/orders")
}
func runOrderHistory(cmd *cobra.Command, _ []string) error {
	if err := pagination(cmd); err != nil {
		return err
	}
	if err := symbol(flagString(cmd, "symbol"), false); err != nil {
		return err
	}
	if err := validateOrderIDs(flagString(cmd, "order-ids")); err != nil {
		return err
	}
	begin, end := flagInt32(cmd, "begin"), flagInt32(cmd, "end")
	if err := validateTimes(int64(begin), int64(end)); err != nil {
		return err
	}
	side := flagInt32(cmd, "side")
	if cmd.Flags().Changed("side") && side != 1 && side != 2 {
		return invalid("side must be 1 or 2")
	}
	opts := &gateapi.QueryStockOrderHistoryOpts{Symbol: optionalString(cmd, "symbol"), OrderIds: optionalString(cmd, "order-ids"), Page: optional.NewInt32(flagInt32(cmd, "page")), PageSize: optional.NewInt32(flagInt32(cmd, "page-size"))}
	if cmd.Flags().Changed("begin") {
		opts.BeginTime = optional.NewInt32(begin)
	}
	if cmd.Flags().Changed("end") {
		opts.EndTime = optional.NewInt32(end)
	}
	if cmd.Flags().Changed("side") {
		opts.Side = optional.NewInt32(side)
	}
	c, err := sdkClient(cmd, true)
	if err != nil {
		return err
	}
	result, response, err := c.StockAPI.QueryStockOrderHistory(c.Context(), opts)
	return finish(cmd, result, response, err, "GET", "/stock/orders/history")
}
func runCreateOrder(cmd *cobra.Command, _ []string) error {
	volume, price, symbolValue, kind, session := flagString(cmd, "volume"), flagString(cmd, "price"), flagString(cmd, "symbol"), flagString(cmd, "price-type"), flagString(cmd, "trading-session")
	if err := symbol(symbolValue, true); err != nil {
		return err
	}
	if err := positiveDecimal(volume); err != nil {
		return err
	}
	if err := text(flagString(cmd, "client-order-id"), "client-order-id", true); err != nil {
		return err
	}
	side := flagInt32(cmd, "side")
	if side != 1 && side != 2 {
		return invalid("side must be 1 (sell) or 2 (buy)")
	}
	if flagString(cmd, "time-in-force") != "day" {
		return invalid("time-in-force must be day")
	}
	// 公共SDK规定market仅regular，limit仅all；禁止把无效组合交给网关后产生歧义。
	switch kind {
	case "market":
		if session != "regular" || cmd.Flags().Changed("price") {
			return invalid("market orders require regular session and no price")
		}
	case "limit":
		if session != "all" {
			return invalid("limit orders require all session")
		}
		if err := positiveDecimal(price); err != nil {
			return err
		}
	default:
		return invalid("price-type must be market or limit")
	}
	request := gateapi.TradFiSpotOrderRequest{Volume: volume, Symbol: symbolValue, Side: side, PriceType: kind, TradingSession: session, TimeInForce: "day", Price: price, ClientOrderId: flagString(cmd, "client-order-id")}
	c, err := sdkClient(cmd, true)
	if err != nil {
		return err
	}
	result, response, err := c.StockAPI.CreateStockOrder(c.Context(), request)
	return finish(cmd, result, response, err, "POST", "/stock/orders")
}
func runUpdateOrder(cmd *cobra.Command, _ []string) error {
	id, err := positiveID(cmd)
	if err != nil {
		return err
	}
	volume, price := flagString(cmd, "volume"), flagString(cmd, "price")
	if err := positiveDecimal(volume); err != nil {
		return err
	}
	if err := positiveDecimal(price); err != nil {
		return err
	}
	c, err := sdkClient(cmd, true)
	if err != nil {
		return err
	}
	result, response, err := c.StockAPI.UpdateStockOrder(c.Context(), id, gateapi.TradFiSpotOrderUpdateRequest{Volume: volume, Price: price})
	return finish(cmd, result, response, err, "PUT", fmt.Sprintf("/stock/orders/%d", id))
}
func runCancelOrder(cmd *cobra.Command, _ []string) error {
	id, err := positiveID(cmd)
	if err != nil {
		return err
	}
	c, err := sdkClient(cmd, true)
	if err != nil {
		return err
	}
	result, response, err := c.StockAPI.DeleteStockOrder(c.Context(), id)
	return finish(cmd, result, response, err, "DELETE", fmt.Sprintf("/stock/orders/%d", id))
}
func runCancelAll(cmd *cobra.Command, _ []string) error {
	c, err := sdkClient(cmd, true)
	if err != nil {
		return err
	}
	result, response, err := c.StockAPI.DeleteAllStockOrders(c.Context())
	return finish(cmd, result, response, err, "DELETE", "/stock/orders")
}
