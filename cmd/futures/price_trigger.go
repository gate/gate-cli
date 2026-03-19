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

var priceTriggerCmd = &cobra.Command{
	Use:   "price-trigger",
	Short: "Futures price-triggered order commands",
}

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List price-triggered orders",
		RunE:  runFuturesPriceTriggerList,
	}
	listCmd.Flags().String("status", "open", "Order status: open, finished")
	listCmd.Flags().String("contract", "", "Filter by contract name")
	listCmd.Flags().Int32("limit", 0, "Number of records to return")
	listCmd.Flags().Int32("offset", 0, "Records to skip")
	addSettleFlag(listCmd)

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a price-triggered order",
		RunE:  runFuturesPriceTriggerCreate,
	}
	createCmd.Flags().String("contract", "", "Contract name (required)")
	createCmd.Flags().String("trigger-price", "", "Trigger price (required)")
	createCmd.Flags().Int32("trigger-rule", 1, "Trigger condition: 1=>=, 2=<=")
	createCmd.Flags().Int32("price-type", 0, "Reference price type: 0=latest, 1=mark, 2=index")
	createCmd.Flags().Int64("size", 0, "Order size (0 for full close) (required)")
	createCmd.Flags().String("price", "", "Order price (0 for market) (required)")
	createCmd.MarkFlagRequired("contract")
	createCmd.MarkFlagRequired("trigger-price")
	createCmd.MarkFlagRequired("price")
	addSettleFlag(createCmd)

	cancelAllCmd := &cobra.Command{
		Use:   "cancel-all",
		Short: "Cancel all price-triggered orders",
		RunE:  runFuturesPriceTriggerCancelAll,
	}
	cancelAllCmd.Flags().String("contract", "", "Limit cancellation to this contract")
	addSettleFlag(cancelAllCmd)

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get a price-triggered order by ID",
		RunE:  runFuturesPriceTriggerGet,
	}
	getCmd.Flags().Int32("id", 0, "Order ID (required)")
	getCmd.MarkFlagRequired("id")
	addSettleFlag(getCmd)

	cancelCmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel a price-triggered order by ID",
		RunE:  runFuturesPriceTriggerCancel,
	}
	cancelCmd.Flags().Int32("id", 0, "Order ID (required)")
	cancelCmd.MarkFlagRequired("id")
	addSettleFlag(cancelCmd)

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update a price-triggered order",
		RunE:  runFuturesPriceTriggerUpdate,
	}
	updateCmd.Flags().Int32("id", 0, "Order ID (required)")
	updateCmd.Flags().Int64("size", 0, "New order size")
	updateCmd.Flags().String("price", "", "New order price")
	updateCmd.Flags().String("trigger-price", "", "New trigger price")
	updateCmd.MarkFlagRequired("id")
	addSettleFlag(updateCmd)

	priceTriggerCmd.AddCommand(listCmd, createCmd, cancelAllCmd, getCmd, cancelCmd, updateCmd)
	Cmd.AddCommand(priceTriggerCmd)
}

func runFuturesPriceTriggerList(cmd *cobra.Command, args []string) error {
	status, _ := cmd.Flags().GetString("status")
	contract, _ := cmd.Flags().GetString("contract")
	limit, _ := cmd.Flags().GetInt32("limit")
	offset, _ := cmd.Flags().GetInt32("offset")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListPriceTriggeredOrdersOpts{}
	if contract != "" {
		opts.Contract = optional.NewString(contract)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if offset != 0 {
		opts.Offset = optional.NewInt32(offset)
	}

	result, httpResp, err := c.FuturesAPI.ListPriceTriggeredOrders(c.Context(), settle, status, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/price_orders", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, o := range result {
		rows[i] = []string{
			fmt.Sprintf("%d", o.Id),
			o.Initial.Contract,
			o.Trigger.Price,
			fmt.Sprintf("%d", o.Initial.Size),
			o.Initial.Price,
			o.Status,
		}
	}
	return p.Table([]string{"ID", "Contract", "Trigger Price", "Size", "Order Price", "Status"}, rows)
}

func runFuturesPriceTriggerCreate(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	triggerPrice, _ := cmd.Flags().GetString("trigger-price")
	triggerRule, _ := cmd.Flags().GetInt32("trigger-rule")
	priceType, _ := cmd.Flags().GetInt32("price-type")
	size, _ := cmd.Flags().GetInt64("size")
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

	req := gateapi.FuturesPriceTriggeredOrder{
		Initial: gateapi.FuturesInitialOrder{
			Contract: contract,
			Size:     size,
			Price:    price,
		},
		Trigger: gateapi.FuturesPriceTrigger{
			Price:     triggerPrice,
			Rule:      triggerRule,
			PriceType: priceType,
		},
	}
	body, _ := json.Marshal(req)
	result, httpResp, err := c.FuturesAPI.CreatePriceTriggeredOrder(c.Context(), settle, req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/futures/"+settle+"/price_orders", string(body)))
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

func runFuturesPriceTriggerCancelAll(cmd *cobra.Command, args []string) error {
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

	opts := &gateapi.CancelPriceTriggeredOrderListOpts{}
	if contract != "" {
		opts.Contract = optional.NewString(contract)
	}

	result, httpResp, err := c.FuturesAPI.CancelPriceTriggeredOrderList(c.Context(), settle, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "DELETE", "/api/v4/futures/"+settle+"/price_orders", ""))
		return nil
	}
	return p.Print(result)
}

func runFuturesPriceTriggerGet(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetInt32("id")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.FuturesAPI.GetPriceTriggeredOrder(c.Context(), settle, id)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", fmt.Sprintf("/api/v4/futures/%s/price_orders/%d", settle, id), ""))
		return nil
	}
	return p.Print(result)
}

func runFuturesPriceTriggerCancel(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetInt32("id")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.FuturesAPI.CancelPriceTriggeredOrder(c.Context(), settle, id)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "DELETE", fmt.Sprintf("/api/v4/futures/%s/price_orders/%d", settle, id), ""))
		return nil
	}
	return p.Print(result)
}

func runFuturesPriceTriggerUpdate(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetInt32("id")
	size, _ := cmd.Flags().GetInt64("size")
	price, _ := cmd.Flags().GetString("price")
	triggerPrice, _ := cmd.Flags().GetString("trigger-price")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	req := gateapi.FuturesUpdatePriceTriggeredOrder{}
	if size != 0 {
		req.Size = size
	}
	if price != "" {
		req.Price = price
	}
	if triggerPrice != "" {
		req.TriggerPrice = triggerPrice
	}
	body, _ := json.Marshal(req)
	result, httpResp, err := c.FuturesAPI.UpdatePriceTriggeredOrder(c.Context(), settle, id, req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "PUT", fmt.Sprintf("/api/v4/futures/%s/price_orders/%d", settle, id), string(body)))
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
