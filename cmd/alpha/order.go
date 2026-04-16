package alpha

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
	Short: "Alpha order commands",
}

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List Alpha orders",
		RunE:  runAlphaOrderList,
	}
	listCmd.Flags().String("currency", "", "Filter by currency symbol")
	listCmd.Flags().String("side", "", "Filter by side: buy or sell")
	listCmd.Flags().Int32("status", 0, "Filter by status: 0=all, 1=processing, 2=successful, 3=failed, 4=cancelled")
	listCmd.Flags().Int64("from", 0, "Start timestamp (Unix seconds)")
	listCmd.Flags().Int64("to", 0, "End timestamp (Unix seconds)")
	listCmd.Flags().Int32("limit", 100, "Maximum number of records")
	listCmd.Flags().Int32("page", 1, "Page number")

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get an Alpha order by ID",
		RunE:  runAlphaOrderGet,
	}
	getCmd.Flags().String("id", "", "Order ID (required)")
	getCmd.MarkFlagRequired("id")

	quoteCmd := &cobra.Command{
		Use:   "quote",
		Short: "Get a quote for an Alpha order",
		RunE:  runAlphaOrderQuote,
	}
	quoteCmd.Flags().String("currency", "", "Currency symbol (required)")
	quoteCmd.Flags().String("side", "", "Order side: buy or sell (required)")
	quoteCmd.Flags().String("amount", "", "Trade amount (required)")
	quoteCmd.Flags().String("gas-mode", "speed", "Trading mode: speed (smart) or custom")
	quoteCmd.Flags().String("slippage", "", "Slippage tolerance in percent (e.g. 10 for 10%), required when gas-mode=custom")
	quoteCmd.MarkFlagRequired("currency")
	quoteCmd.MarkFlagRequired("side")
	quoteCmd.MarkFlagRequired("amount")

	placeCmd := &cobra.Command{
		Use:   "place",
		Short: "Place an Alpha order using a quote ID",
		RunE:  runAlphaOrderPlace,
	}
	placeCmd.Flags().String("currency", "", "Currency symbol (required)")
	placeCmd.Flags().String("side", "", "Order side: buy or sell (required)")
	placeCmd.Flags().String("amount", "", "Trade amount (required)")
	placeCmd.Flags().String("gas-mode", "speed", "Trading mode: speed (smart) or custom")
	placeCmd.Flags().String("slippage", "", "Slippage tolerance in percent, required when gas-mode=custom")
	placeCmd.Flags().String("quote-id", "", "Quote ID from the quote command (required)")
	placeCmd.MarkFlagRequired("currency")
	placeCmd.MarkFlagRequired("side")
	placeCmd.MarkFlagRequired("amount")
	placeCmd.MarkFlagRequired("quote-id")

	orderCmd.AddCommand(listCmd, getCmd, quoteCmd, placeCmd)
	Cmd.AddCommand(orderCmd)
}

func runAlphaOrderList(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	side, _ := cmd.Flags().GetString("side")
	status, _ := cmd.Flags().GetInt32("status")
	from, _ := cmd.Flags().GetInt64("from")
	to, _ := cmd.Flags().GetInt64("to")
	limit, _ := cmd.Flags().GetInt32("limit")
	page, _ := cmd.Flags().GetInt32("page")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListAlphaOrderOpts{
		Limit: optional.NewInt32(limit),
		Page:  optional.NewInt32(page),
	}
	if currency != "" {
		opts.Currency = optional.NewString(currency)
	}
	if side != "" {
		opts.Side = optional.NewString(side)
	}
	if status != 0 {
		opts.Status = optional.NewInt32(status)
	}
	if from != 0 {
		opts.From = optional.NewInt64(from)
	}
	if to != 0 {
		opts.To = optional.NewInt64(to)
	}

	result, httpResp, err := c.AlphaAPI.ListAlphaOrder(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/alpha/orders", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, o := range result {
		rows[i] = []string{o.OrderId, o.Currency, o.Side, o.UsdtAmount, o.CurrencyAmount, fmt.Sprintf("%d", o.Status)}
	}
	return p.Table([]string{"ID", "Currency", "Side", "USDT Amount", "Token Amount", "Status"}, rows)
}

func runAlphaOrderGet(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetString("id")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.AlphaAPI.GetAlphaOrder(c.Context(), id)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/alpha/orders/"+id, ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"ID", "Currency", "Side", "USDT Amount", "Token Amount", "Status", "Gas Fee", "Tx Hash"},
		[][]string{{result.OrderId, result.Currency, result.Side, result.UsdtAmount, result.CurrencyAmount, fmt.Sprintf("%d", result.Status), result.GasFee, result.TxHash}},
	)
}

func runAlphaOrderQuote(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	side, _ := cmd.Flags().GetString("side")
	amount, _ := cmd.Flags().GetString("amount")
	gasMode, _ := cmd.Flags().GetString("gas-mode")
	slippage, _ := cmd.Flags().GetString("slippage")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	req := gateapi.QuoteRequest{
		Currency: currency,
		Side:     side,
		Amount:   amount,
		GasMode:  gasMode,
	}
	if slippage != "" {
		req.Slippage = slippage
	}

	body, _ := json.Marshal(req)
	result, httpResp, err := c.AlphaAPI.QuoteAlphaOrder(c.Context(), req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/alpha/orders/quote", string(body)))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"Quote ID", "Price", "Slippage", "Min Received", "Max Received", "Gas Fee (USDT)", "Order Fee"},
		[][]string{{result.QuoteId, result.Price, result.Slippage, result.TargetTokenMinAmount, result.TargetTokenMaxAmount, result.EstimateGasFeeAmountUsdt, result.OrderFee}},
	)
}

func runAlphaOrderPlace(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	side, _ := cmd.Flags().GetString("side")
	amount, _ := cmd.Flags().GetString("amount")
	gasMode, _ := cmd.Flags().GetString("gas-mode")
	slippage, _ := cmd.Flags().GetString("slippage")
	quoteID, _ := cmd.Flags().GetString("quote-id")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	req := gateapi.PlaceOrderRequest{
		Currency: currency,
		Side:     side,
		Amount:   amount,
		GasMode:  gasMode,
		QuoteId:  quoteID,
	}
	if slippage != "" {
		req.Slippage = slippage
	}

	body, _ := json.Marshal(req)
	result, httpResp, err := c.AlphaAPI.PlaceAlphaOrder(c.Context(), req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/alpha/orders", string(body)))
		return nil
	}
	return p.Print(result)
}
