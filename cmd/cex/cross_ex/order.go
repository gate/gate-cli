package crossex

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
	Short: "Cross-exchange order commands",
}

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all current open orders",
		RunE:  runOrderList,
	}
	listCmd.Flags().String("symbol", "", "Filter by symbol")
	listCmd.Flags().String("exchange-type", "", "Filter by exchange")
	listCmd.Flags().String("business-type", "", "Filter by business type")

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get order by ID",
		RunE:  runOrderGet,
	}
	getCmd.Flags().String("order-id", "", "Order ID (required)")
	getCmd.MarkFlagRequired("order-id")

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create an order",
		RunE:  runOrderCreate,
	}
	createCmd.Flags().String("json", "", "JSON body for order request (required)")
	createCmd.MarkFlagRequired("json")

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Modify an order",
		RunE:  runOrderUpdate,
	}
	updateCmd.Flags().String("order-id", "", "Order ID or Text (required)")
	updateCmd.MarkFlagRequired("order-id")
	updateCmd.Flags().String("json", "", "JSON body for update request (required)")
	updateCmd.MarkFlagRequired("json")

	cancelCmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel an order",
		RunE:  runOrderCancel,
	}
	cancelCmd.Flags().String("order-id", "", "Order ID (required)")
	cancelCmd.MarkFlagRequired("order-id")

	historyCmd := &cobra.Command{
		Use:   "history",
		Short: "Query order history",
		RunE:  runOrderHistory,
	}
	historyCmd.Flags().Int32("page", 0, "Page number")
	historyCmd.Flags().Int32("limit", 0, "Max records")
	historyCmd.Flags().String("symbol", "", "Filter by symbol")
	historyCmd.Flags().Int32("from", 0, "Start millisecond timestamp")
	historyCmd.Flags().Int32("to", 0, "End millisecond timestamp")

	tradesCmd := &cobra.Command{
		Use:   "trades",
		Short: "Query filled trade history",
		RunE:  runOrderTrades,
	}
	tradesCmd.Flags().Int32("page", 0, "Page number")
	tradesCmd.Flags().Int32("limit", 0, "Max records")
	tradesCmd.Flags().String("symbol", "", "Filter by symbol")
	tradesCmd.Flags().Int32("from", 0, "Start millisecond timestamp")
	tradesCmd.Flags().Int32("to", 0, "End millisecond timestamp")

	orderCmd.AddCommand(listCmd, getCmd, createCmd, updateCmd, cancelCmd, historyCmd, tradesCmd)
	Cmd.AddCommand(orderCmd)
}

func runOrderList(cmd *cobra.Command, args []string) error {
	symbol, _ := cmd.Flags().GetString("symbol")
	exchangeType, _ := cmd.Flags().GetString("exchange-type")
	businessType, _ := cmd.Flags().GetString("business-type")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListCrossexOpenOrdersOpts{}
	if symbol != "" {
		opts.Symbol = optional.NewString(symbol)
	}
	if exchangeType != "" {
		opts.ExchangeType = optional.NewString(exchangeType)
	}
	if businessType != "" {
		opts.BusinessType = optional.NewString(businessType)
	}

	result, httpResp, err := c.CrossExAPI.ListCrossexOpenOrders(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/crossex/orders", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{r.OrderId, r.Symbol, r.Side, r.Type, r.Qty, r.Price, r.State, r.CreateTime}
	}
	return p.Table([]string{"Order ID", "Symbol", "Side", "Type", "Qty", "Price", "State", "Created"}, rows)
}

func runOrderGet(cmd *cobra.Command, args []string) error {
	orderID, _ := cmd.Flags().GetString("order-id")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.CrossExAPI.GetCrossexOrder(c.Context(), orderID)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/crossex/orders/"+orderID, ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"Order ID", "Symbol", "Side", "Type", "Qty", "Price", "State", "Executed Qty", "Avg Price"},
		[][]string{{result.OrderId, result.Symbol, result.Side, result.Type, result.Qty, result.Price, result.State, result.ExecutedQty, result.ExecutedAvgPrice}},
	)
}

func runOrderCreate(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.CrossexOrderRequest
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}

	opts := &gateapi.CreateCrossexOrderOpts{
		CrossexOrderRequest: optional.NewInterface(body),
	}
	result, httpResp, err := c.CrossExAPI.CreateCrossexOrder(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/crossex/orders", jsonStr))
		return nil
	}
	return p.Print(result)
}

func runOrderUpdate(cmd *cobra.Command, args []string) error {
	orderID, _ := cmd.Flags().GetString("order-id")
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.CrossexOrderUpdateRequest
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}

	opts := &gateapi.UpdateCrossexOrderOpts{
		CrossexOrderUpdateRequest: optional.NewInterface(body),
	}
	result, httpResp, err := c.CrossExAPI.UpdateCrossexOrder(c.Context(), orderID, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "PUT", "/api/v4/crossex/orders/"+orderID, jsonStr))
		return nil
	}
	return p.Print(result)
}

func runOrderCancel(cmd *cobra.Command, args []string) error {
	orderID, _ := cmd.Flags().GetString("order-id")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.CrossExAPI.CancelCrossexOrder(c.Context(), orderID)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "DELETE", "/api/v4/crossex/orders/"+orderID, ""))
		return nil
	}
	return p.Print(result)
}

func runOrderHistory(cmd *cobra.Command, args []string) error {
	page, _ := cmd.Flags().GetInt32("page")
	limit, _ := cmd.Flags().GetInt32("limit")
	symbol, _ := cmd.Flags().GetString("symbol")
	from, _ := cmd.Flags().GetInt32("from")
	to, _ := cmd.Flags().GetInt32("to")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListCrossexHistoryOrdersOpts{}
	if page != 0 {
		opts.Page = optional.NewInt32(page)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if symbol != "" {
		opts.Symbol = optional.NewString(symbol)
	}
	if from != 0 {
		opts.From = optional.NewInt32(from)
	}
	if to != 0 {
		opts.To = optional.NewInt32(to)
	}

	result, httpResp, err := c.CrossExAPI.ListCrossexHistoryOrders(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/crossex/history_orders", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{r.OrderId, r.Symbol, r.Side, r.Type, r.Qty, r.Price, r.State, r.ExecutedQty, r.CreateTime}
	}
	return p.Table([]string{"Order ID", "Symbol", "Side", "Type", "Qty", "Price", "State", "Exec Qty", "Created"}, rows)
}

func runOrderTrades(cmd *cobra.Command, args []string) error {
	page, _ := cmd.Flags().GetInt32("page")
	limit, _ := cmd.Flags().GetInt32("limit")
	symbol, _ := cmd.Flags().GetString("symbol")
	from, _ := cmd.Flags().GetInt32("from")
	to, _ := cmd.Flags().GetInt32("to")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListCrossexHistoryTradesOpts{}
	if page != 0 {
		opts.Page = optional.NewInt32(page)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if symbol != "" {
		opts.Symbol = optional.NewString(symbol)
	}
	if from != 0 {
		opts.From = optional.NewInt32(from)
	}
	if to != 0 {
		opts.To = optional.NewInt32(to)
	}

	result, httpResp, err := c.CrossExAPI.ListCrossexHistoryTrades(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/crossex/history_trades", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{r.TransactionId, r.OrderId, r.Symbol, r.Side, r.Qty, r.Price, r.Fee, r.FeeCoin, r.CreateTime}
	}
	return p.Table([]string{"Trade ID", "Order ID", "Symbol", "Side", "Qty", "Price", "Fee", "Fee Coin", "Created"}, rows)
}
