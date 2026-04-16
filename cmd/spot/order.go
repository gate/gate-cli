package spot

import (
	"encoding/json"
	"fmt"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

var orderCmd = &cobra.Command{
	Use:   "order",
	Short: "Spot order commands",
}

func init() {
	buyCmd := &cobra.Command{
		Use:   "buy",
		Short: "Place a buy order (--price for limit; --quote for market)",
		RunE:  func(cmd *cobra.Command, args []string) error { return runSpotCreateOrder(cmd, "buy") },
	}
	buyCmd.Flags().String("pair", "", "Currency pair, e.g. BTC_USDT (required)")
	buyCmd.Flags().String("amount", "", "Amount of base currency to buy (required for limit orders)")
	buyCmd.Flags().String("quote", "", "Amount of quote currency to spend (market buy only; e.g. --quote 10 to spend 10 USDT)")
	buyCmd.Flags().String("price", "", "Limit price (omit for market order)")
	buyCmd.MarkFlagRequired("pair")

	sellCmd := &cobra.Command{
		Use:   "sell",
		Short: "Place a sell order (omit --price for market order)",
		RunE:  func(cmd *cobra.Command, args []string) error { return runSpotCreateOrder(cmd, "sell") },
	}
	addOrderFlags(sellCmd)

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get an order by ID",
		RunE:  runSpotOrderGet,
	}
	getCmd.Flags().String("id", "", "Order ID (required)")
	getCmd.Flags().String("pair", "", "Currency pair (required)")
	getCmd.MarkFlagRequired("id")
	getCmd.MarkFlagRequired("pair")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List orders",
		RunE:  runSpotOrderList,
	}
	listCmd.Flags().String("pair", "", "Currency pair (required)")
	listCmd.Flags().String("status", "open", "Order status: open, finished")
	listCmd.Flags().Int32("limit", 20, "Number of orders to return")
	listCmd.MarkFlagRequired("pair")

	cancelCmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel one order (--id) or all open orders (--all)",
		RunE:  runSpotOrderCancel,
	}
	cancelCmd.Flags().String("id", "", "Order ID to cancel")
	cancelCmd.Flags().String("pair", "", "Currency pair (required)")
	cancelCmd.Flags().Bool("all", false, "Cancel all open orders for the pair")
	cancelCmd.MarkFlagRequired("pair")

	myTradesCmd := &cobra.Command{
		Use:   "my-trades",
		Short: "List personal trading history",
		RunE:  runSpotMyTrades,
	}
	myTradesCmd.Flags().String("pair", "", "Filter by currency pair")
	myTradesCmd.Flags().Int32("limit", 0, "Number of records to return")
	myTradesCmd.Flags().Int64("from", 0, "Start Unix timestamp")
	myTradesCmd.Flags().Int64("to", 0, "End Unix timestamp")

	amendCmd := &cobra.Command{
		Use:   "amend",
		Short: "Amend an order (change amount or price)",
		RunE:  runSpotAmendOrder,
	}
	amendCmd.Flags().String("id", "", "Order ID (required)")
	amendCmd.Flags().String("pair", "", "Currency pair (required)")
	amendCmd.Flags().String("amount", "", "New amount")
	amendCmd.Flags().String("price", "", "New price")
	amendCmd.MarkFlagRequired("id")
	amendCmd.MarkFlagRequired("pair")

	batchCreateCmd := &cobra.Command{
		Use:   "batch-create",
		Short: "Batch place orders (JSON array of orders)",
		RunE:  runSpotBatchCreateOrders,
	}
	batchCreateCmd.Flags().String("orders-json", "", "JSON array of orders (required)")
	batchCreateCmd.MarkFlagRequired("orders-json")

	batchCancelCmd := &cobra.Command{
		Use:   "batch-cancel",
		Short: "Cancel orders by ID list",
		RunE:  runSpotBatchCancelOrders,
	}
	batchCancelCmd.Flags().String("orders-json", "", "JSON array of {currency_pair, id} objects (required)")
	batchCancelCmd.MarkFlagRequired("orders-json")

	batchAmendCmd := &cobra.Command{
		Use:   "batch-amend",
		Short: "Batch amend orders",
		RunE:  runSpotBatchAmendOrders,
	}
	batchAmendCmd.Flags().String("orders-json", "", "JSON array of BatchAmendItem objects (required)")
	batchAmendCmd.MarkFlagRequired("orders-json")

	allOpenCmd := &cobra.Command{
		Use:   "all-open",
		Short: "List all open orders across all currency pairs",
		RunE:  runSpotAllOpenOrders,
	}
	allOpenCmd.Flags().Int32("limit", 0, "Number of records per page")

	crossLiquidateCmd := &cobra.Command{
		Use:   "cross-liquidate",
		Short: "Create a cross margin liquidation order",
		RunE:  runSpotCrossLiquidate,
	}
	crossLiquidateCmd.Flags().String("pair", "", "Currency pair (required)")
	crossLiquidateCmd.Flags().String("amount", "", "Trade amount (required)")
	crossLiquidateCmd.Flags().String("price", "", "Order price (required)")
	crossLiquidateCmd.MarkFlagRequired("pair")
	crossLiquidateCmd.MarkFlagRequired("amount")
	crossLiquidateCmd.MarkFlagRequired("price")

	countdownCmd := &cobra.Command{
		Use:   "countdown-cancel-all",
		Short: "Set countdown to cancel all spot orders",
		RunE:  runSpotCountdownCancelAll,
	}
	countdownCmd.Flags().Int32("timeout", 0, "Countdown in seconds (0 = cancel countdown, min 5) (required)")
	countdownCmd.Flags().String("pair", "", "Limit cancellation to this currency pair")
	countdownCmd.MarkFlagRequired("timeout")

	orderCmd.AddCommand(buyCmd, sellCmd, getCmd, listCmd, cancelCmd,
		myTradesCmd, amendCmd, batchCreateCmd, batchCancelCmd, batchAmendCmd,
		allOpenCmd, crossLiquidateCmd, countdownCmd)
	Cmd.AddCommand(orderCmd)
}

func addOrderFlags(cmd *cobra.Command) {
	cmd.Flags().String("pair", "", "Currency pair, e.g. BTC_USDT (required)")
	cmd.Flags().String("amount", "", "Amount to trade (required)")
	cmd.Flags().String("price", "", "Limit price (omit for market order)")
	cmd.MarkFlagRequired("pair")
	cmd.MarkFlagRequired("amount")
}

// buildSpotOrder assembles the Order struct from CLI arguments and validates
// buy-specific flag semantics. For market buy orders Gate treats Amount as
// quote currency (e.g. USDT to spend), so the CLI requires --quote to make
// this explicit and rejects --amount to prevent confusion.
func buildSpotOrder(side, pair, amount, price, quote string) (gateapi.Order, error) {
	order := gateapi.Order{CurrencyPair: pair, Side: side}
	if price == "" {
		order.Type = "market"
		order.TimeInForce = "ioc"
		if side == "buy" {
			// Gate market buy: Amount = quote currency to spend, not base amount.
			if amount != "" {
				return gateapi.Order{}, fmt.Errorf("for market buy, use --quote <quote-amount> (e.g. --quote 10 to spend 10 USDT);\n--amount specifies base currency and is not applicable to market buy orders")
			}
			if quote == "" {
				return gateapi.Order{}, fmt.Errorf("market buy requires --quote <amount> (quote currency to spend, e.g. --quote 10)")
			}
			order.Amount = quote
		} else {
			if amount == "" {
				return gateapi.Order{}, fmt.Errorf("market sell requires --amount <base-amount>")
			}
			order.Amount = amount
		}
	} else {
		order.Type = "limit"
		order.Price = price
		if amount == "" {
			return gateapi.Order{}, fmt.Errorf("limit %s requires --amount <base-amount>", side)
		}
		order.Amount = amount
	}
	return order, nil
}

func runSpotCreateOrder(cmd *cobra.Command, side string) error {
	pair, _ := cmd.Flags().GetString("pair")
	amount, _ := cmd.Flags().GetString("amount")
	price, _ := cmd.Flags().GetString("price")
	quote, _ := cmd.Flags().GetString("quote")

	order, err := buildSpotOrder(side, pair, amount, price, quote)
	if err != nil {
		return err
	}

	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body, _ := json.Marshal(order)
	result, httpResp, err := c.SpotAPI.CreateOrder(c.Context(), order, nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/spot/orders", string(body)))
		return nil
	}
	return p.Print(result)
}

func runSpotOrderGet(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetString("id")
	pair, _ := cmd.Flags().GetString("pair")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	order, httpResp, err := c.SpotAPI.GetOrder(c.Context(), id, pair, nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", fmt.Sprintf("/api/v4/spot/orders/%s", id), ""))
		return nil
	}
	return p.Print(order)
}

func runSpotOrderList(cmd *cobra.Command, args []string) error {
	pair, _ := cmd.Flags().GetString("pair")
	status, _ := cmd.Flags().GetString("status")
	limit, _ := cmd.Flags().GetInt32("limit")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	orders, httpResp, err := c.SpotAPI.ListOrders(c.Context(), pair, status, &gateapi.ListOrdersOpts{
		Limit: optional.NewInt32(limit),
	})
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/spot/orders", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(orders)
	}
	rows := make([][]string, len(orders))
	for i, o := range orders {
		rows[i] = []string{o.Id, o.CurrencyPair, o.Side, o.Type, o.Amount, o.Price, o.Status}
	}
	return p.Table([]string{"ID", "Pair", "Side", "Type", "Amount", "Price", "Status"}, rows)
}

func runSpotOrderCancel(cmd *cobra.Command, args []string) error {
	pair, _ := cmd.Flags().GetString("pair")
	all, _ := cmd.Flags().GetBool("all")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	if all {
		cancelled, httpResp, err := c.SpotAPI.CancelOrders(c.Context(), &gateapi.CancelOrdersOpts{
			CurrencyPair: optional.NewString(pair),
		})
		if err != nil {
			p.PrintError(client.ParseGateError(err, httpResp, "DELETE", "/api/v4/spot/orders", ""))
			return nil
		}
		return p.Print(cancelled)
	}

	id, _ := cmd.Flags().GetString("id")
	if id == "" {
		return fmt.Errorf("provide --id <order-id> or --all to cancel all open orders")
	}
	result, httpResp, err := c.SpotAPI.CancelOrder(c.Context(), id, pair, nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "DELETE", fmt.Sprintf("/api/v4/spot/orders/%s", id), ""))
		return nil
	}
	return p.Print(result)
}

func runSpotMyTrades(cmd *cobra.Command, args []string) error {
	pair, _ := cmd.Flags().GetString("pair")
	limit, _ := cmd.Flags().GetInt32("limit")
	from, _ := cmd.Flags().GetInt64("from")
	to, _ := cmd.Flags().GetInt64("to")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListMyTradesOpts{}
	if pair != "" {
		opts.CurrencyPair = optional.NewString(pair)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if from != 0 {
		opts.From = optional.NewInt64(from)
	}
	if to != 0 {
		opts.To = optional.NewInt64(to)
	}

	result, httpResp, err := c.SpotAPI.ListMyTrades(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/spot/my_trades", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, t := range result {
		rows[i] = []string{t.Id, t.CurrencyPair, t.Side, t.Role, t.Amount, t.Price, t.Fee, t.FeeCurrency}
	}
	return p.Table([]string{"ID", "Pair", "Side", "Role", "Amount", "Price", "Fee", "Fee Currency"}, rows)
}

func runSpotAmendOrder(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetString("id")
	pair, _ := cmd.Flags().GetString("pair")
	amount, _ := cmd.Flags().GetString("amount")
	price, _ := cmd.Flags().GetString("price")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	patch := gateapi.OrderPatch{CurrencyPair: pair}
	if amount != "" {
		patch.Amount = amount
	}
	if price != "" {
		patch.Price = price
	}
	body, _ := json.Marshal(patch)
	result, httpResp, err := c.SpotAPI.AmendOrder(c.Context(), id, patch, &gateapi.AmendOrderOpts{
		CurrencyPair: optional.NewString(pair),
	})
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "PATCH", fmt.Sprintf("/api/v4/spot/orders/%s", id), string(body)))
		return nil
	}
	return p.Print(result)
}

func runSpotBatchCreateOrders(cmd *cobra.Command, args []string) error {
	ordersJSON, _ := cmd.Flags().GetString("orders-json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var orders []gateapi.Order
	if err := json.Unmarshal([]byte(ordersJSON), &orders); err != nil {
		return fmt.Errorf("invalid --orders-json: %w", err)
	}
	result, httpResp, err := c.SpotAPI.CreateBatchOrders(c.Context(), orders, nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/spot/batch_orders", ordersJSON))
		return nil
	}
	return p.Print(result)
}

func runSpotBatchCancelOrders(cmd *cobra.Command, args []string) error {
	ordersJSON, _ := cmd.Flags().GetString("orders-json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var orders []gateapi.CancelBatchOrder
	if err := json.Unmarshal([]byte(ordersJSON), &orders); err != nil {
		return fmt.Errorf("invalid --orders-json: %w", err)
	}
	result, httpResp, err := c.SpotAPI.CancelBatchOrders(c.Context(), orders, nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/spot/cancel_batch_orders", ordersJSON))
		return nil
	}
	return p.Print(result)
}

func runSpotBatchAmendOrders(cmd *cobra.Command, args []string) error {
	ordersJSON, _ := cmd.Flags().GetString("orders-json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var items []gateapi.BatchAmendItem
	if err := json.Unmarshal([]byte(ordersJSON), &items); err != nil {
		return fmt.Errorf("invalid --orders-json: %w", err)
	}
	result, httpResp, err := c.SpotAPI.AmendBatchOrders(c.Context(), items, nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/spot/amend_batch_orders", ordersJSON))
		return nil
	}
	return p.Print(result)
}

func runSpotAllOpenOrders(cmd *cobra.Command, args []string) error {
	limit, _ := cmd.Flags().GetInt32("limit")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListAllOpenOrdersOpts{}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.SpotAPI.ListAllOpenOrders(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/spot/open_orders", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, o := range result {
		rows[i] = []string{o.CurrencyPair, fmt.Sprintf("%d", o.Total)}
	}
	return p.Table([]string{"Pair", "Open Orders"}, rows)
}

func runSpotCrossLiquidate(cmd *cobra.Command, args []string) error {
	pair, _ := cmd.Flags().GetString("pair")
	amount, _ := cmd.Flags().GetString("amount")
	price, _ := cmd.Flags().GetString("price")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	req := gateapi.LiquidateOrder{CurrencyPair: pair, Amount: amount, Price: price}
	body, _ := json.Marshal(req)
	result, httpResp, err := c.SpotAPI.CreateCrossLiquidateOrder(c.Context(), req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/spot/cross_liquidate_orders", string(body)))
		return nil
	}
	return p.Print(result)
}

func runSpotCountdownCancelAll(cmd *cobra.Command, args []string) error {
	timeout, _ := cmd.Flags().GetInt32("timeout")
	pair, _ := cmd.Flags().GetString("pair")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	req := gateapi.CountdownCancelAllSpotTask{Timeout: timeout, CurrencyPair: pair}
	body, _ := json.Marshal(req)
	result, httpResp, err := c.SpotAPI.CountdownCancelAllSpot(c.Context(), req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/spot/countdown_cancel_all", string(body)))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"Trigger Time (ms)"},
		[][]string{{fmt.Sprintf("%d", result.TriggerTime)}},
	)
}
