package futures

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	"github.com/gate/gate-cli/internal/output"
	gateapi "github.com/gate/gateapi-go/v7"
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
	createCmd.Flags().String("tif", "", "Time in force: gtc or ioc; omitted by default (server default); market orders require ioc")
	if err := createCmd.MarkFlagRequired("contract"); err != nil {
		panic(err)
	} // 该flag必须先声明，失败是注册错误。
	if err := createCmd.MarkFlagRequired("trigger-price"); err != nil {
		panic(err)
	} // 该flag必须先声明，失败是注册错误。
	if err := createCmd.MarkFlagRequired("price"); err != nil {
		panic(err)
	} // 该flag必须先声明，失败是注册错误。
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
	getCmd.Flags().Int64("id", 0, "Order ID (required)")
	if err := getCmd.MarkFlagRequired("id"); err != nil {
		panic(err)
	} // 该flag必须先声明，失败是注册错误。
	addSettleFlag(getCmd)

	cancelCmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel a price-triggered order by ID",
		RunE:  runFuturesPriceTriggerCancel,
	}
	cancelCmd.Flags().Int64("id", 0, "Order ID (required)")
	if err := cancelCmd.MarkFlagRequired("id"); err != nil {
		panic(err)
	} // 该flag必须先声明，失败是注册错误。
	addSettleFlag(cancelCmd)

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update a price-triggered order",
		RunE:  runFuturesPriceTriggerUpdate,
	}
	updateCmd.Flags().String("id", "", "Path order ID (required)")
	// SDK契约区分URL路径ID与body中待更新的触发单ID，不能自动互相代填。
	updateCmd.Flags().Int64("body-order-id", 0, "Body trigger order ID (required)")
	if err := updateCmd.MarkFlagRequired("body-order-id"); err != nil {
		panic(err)
	} // 该flag必须先声明，失败是注册错误。
	updateCmd.Flags().Int64("size", 0, "New order size")
	updateCmd.Flags().String("price", "", "New order price")
	updateCmd.Flags().String("trigger-price", "", "New trigger price")
	if err := updateCmd.MarkFlagRequired("id"); err != nil {
		panic(err)
	} // 该flag必须先声明，失败是注册错误。
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
		// 服务端拒绝必须传播为命令失败，避免Agentic鉴权错误被自动化当作执行成功。
		return client.CommandError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/price_orders", "")
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
	// Only explicit new input is validated: omitted TIF must retain the old
	// SDK omitempty wire and must not silently change the user's trade intent.
	tif, _ := cmd.Flags().GetString("tif")
	if cmd.Flags().Changed("tif") && tif != "gtc" && tif != "ioc" {
		return output.InvalidArgsError("--tif must be gtc or ioc when explicitly provided")
	}
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
			Tif:      tif,
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
		return client.CommandError(err, httpResp, "POST", "/api/v4/futures/"+settle+"/price_orders", string(body))
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
		return client.CommandError(err, httpResp, "DELETE", "/api/v4/futures/"+settle+"/price_orders", "")
	}
	return p.Print(result)
}

func runFuturesPriceTriggerGet(cmd *cobra.Command, args []string) error {
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

	result, httpResp, err := c.FuturesAPI.GetPriceTriggeredOrder(c.Context(), settle, id)
	if err != nil {
		return client.CommandError(err, httpResp, "GET", fmt.Sprintf("/api/v4/futures/%s/price_orders/%d", settle, id), "")
	}
	return p.Print(result)
}

func runFuturesPriceTriggerCancel(cmd *cobra.Command, args []string) error {
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

	result, httpResp, err := c.FuturesAPI.CancelPriceTriggeredOrder(c.Context(), settle, id)
	if err != nil {
		return client.CommandError(err, httpResp, "DELETE", fmt.Sprintf("/api/v4/futures/%s/price_orders/%d", settle, id), "")
	}
	return p.Print(result)
}

func runFuturesPriceTriggerUpdate(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetString("id")
	bodyID, _ := cmd.Flags().GetInt64("body-order-id")
	if id == "" || strings.ContainsAny(id, "/%\\?#") || bodyID <= 0 {
		return fmt.Errorf("path id and positive body-order-id are required")
	}
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

	req := gateapi.FuturesUpdatePriceTriggeredOrder{OrderId: bodyID}
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
		return client.CommandError(err, httpResp, "PUT", fmt.Sprintf("/api/v4/futures/%s/price_orders/amend/%s", settle, id), string(body))
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"Order ID"},
		[][]string{{fmt.Sprintf("%d", result.Id)}},
	)
}
