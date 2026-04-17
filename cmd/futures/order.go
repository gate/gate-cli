package futures

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
	Short: "Futures order commands",
}

func init() {
	// Direction commands sharing the same flags
	for _, def := range []struct{ use, short string }{
		{"long", "Open or add to a long position (positive size)"},
		{"short", "Open or add to a short position (negative size)"},
		{"add", "Add to existing position (same direction as current position)"},
		{"remove", "Reduce existing position (opposite direction)"},
	} {
		use, short := def.use, def.short
		cmd := &cobra.Command{
			Use:   use,
			Short: short,
			RunE:  func(c *cobra.Command, args []string) error { return runFuturesDirectionOrder(c, use) },
		}
		addFuturesOrderFlags(cmd)
		orderCmd.AddCommand(cmd)
	}

	// close
	closeCmd := &cobra.Command{
		Use:   "close",
		Short: "Close position (full close by default, partial close with --size)",
		RunE:  runFuturesClose,
	}
	closeCmd.Flags().String("contract", "", "Contract name, e.g. BTC_USDT (required)")
	closeCmd.Flags().String("size", "0", "Partial close size (default 0 = full close)")
	closeCmd.Flags().String("side", "long", "Position side to close: long or short (relevant in dual-position mode)")
	closeCmd.MarkFlagRequired("contract")
	addSettleFlag(closeCmd)

	// get
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get a futures order by ID",
		RunE:  runFuturesOrderGet,
	}
	getCmd.Flags().Int64("id", 0, "Order ID (required)")
	getCmd.MarkFlagRequired("id")
	addSettleFlag(getCmd)

	// list
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List futures orders",
		RunE:  runFuturesOrderList,
	}
	listCmd.Flags().String("contract", "", "Filter by contract name")
	listCmd.Flags().String("status", "open", "Order status: open, finished")
	listCmd.Flags().Int32("limit", 20, "Number of orders to return")
	addSettleFlag(listCmd)

	// cancel
	cancelCmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel one order (--id) or all open orders (--all --contract)",
		RunE:  runFuturesOrderCancel,
	}
	cancelCmd.Flags().Int64("id", 0, "Order ID to cancel")
	cancelCmd.Flags().String("contract", "", "Contract name (required with --all)")
	cancelCmd.Flags().Bool("all", false, "Cancel all open orders for the contract")
	addSettleFlag(cancelCmd)

	myTradesCmd := &cobra.Command{
		Use:   "my-trades",
		Short: "List personal futures trade history",
		RunE:  runFuturesMyTrades,
	}
	myTradesCmd.Flags().String("contract", "", "Filter by contract name")
	myTradesCmd.Flags().Int32("limit", 0, "Number of records to return")
	addSettleFlag(myTradesCmd)

	myTradesTimerangeCmd := &cobra.Command{
		Use:   "my-trades-timerange",
		Short: "List personal futures trade history by time range",
		RunE:  runFuturesMyTradesTimerange,
	}
	myTradesTimerangeCmd.Flags().String("contract", "", "Filter by contract name")
	myTradesTimerangeCmd.Flags().Int64("from", 0, "Start Unix timestamp")
	myTradesTimerangeCmd.Flags().Int64("to", 0, "End Unix timestamp")
	myTradesTimerangeCmd.Flags().Int32("limit", 0, "Number of records to return")
	addSettleFlag(myTradesTimerangeCmd)

	amendCmd := &cobra.Command{
		Use:   "amend",
		Short: "Amend a futures order",
		RunE:  runFuturesAmendOrder,
	}
	amendCmd.Flags().Int64("id", 0, "Order ID (required)")
	amendCmd.Flags().String("size", "", "New order size (including filled)")
	amendCmd.Flags().String("price", "", "New order price")
	amendCmd.MarkFlagRequired("id")
	addSettleFlag(amendCmd)

	batchCreateCmd := &cobra.Command{
		Use:   "batch-create",
		Short: "Batch create futures orders (JSON array)",
		RunE:  runFuturesBatchCreateOrders,
	}
	batchCreateCmd.Flags().String("orders-json", "", "JSON array of FuturesOrder objects (required)")
	batchCreateCmd.MarkFlagRequired("orders-json")
	addSettleFlag(batchCreateCmd)

	batchCancelCmd := &cobra.Command{
		Use:   "batch-cancel",
		Short: "Cancel futures orders by ID list",
		RunE:  runFuturesBatchCancelOrders,
	}
	batchCancelCmd.Flags().StringSlice("ids", nil, "Order IDs to cancel (required)")
	batchCancelCmd.MarkFlagRequired("ids")
	addSettleFlag(batchCancelCmd)

	batchAmendCmd := &cobra.Command{
		Use:   "batch-amend",
		Short: "Batch amend futures orders",
		RunE:  runFuturesBatchAmendOrders,
	}
	batchAmendCmd.Flags().String("orders-json", "", "JSON array of BatchAmendOrderReq objects (required)")
	batchAmendCmd.MarkFlagRequired("orders-json")
	addSettleFlag(batchAmendCmd)

	bboCmd := &cobra.Command{
		Use:   "bbo",
		Short: "Create a best bid/offer (BBO) futures order",
		RunE:  runFuturesBBOOrder,
	}
	bboCmd.Flags().String("contract", "", "Contract name (required)")
	bboCmd.Flags().Int64("size", 0, "Number of contracts, positive=buy, negative=sell (required)")
	bboCmd.Flags().String("direction", "", "Direction: buy or sell (required)")
	bboCmd.MarkFlagRequired("contract")
	bboCmd.MarkFlagRequired("size")
	bboCmd.MarkFlagRequired("direction")
	addSettleFlag(bboCmd)

	countdownCmd := &cobra.Command{
		Use:   "countdown-cancel-all",
		Short: "Set countdown to cancel all futures orders",
		RunE:  runFuturesCountdownCancelAll,
	}
	countdownCmd.Flags().Int32("timeout", 0, "Countdown in seconds (0 = cancel countdown, min 5) (required)")
	countdownCmd.Flags().String("contract", "", "Limit cancellation to this contract")
	countdownCmd.MarkFlagRequired("timeout")
	addSettleFlag(countdownCmd)

	listTimerangeCmd := &cobra.Command{
		Use:   "list-timerange",
		Short: "List futures orders by time range",
		RunE:  runFuturesOrdersTimerange,
	}
	listTimerangeCmd.Flags().String("contract", "", "Filter by contract name")
	listTimerangeCmd.Flags().Int64("from", 0, "Start Unix timestamp")
	listTimerangeCmd.Flags().Int64("to", 0, "End Unix timestamp")
	listTimerangeCmd.Flags().Int32("limit", 0, "Number of records to return")
	addSettleFlag(listTimerangeCmd)

	orderCmd.AddCommand(closeCmd, getCmd, listCmd, cancelCmd,
		myTradesCmd, myTradesTimerangeCmd, amendCmd,
		batchCreateCmd, batchCancelCmd, batchAmendCmd,
		bboCmd, countdownCmd, listTimerangeCmd)
	Cmd.AddCommand(orderCmd)
}

func addFuturesOrderFlags(cmd *cobra.Command) {
	cmd.Flags().String("contract", "", "Contract name, e.g. BTC_USDT (required)")
	cmd.Flags().String("size", "", "Number of contracts (required)")
	cmd.Flags().String("price", "", "Limit price (omit for market order)")
	cmd.MarkFlagRequired("contract")
	cmd.MarkFlagRequired("size")
	addSettleFlag(cmd)
}

// positionIsShort returns true if the current position is short (negative size).
// Returns an error if no open position exists.
func positionIsShort(positions []gateapi.Position) (bool, error) {
	for _, pos := range positions {
		if pos.Size == "" || pos.Size == "0" {
			continue
		}
		return len(pos.Size) > 0 && pos.Size[0] == '-', nil
	}
	return false, fmt.Errorf("no open position found")
}

// applyDirectionSign computes the signed size and reduce-only flag for add/remove orders.
// sizeStr is the raw (positive) size from the user flag.
// isShort indicates whether the existing position is short.
func applyDirectionSign(direction, sizeStr string, isShort bool) (finalSize string, reduceOnly bool) {
	finalSize = sizeStr
	if direction == "add" {
		// Add in the same direction as the existing position.
		if isShort && len(sizeStr) > 0 && sizeStr[0] != '-' {
			finalSize = "-" + sizeStr
		}
	} else {
		// Remove: opposite direction + reduce-only.
		if isShort {
			// Buy to reduce short → positive size.
			if len(sizeStr) > 0 && sizeStr[0] == '-' {
				finalSize = sizeStr[1:]
			}
		} else {
			// Sell to reduce long → negative size.
			if len(sizeStr) > 0 && sizeStr[0] != '-' {
				finalSize = "-" + sizeStr
			}
		}
		reduceOnly = true
	}
	return
}

// closePartialSingleSize returns the signed contract size for a single-mode partial close.
// sizeStr is the raw (positive) size from the user flag.
func closePartialSingleSize(sizeStr string, isShort bool) string {
	if isShort {
		return sizeStr // positive = buy = reduce short
	}
	return "-" + sizeStr // negative = sell = reduce long
}

// applyFuturesTif sets Price and Tif on a FuturesOrder based on whether
// a limit price was provided. Market orders use Price="0" and Tif="ioc";
// limit orders use the given price with Tif="gtc".
func applyFuturesTif(order *gateapi.FuturesOrder, price string) {
	if price == "" {
		order.Price = "0"
		order.Tif = "ioc"
	} else {
		order.Price = price
		order.Tif = "gtc"
	}
}

// runFuturesDirectionOrder handles long/short/add/remove:
//
//	long   → +size (open or add to long)
//	short  → -size (open or add to short)
//	add    → queries current position; uses same direction as existing position
//	remove → queries current position; uses opposite direction with reduce-only
func runFuturesDirectionOrder(cmd *cobra.Command, direction string) error {
	contract, _ := cmd.Flags().GetString("contract")
	sizeStr, _ := cmd.Flags().GetString("size")
	price, _ := cmd.Flags().GetString("price")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	finalSize := sizeStr
	var reduceOnly bool

	switch direction {
	case "long":
		// positive size — already set
	case "short":
		if len(sizeStr) > 0 && sizeStr[0] != '-' {
			finalSize = "-" + sizeStr
		}
	case "add", "remove":
		// Query current position to determine the correct direction.
		positions, httpResp, posErr := c.GetFuturesPosition(settle, contract)
		if posErr != nil {
			p.PrintError(client.ParseGateError(posErr, httpResp, "GET", "/api/v4/futures/"+settle+"/positions/"+contract, ""))
			return nil
		}
		isShort, posErr := positionIsShort(positions)
		if posErr != nil {
			return fmt.Errorf("cannot %s: %w", direction, posErr)
		}
		finalSize, reduceOnly = applyDirectionSign(direction, sizeStr, isShort)
	}

	order := gateapi.FuturesOrder{
		Contract:   contract,
		Size:       finalSize,
		ReduceOnly: reduceOnly,
	}
	applyFuturesTif(&order, price)

	body, _ := json.Marshal(order)
	result, httpResp, err := c.FuturesAPI.CreateFuturesOrder(c.Context(), settle, order, nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/futures/"+settle+"/orders", string(body)))
		return nil
	}
	return p.Print(result)
}

func runFuturesClose(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	sizeStr, _ := cmd.Flags().GetString("size")
	side, _ := cmd.Flags().GetString("side")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	isDual := c.IsDualMode(settle)
	isFullClose := sizeStr == "0" || sizeStr == ""

	order := gateapi.FuturesOrder{
		Contract: contract,
	}
	applyFuturesTif(&order, "") // close is always a market order

	switch {
	case isDual && isFullClose:
		// Dual mode full close: AutoSize tells Gate which side to flatten.
		// "close_long" submits a sell order of the full long size;
		// "close_short" submits a buy order of the full short size.
		autoSize := "close_long"
		if side == "short" {
			autoSize = "close_short"
		}
		order.Size = "0"
		order.AutoSize = autoSize

	case isDual && !isFullClose:
		// Dual mode partial close: reduce-only with explicit signed size.
		// Selling reduces longs; buying reduces shorts.
		if side == "short" {
			order.Size = sizeStr // positive = buy = reduce short
		} else {
			order.Size = "-" + sizeStr // negative = sell = reduce long
		}
		order.ReduceOnly = true

	case !isDual && isFullClose:
		// Single mode full close: gate closes whichever direction is open.
		order.Size = "0"
		order.Close = true

	default:
		// Single mode partial close: query position to determine direction.
		positions, httpResp2, posErr := c.GetFuturesPosition(settle, contract)
		if posErr != nil {
			p.PrintError(client.ParseGateError(posErr, httpResp2, "GET", "/api/v4/futures/"+settle+"/positions/"+contract, ""))
			return nil
		}
		isShort, posErr := positionIsShort(positions)
		if posErr != nil {
			return fmt.Errorf("cannot partial close: %w", posErr)
		}
		order.Size = closePartialSingleSize(sizeStr, isShort)
		order.ReduceOnly = true
	}

	body, _ := json.Marshal(order)
	result, httpResp, err := c.FuturesAPI.CreateFuturesOrder(c.Context(), settle, order, nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/futures/"+settle+"/orders", string(body)))
		return nil
	}
	return p.Print(result)
}

func runFuturesOrderGet(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetInt64("id")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.FuturesAPI.GetFuturesOrder(c.Context(), settle, fmt.Sprintf("%d", id))
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", fmt.Sprintf("/api/v4/futures/%s/orders/%d", settle, id), ""))
		return nil
	}
	return p.Print(result)
}

func runFuturesOrderList(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	status, _ := cmd.Flags().GetString("status")
	limit, _ := cmd.Flags().GetInt32("limit")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListFuturesOrdersOpts{
		Limit: optional.NewInt32(limit),
	}
	if contract != "" {
		opts.Contract = optional.NewString(contract)
	}

	orders, httpResp, err := c.FuturesAPI.ListFuturesOrders(c.Context(), settle, status, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/orders", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(orders)
	}
	rows := make([][]string, len(orders))
	for i, o := range orders {
		rows[i] = []string{fmt.Sprintf("%d", o.Id), o.Contract, o.Size, o.Price, o.Tif, o.Status}
	}
	return p.Table([]string{"ID", "Contract", "Size", "Price", "TIF", "Status"}, rows)
}

func runFuturesOrderCancel(cmd *cobra.Command, args []string) error {
	settle := cmdutil.GetSettle(cmd)
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
		contract, _ := cmd.Flags().GetString("contract")
		if contract == "" {
			return fmt.Errorf("--contract is required when using --all")
		}
		cancelled, httpResp, err := c.FuturesAPI.CancelFuturesOrders(c.Context(), settle, &gateapi.CancelFuturesOrdersOpts{
			Contract: optional.NewString(contract),
		})
		if err != nil {
			p.PrintError(client.ParseGateError(err, httpResp, "DELETE", "/api/v4/futures/"+settle+"/orders", ""))
			return nil
		}
		return p.Print(cancelled)
	}

	id, _ := cmd.Flags().GetInt64("id")
	if id == 0 {
		return fmt.Errorf("provide --id <order-id> or --all to cancel all open orders")
	}
	result, httpResp, err := c.FuturesAPI.CancelFuturesOrder(c.Context(), settle, fmt.Sprintf("%d", id), nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "DELETE", fmt.Sprintf("/api/v4/futures/%s/orders/%d", settle, id), ""))
		return nil
	}
	return p.Print(result)
}

func runFuturesMyTrades(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	limit, _ := cmd.Flags().GetInt32("limit")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.GetMyTradesOpts{}
	if contract != "" {
		opts.Contract = optional.NewString(contract)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.FuturesAPI.GetMyTrades(c.Context(), settle, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/my_trades", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, t := range result {
		rows[i] = []string{fmt.Sprintf("%d", t.Id), t.Contract, t.OrderId, t.Size, t.Price, fmt.Sprintf("%g", t.CreateTime)}
	}
	return p.Table([]string{"ID", "Contract", "Order ID", "Size", "Price", "Time"}, rows)
}

func runFuturesMyTradesTimerange(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	from, _ := cmd.Flags().GetInt64("from")
	to, _ := cmd.Flags().GetInt64("to")
	limit, _ := cmd.Flags().GetInt32("limit")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.GetMyTradesWithTimeRangeOpts{}
	if contract != "" {
		opts.Contract = optional.NewString(contract)
	}
	if from != 0 {
		opts.From = optional.NewInt64(from)
	}
	if to != 0 {
		opts.To = optional.NewInt64(to)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.FuturesAPI.GetMyTradesWithTimeRange(c.Context(), settle, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/my_trades/timerange", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, t := range result {
		rows[i] = []string{t.TradeId, t.Contract, t.OrderId, t.Size, fmt.Sprintf("%g", t.CreateTime)}
	}
	return p.Table([]string{"Trade ID", "Contract", "Order ID", "Size", "Time"}, rows)
}

func runFuturesAmendOrder(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetInt64("id")
	size, _ := cmd.Flags().GetString("size")
	price, _ := cmd.Flags().GetString("price")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	amendment := gateapi.FuturesOrderAmendment{}
	if size != "" {
		amendment.Size = size
	}
	if price != "" {
		amendment.Price = price
	}
	body, _ := json.Marshal(amendment)
	result, httpResp, err := c.FuturesAPI.AmendFuturesOrder(c.Context(), settle, fmt.Sprintf("%d", id), amendment, nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "PUT", fmt.Sprintf("/api/v4/futures/%s/orders/%d", settle, id), string(body)))
		return nil
	}
	return p.Print(result)
}

func runFuturesBatchCreateOrders(cmd *cobra.Command, args []string) error {
	ordersJSON, _ := cmd.Flags().GetString("orders-json")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var orders []gateapi.FuturesOrder
	if err := json.Unmarshal([]byte(ordersJSON), &orders); err != nil {
		return fmt.Errorf("invalid --orders-json: %w", err)
	}
	result, httpResp, err := c.FuturesAPI.CreateBatchFuturesOrder(c.Context(), settle, orders, nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/futures/"+settle+"/batch_orders", ordersJSON))
		return nil
	}
	return p.Print(result)
}

func runFuturesBatchCancelOrders(cmd *cobra.Command, args []string) error {
	ids, _ := cmd.Flags().GetStringSlice("ids")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.FuturesAPI.CancelBatchFutureOrders(c.Context(), settle, ids, nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/futures/"+settle+"/batch_cancel_orders", ""))
		return nil
	}
	return p.Print(result)
}

func runFuturesBatchAmendOrders(cmd *cobra.Command, args []string) error {
	ordersJSON, _ := cmd.Flags().GetString("orders-json")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var items []gateapi.BatchAmendOrderReq
	if err := json.Unmarshal([]byte(ordersJSON), &items); err != nil {
		return fmt.Errorf("invalid --orders-json: %w", err)
	}
	result, httpResp, err := c.FuturesAPI.AmendBatchFutureOrders(c.Context(), settle, items, nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/futures/"+settle+"/batch_amend_orders", ordersJSON))
		return nil
	}
	return p.Print(result)
}

func runFuturesBBOOrder(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	size, _ := cmd.Flags().GetInt64("size")
	direction, _ := cmd.Flags().GetString("direction")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	req := gateapi.FuturesBboOrder{Contract: contract, Size: size, Direction: direction}
	body, _ := json.Marshal(req)
	result, httpResp, err := c.FuturesAPI.CreateFuturesBBOOrder(c.Context(), settle, req, nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/futures/"+settle+"/bbo_order", string(body)))
		return nil
	}
	return p.Print(result)
}

func runFuturesCountdownCancelAll(cmd *cobra.Command, args []string) error {
	timeout, _ := cmd.Flags().GetInt32("timeout")
	contract, _ := cmd.Flags().GetString("contract")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	req := gateapi.CountdownCancelAllFuturesTask{Timeout: timeout, Contract: contract}
	body, _ := json.Marshal(req)
	result, httpResp, err := c.FuturesAPI.CountdownCancelAllFutures(c.Context(), settle, req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/futures/"+settle+"/countdown_cancel_all", string(body)))
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

func runFuturesOrdersTimerange(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	from, _ := cmd.Flags().GetInt64("from")
	to, _ := cmd.Flags().GetInt64("to")
	limit, _ := cmd.Flags().GetInt32("limit")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.GetOrdersWithTimeRangeOpts{}
	if contract != "" {
		opts.Contract = optional.NewString(contract)
	}
	if from != 0 {
		opts.From = optional.NewInt64(from)
	}
	if to != 0 {
		opts.To = optional.NewInt64(to)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.FuturesAPI.GetOrdersWithTimeRange(c.Context(), settle, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/orders_timerange", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, o := range result {
		rows[i] = []string{fmt.Sprintf("%d", o.Id), o.Contract, o.Size, o.Price, o.Tif, o.Status}
	}
	return p.Table([]string{"ID", "Contract", "Size", "Price", "TIF", "Status"}, rows)
}
