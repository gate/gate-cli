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

var priceTriggerCmd = &cobra.Command{
	Use:   "price-trigger",
	Short: "Spot price-triggered order commands",
}

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List price-triggered orders",
		RunE:  runSpotPriceTriggerList,
	}
	listCmd.Flags().String("status", "open", "Order status: open, finished")
	listCmd.Flags().String("market", "", "Filter by currency pair")
	listCmd.Flags().Int32("limit", 0, "Number of records to return")
	listCmd.Flags().Int32("offset", 0, "Records to skip")

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a price-triggered order",
		RunE:  runSpotPriceTriggerCreate,
	}
	createCmd.Flags().String("market", "", "Currency pair, e.g. BTC_USDT (required)")
	createCmd.Flags().String("trigger-price", "", "Trigger price (required)")
	createCmd.Flags().String("trigger-rule", ">=", "Trigger condition: >= or <=")
	createCmd.Flags().Int32("trigger-expiration", 86400, "Trigger expiration in seconds")
	createCmd.Flags().String("side", "", "Order side: buy or sell (required)")
	createCmd.Flags().String("price", "", "Order price (required)")
	createCmd.Flags().String("amount", "", "Order amount (required)")
	createCmd.Flags().String("account", "normal", "Account type: normal, margin, unified")
	createCmd.MarkFlagRequired("market")
	createCmd.MarkFlagRequired("trigger-price")
	createCmd.MarkFlagRequired("side")
	createCmd.MarkFlagRequired("price")
	createCmd.MarkFlagRequired("amount")

	cancelAllCmd := &cobra.Command{
		Use:   "cancel-all",
		Short: "Cancel all price-triggered orders",
		RunE:  runSpotPriceTriggerCancelAll,
	}
	cancelAllCmd.Flags().String("market", "", "Limit cancellation to this currency pair")

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get a price-triggered order by ID",
		RunE:  runSpotPriceTriggerGet,
	}
	getCmd.Flags().String("id", "", "Order ID (required)")
	getCmd.MarkFlagRequired("id")

	cancelCmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel a price-triggered order by ID",
		RunE:  runSpotPriceTriggerCancel,
	}
	cancelCmd.Flags().String("id", "", "Order ID (required)")
	cancelCmd.MarkFlagRequired("id")

	priceTriggerCmd.AddCommand(listCmd, createCmd, cancelAllCmd, getCmd, cancelCmd)
	Cmd.AddCommand(priceTriggerCmd)
}

func runSpotPriceTriggerList(cmd *cobra.Command, args []string) error {
	status, _ := cmd.Flags().GetString("status")
	market, _ := cmd.Flags().GetString("market")
	limit, _ := cmd.Flags().GetInt32("limit")
	offset, _ := cmd.Flags().GetInt32("offset")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListSpotPriceTriggeredOrdersOpts{}
	if market != "" {
		opts.Market = optional.NewString(market)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if offset != 0 {
		opts.Offset = optional.NewInt32(offset)
	}

	result, httpResp, err := c.SpotAPI.ListSpotPriceTriggeredOrders(c.Context(), status, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/spot/price_orders", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, o := range result {
		rows[i] = []string{fmt.Sprintf("%d", o.Id), o.Market, o.Trigger.Price, o.Trigger.Rule, o.Put.Side, o.Put.Price, o.Status}
	}
	return p.Table([]string{"ID", "Market", "Trigger Price", "Rule", "Side", "Price", "Status"}, rows)
}

func runSpotPriceTriggerCreate(cmd *cobra.Command, args []string) error {
	market, _ := cmd.Flags().GetString("market")
	triggerPrice, _ := cmd.Flags().GetString("trigger-price")
	triggerRule, _ := cmd.Flags().GetString("trigger-rule")
	triggerExp, _ := cmd.Flags().GetInt32("trigger-expiration")
	side, _ := cmd.Flags().GetString("side")
	price, _ := cmd.Flags().GetString("price")
	amount, _ := cmd.Flags().GetString("amount")
	account, _ := cmd.Flags().GetString("account")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	req := gateapi.SpotPriceTriggeredOrder{
		Market: market,
		Trigger: gateapi.SpotPriceTrigger{
			Price:      triggerPrice,
			Rule:       triggerRule,
			Expiration: triggerExp,
		},
		Put: gateapi.SpotPricePutOrder{
			Side:    side,
			Price:   price,
			Amount:  amount,
			Account: account,
		},
	}
	body, _ := json.Marshal(req)
	result, httpResp, err := c.SpotAPI.CreateSpotPriceTriggeredOrder(c.Context(), req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/spot/price_orders", string(body)))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"Order ID"},
		[][]string{{fmt.Sprintf("%d", result.Id)}},
	)
}

func runSpotPriceTriggerCancelAll(cmd *cobra.Command, args []string) error {
	market, _ := cmd.Flags().GetString("market")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.CancelSpotPriceTriggeredOrderListOpts{}
	if market != "" {
		opts.Market = optional.NewString(market)
	}

	result, httpResp, err := c.SpotAPI.CancelSpotPriceTriggeredOrderList(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "DELETE", "/api/v4/spot/price_orders", ""))
		return nil
	}
	return p.Print(result)
}

func runSpotPriceTriggerGet(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetString("id")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.SpotAPI.GetSpotPriceTriggeredOrder(c.Context(), id)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/spot/price_orders/"+id, ""))
		return nil
	}
	return p.Print(result)
}

func runSpotPriceTriggerCancel(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetString("id")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.SpotAPI.CancelSpotPriceTriggeredOrder(c.Context(), id)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "DELETE", "/api/v4/spot/price_orders/"+id, ""))
		return nil
	}
	return p.Print(result)
}
