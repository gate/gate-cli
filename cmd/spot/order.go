package spot

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

	orderCmd.AddCommand(buyCmd, sellCmd, getCmd, listCmd, cancelCmd)
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
