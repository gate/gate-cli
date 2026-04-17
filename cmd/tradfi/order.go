package tradfi

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
	Short: "TradFi order commands",
}

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List open TradFi orders",
		RunE:  runTradfiOrderList,
	}

	historyCmd := &cobra.Command{
		Use:   "history",
		Short: "List TradFi order history",
		RunE:  runTradfiOrderHistory,
	}
	historyCmd.Flags().Int64("begin", 0, "Begin timestamp (Unix seconds)")
	historyCmd.Flags().Int64("end", 0, "End timestamp (Unix seconds)")
	historyCmd.Flags().String("symbol", "", "Filter by symbol")
	historyCmd.Flags().Int32("side", 0, "Filter by side: 1=buy, 2=sell")

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a TradFi order",
		RunE:  runTradfiOrderCreate,
	}
	createCmd.Flags().String("symbol", "", "Symbol name, e.g. XAUUSD (required)")
	createCmd.Flags().Int32("side", 0, "Order side: 1=buy, 2=sell (required)")
	createCmd.Flags().String("volume", "", "Order volume (required)")
	createCmd.Flags().String("price", "0", "Order price (use 0 for market orders)")
	createCmd.Flags().String("price-type", "0", "Price type: 0=market, 1=limit")
	createCmd.Flags().String("tp", "", "Take-profit price")
	createCmd.Flags().String("sl", "", "Stop-loss price")
	createCmd.MarkFlagRequired("symbol")
	createCmd.MarkFlagRequired("side")
	createCmd.MarkFlagRequired("volume")

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update a TradFi order (price, TP, SL)",
		RunE:  runTradfiOrderUpdate,
	}
	updateCmd.Flags().Int32("id", 0, "Order ID (required)")
	updateCmd.Flags().String("price", "", "New price (required)")
	updateCmd.Flags().String("tp", "", "New take-profit price")
	updateCmd.Flags().String("sl", "", "New stop-loss price")
	updateCmd.MarkFlagRequired("id")
	updateCmd.MarkFlagRequired("price")

	cancelCmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel a TradFi order",
		RunE:  runTradfiOrderCancel,
	}
	cancelCmd.Flags().Int32("id", 0, "Order ID (required)")
	cancelCmd.MarkFlagRequired("id")

	orderCmd.AddCommand(listCmd, historyCmd, createCmd, updateCmd, cancelCmd)
	Cmd.AddCommand(orderCmd)
}

func runTradfiOrderList(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.TradFiAPI.QueryOrderList(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/tradfi/orders", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	if result.Data.List == nil {
		return p.Table([]string{"ID", "Symbol", "Side", "Volume", "Price", "TP", "SL", "State"}, nil)
	}
	rows := make([][]string, len(result.Data.List))
	for i, o := range result.Data.List {
		rows[i] = []string{
			fmt.Sprintf("%d", o.OrderId), o.Symbol,
			fmt.Sprintf("%d", o.Side), o.Volume,
			o.Price, o.PriceTp, o.PriceSl, o.StateDesc,
		}
	}
	return p.Table([]string{"ID", "Symbol", "Side", "Volume", "Price", "TP", "SL", "State"}, rows)
}

func runTradfiOrderHistory(cmd *cobra.Command, args []string) error {
	begin, _ := cmd.Flags().GetInt64("begin")
	end, _ := cmd.Flags().GetInt64("end")
	symbol, _ := cmd.Flags().GetString("symbol")
	side, _ := cmd.Flags().GetInt32("side")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.QueryOrderHistoryListOpts{}
	if begin != 0 {
		opts.BeginTime = optional.NewInt64(begin)
	}
	if end != 0 {
		opts.EndTime = optional.NewInt64(end)
	}
	if symbol != "" {
		opts.Symbol = optional.NewString(symbol)
	}
	if side != 0 {
		opts.Side = optional.NewInt32(side)
	}

	result, httpResp, err := c.TradFiAPI.QueryOrderHistoryList(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/tradfi/orders/history", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	if result.Data.List == nil {
		return p.Table([]string{"ID", "Symbol", "Side", "Volume", "Price", "TP", "SL", "State"}, nil)
	}
	rows := make([][]string, len(result.Data.List))
	for i, o := range result.Data.List {
		rows[i] = []string{
			fmt.Sprintf("%d", o.OrderId), o.Symbol,
			fmt.Sprintf("%d", o.Side), o.Volume,
			o.Price, o.PriceTp, o.PriceSl, o.StateDesc,
		}
	}
	return p.Table([]string{"ID", "Symbol", "Side", "Volume", "Price", "TP", "SL", "State"}, rows)
}

func runTradfiOrderCreate(cmd *cobra.Command, args []string) error {
	symbol, _ := cmd.Flags().GetString("symbol")
	side, _ := cmd.Flags().GetInt32("side")
	volume, _ := cmd.Flags().GetString("volume")
	price, _ := cmd.Flags().GetString("price")
	priceType, _ := cmd.Flags().GetString("price-type")
	tp, _ := cmd.Flags().GetString("tp")
	sl, _ := cmd.Flags().GetString("sl")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	req := gateapi.TradFiOrderRequest{
		Symbol:    symbol,
		Side:      side,
		Volume:    volume,
		Price:     price,
		PriceType: priceType,
	}
	if tp != "" {
		req.PriceTp = tp
	}
	if sl != "" {
		req.PriceSl = sl
	}

	body, _ := json.Marshal(req)
	result, httpResp, err := c.TradFiAPI.CreateTradFiOrder(c.Context(), req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/tradfi/orders", string(body)))
		return nil
	}
	return p.Print(result)
}

func runTradfiOrderUpdate(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetInt32("id")
	price, _ := cmd.Flags().GetString("price")
	tp, _ := cmd.Flags().GetString("tp")
	sl, _ := cmd.Flags().GetString("sl")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	req := gateapi.TradFiOrderUpdateRequest{Price: price}
	if tp != "" {
		req.PriceTp = &tp
	}
	if sl != "" {
		req.PriceSl = &sl
	}

	body, _ := json.Marshal(req)
	result, httpResp, err := c.TradFiAPI.UpdateOrder(c.Context(), id, req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "PUT", fmt.Sprintf("/tradfi/orders/%d", id), string(body)))
		return nil
	}
	return p.Print(result)
}

func runTradfiOrderCancel(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetInt32("id")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.TradFiAPI.DeleteOrder(c.Context(), id)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "DELETE", fmt.Sprintf("/tradfi/orders/%d", id), ""))
		return nil
	}
	return p.Print(result)
}
