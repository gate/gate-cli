package futures

import (
	"encoding/json"
	"fmt"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
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
	closeCmd.Flags().String("size", "0", "Partial close size (default 0 = full close via close flag)")
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

	orderCmd.AddCommand(closeCmd, getCmd, listCmd, cancelCmd)
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

// runFuturesDirectionOrder handles long/short/add/remove with sign conversion:
//
//	long  → +size  (buy / open long)
//	short → -size  (sell / open short)
//	add   → +size  (increase position, same direction assumed)
//	remove→ -size  (decrease position)
func runFuturesDirectionOrder(cmd *cobra.Command, direction string) error {
	contract, _ := cmd.Flags().GetString("contract")
	sizeStr, _ := cmd.Flags().GetString("size")
	price, _ := cmd.Flags().GetString("price")
	settle := getSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	// Negate size for short/remove directions
	finalSize := sizeStr
	if (direction == "short" || direction == "remove") && len(sizeStr) > 0 && sizeStr[0] != '-' {
		finalSize = "-" + sizeStr
	}

	order := gateapi.FuturesOrder{
		Contract: contract,
		Size:     finalSize,
	}
	if price == "" {
		order.Price = "0"
		order.Tif = "ioc"
	} else {
		order.Price = price
		order.Tif = "gtc"
	}

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
	settle := getSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	order := gateapi.FuturesOrder{
		Contract: contract,
		Price:    "0",
		Tif:      "ioc",
	}
	if sizeStr == "0" || sizeStr == "" {
		// Full close: set size=0 and close=true
		order.Size = "0"
		order.Close = true
	} else {
		// Partial close via reduce-only
		order.Size = "-" + sizeStr // reduce longs; works for standard single-direction positions
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
	settle := getSettle(cmd)
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
	settle := getSettle(cmd)
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
	settle := getSettle(cmd)
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
