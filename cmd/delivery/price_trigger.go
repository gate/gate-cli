package delivery

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
	Short: "Price-triggered delivery order commands",
}

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List price-triggered delivery orders",
		RunE:  runDeliveryPriceTriggerList,
	}
	listCmd.Flags().String("status", "open", "Order status: open or finished (required)")
	listCmd.Flags().String("contract", "", "Filter by contract name")
	listCmd.Flags().Int32("limit", 0, "Number of records to return")
	listCmd.Flags().Int32("offset", 0, "Number of records to skip")
	listCmd.MarkFlagRequired("status")

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a price-triggered delivery order",
		RunE:  runDeliveryPriceTriggerCreate,
	}
	createCmd.Flags().String("contract", "", "Futures contract name (required)")
	createCmd.Flags().String("trigger-price", "", "Trigger price (required)")
	createCmd.Flags().String("order-price", "", "Order price (0 for market)")
	createCmd.Flags().Int64("size", 0, "Order size (required)")
	createCmd.MarkFlagRequired("contract")
	createCmd.MarkFlagRequired("trigger-price")
	createCmd.MarkFlagRequired("size")

	cancelListCmd := &cobra.Command{
		Use:   "cancel-all",
		Short: "Cancel all price-triggered delivery orders for a contract",
		RunE:  runDeliveryPriceTriggerCancelAll,
	}
	cancelListCmd.Flags().String("contract", "", "Futures contract name (required)")
	cancelListCmd.MarkFlagRequired("contract")

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get details of a price-triggered delivery order",
		RunE:  runDeliveryPriceTriggerGet,
	}
	getCmd.Flags().String("id", "", "Order ID (required)")
	getCmd.MarkFlagRequired("id")

	cancelCmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel a single price-triggered delivery order",
		RunE:  runDeliveryPriceTriggerCancel,
	}
	cancelCmd.Flags().String("id", "", "Order ID (required)")
	cancelCmd.MarkFlagRequired("id")

	priceTriggerCmd.AddCommand(listCmd, createCmd, cancelListCmd, getCmd, cancelCmd)
	Cmd.AddCommand(priceTriggerCmd)
}

func runDeliveryPriceTriggerList(cmd *cobra.Command, args []string) error {
	settle := getSettle(cmd)
	status, _ := cmd.Flags().GetString("status")
	contract, _ := cmd.Flags().GetString("contract")
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

	opts := &gateapi.ListPriceTriggeredDeliveryOrdersOpts{}
	if contract != "" {
		opts.Contract = optional.NewString(contract)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if offset != 0 {
		opts.Offset = optional.NewInt32(offset)
	}

	result, httpResp, err := c.DeliveryAPI.ListPriceTriggeredDeliveryOrders(c.Context(), settle, status, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/delivery/"+settle+"/price_orders", ""))
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
			fmt.Sprintf("%d", o.Initial.Size),
			o.Trigger.Price,
			o.Status,
		}
	}
	return p.Table([]string{"ID", "Contract", "Size", "Trigger Price", "Status"}, rows)
}

func runDeliveryPriceTriggerCreate(cmd *cobra.Command, args []string) error {
	settle := getSettle(cmd)
	contract, _ := cmd.Flags().GetString("contract")
	triggerPrice, _ := cmd.Flags().GetString("trigger-price")
	orderPrice, _ := cmd.Flags().GetString("order-price")
	size, _ := cmd.Flags().GetInt64("size")
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
			Price:    orderPrice,
		},
		Trigger: gateapi.FuturesPriceTrigger{
			Price: triggerPrice,
		},
	}
	body, _ := json.Marshal(req)
	result, httpResp, err := c.DeliveryAPI.CreatePriceTriggeredDeliveryOrder(c.Context(), settle, req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/delivery/"+settle+"/price_orders", string(body)))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table([]string{"ID"}, [][]string{{fmt.Sprintf("%d", result.Id)}})
}

func runDeliveryPriceTriggerCancelAll(cmd *cobra.Command, args []string) error {
	settle := getSettle(cmd)
	contract, _ := cmd.Flags().GetString("contract")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.DeliveryAPI.CancelPriceTriggeredDeliveryOrderList(c.Context(), settle, contract)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "DELETE", "/api/v4/delivery/"+settle+"/price_orders", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, o := range result {
		rows[i] = []string{fmt.Sprintf("%d", o.Id), o.Initial.Contract, o.Status}
	}
	return p.Table([]string{"ID", "Contract", "Status"}, rows)
}

func runDeliveryPriceTriggerGet(cmd *cobra.Command, args []string) error {
	settle := getSettle(cmd)
	id, _ := cmd.Flags().GetString("id")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.DeliveryAPI.GetPriceTriggeredDeliveryOrder(c.Context(), settle, id)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/delivery/"+settle+"/price_orders/"+id, ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"ID", "Contract", "Size", "Trigger Price", "Status"},
		[][]string{{
			fmt.Sprintf("%d", result.Id),
			result.Initial.Contract,
			fmt.Sprintf("%d", result.Initial.Size),
			result.Trigger.Price,
			result.Status,
		}},
	)
}

func runDeliveryPriceTriggerCancel(cmd *cobra.Command, args []string) error {
	settle := getSettle(cmd)
	id, _ := cmd.Flags().GetString("id")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.DeliveryAPI.CancelPriceTriggeredDeliveryOrder(c.Context(), settle, id)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "DELETE", "/api/v4/delivery/"+settle+"/price_orders/"+id, ""))
		return nil
	}
	return p.Print(result)
}
