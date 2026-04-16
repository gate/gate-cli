package delivery

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

var orderCmd = &cobra.Command{
	Use:   "order",
	Short: "Delivery futures order commands",
}

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List delivery futures orders",
		RunE:  runDeliveryOrders,
	}
	listCmd.Flags().String("status", "open", "Order status: open or finished (required)")
	listCmd.Flags().String("contract", "", "Filter by contract name")
	listCmd.Flags().Int32("limit", 0, "Number of records to return")
	listCmd.Flags().Int32("offset", 0, "Number of records to skip")
	listCmd.MarkFlagRequired("status")

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a delivery futures order",
		RunE:  runDeliveryCreateOrder,
	}
	createCmd.Flags().String("contract", "", "Futures contract name (required)")
	createCmd.Flags().Int64("size", 0, "Order size, positive for buy, negative for sell (required)")
	createCmd.Flags().String("price", "", "Order price (0 for market order)")
	createCmd.Flags().String("tif", "", "Time in force: gtc, ioc, poc, fok")
	createCmd.MarkFlagRequired("contract")
	createCmd.MarkFlagRequired("size")

	cancelAllCmd := &cobra.Command{
		Use:   "cancel-all",
		Short: "Cancel all open delivery orders for a contract",
		RunE:  runDeliveryCancelOrders,
	}
	cancelAllCmd.Flags().String("contract", "", "Futures contract name (required)")
	cancelAllCmd.Flags().String("side", "", "Filter by side: ask or bid")
	cancelAllCmd.MarkFlagRequired("contract")

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get details of a delivery futures order",
		RunE:  runDeliveryGetOrder,
	}
	getCmd.Flags().String("id", "", "Order ID (required)")
	getCmd.MarkFlagRequired("id")

	cancelCmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel a single delivery futures order",
		RunE:  runDeliveryCancelOrder,
	}
	cancelCmd.Flags().String("id", "", "Order ID (required)")
	cancelCmd.MarkFlagRequired("id")

	myTradesCmd := &cobra.Command{
		Use:   "my-trades",
		Short: "List personal delivery futures trading records",
		RunE:  runDeliveryMyTrades,
	}
	myTradesCmd.Flags().String("contract", "", "Filter by contract name")
	myTradesCmd.Flags().Int32("limit", 0, "Number of records to return")

	orderCmd.AddCommand(listCmd, createCmd, cancelAllCmd, getCmd, cancelCmd, myTradesCmd)
	Cmd.AddCommand(orderCmd)
}

func runDeliveryOrders(cmd *cobra.Command, args []string) error {
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

	opts := &gateapi.ListDeliveryOrdersOpts{}
	if contract != "" {
		opts.Contract = optional.NewString(contract)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if offset != 0 {
		opts.Offset = optional.NewInt32(offset)
	}

	result, httpResp, err := c.DeliveryAPI.ListDeliveryOrders(c.Context(), settle, status, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/delivery/"+settle+"/orders", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, o := range result {
		rows[i] = []string{
			fmt.Sprintf("%d", o.Id),
			o.Contract,
			fmt.Sprintf("%d", o.Size),
			o.Price,
			o.Status,
			strconv.FormatFloat(o.CreateTime, 'f', 3, 64),
		}
	}
	return p.Table([]string{"ID", "Contract", "Size", "Price", "Status", "Created"}, rows)
}

func runDeliveryCreateOrder(cmd *cobra.Command, args []string) error {
	settle := getSettle(cmd)
	contract, _ := cmd.Flags().GetString("contract")
	size, _ := cmd.Flags().GetInt64("size")
	price, _ := cmd.Flags().GetString("price")
	tif, _ := cmd.Flags().GetString("tif")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	req := gateapi.DeliveryOrder{
		Contract: contract,
		Size:     size,
		Price:    price,
		Tif:      tif,
	}
	body, _ := json.Marshal(req)
	result, httpResp, err := c.DeliveryAPI.CreateDeliveryOrder(c.Context(), settle, req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/delivery/"+settle+"/orders", string(body)))
		return nil
	}
	return p.Print(result)
}

func runDeliveryCancelOrders(cmd *cobra.Command, args []string) error {
	settle := getSettle(cmd)
	contract, _ := cmd.Flags().GetString("contract")
	side, _ := cmd.Flags().GetString("side")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.CancelDeliveryOrdersOpts{}
	if side != "" {
		opts.Side = optional.NewString(side)
	}

	result, httpResp, err := c.DeliveryAPI.CancelDeliveryOrders(c.Context(), settle, contract, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "DELETE", "/api/v4/delivery/"+settle+"/orders", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, o := range result {
		rows[i] = []string{fmt.Sprintf("%d", o.Id), o.Contract, o.Status, o.FinishAs}
	}
	return p.Table([]string{"ID", "Contract", "Status", "Finish As"}, rows)
}

func runDeliveryGetOrder(cmd *cobra.Command, args []string) error {
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

	result, httpResp, err := c.DeliveryAPI.GetDeliveryOrder(c.Context(), settle, id)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/delivery/"+settle+"/orders/"+id, ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"ID", "Contract", "Size", "Price", "Status", "Created"},
		[][]string{{
			fmt.Sprintf("%d", result.Id),
			result.Contract,
			fmt.Sprintf("%d", result.Size),
			result.Price,
			result.Status,
			strconv.FormatFloat(result.CreateTime, 'f', 3, 64),
		}},
	)
}

func runDeliveryCancelOrder(cmd *cobra.Command, args []string) error {
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

	result, httpResp, err := c.DeliveryAPI.CancelDeliveryOrder(c.Context(), settle, id)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "DELETE", "/api/v4/delivery/"+settle+"/orders/"+id, ""))
		return nil
	}
	return p.Print(result)
}

func runDeliveryMyTrades(cmd *cobra.Command, args []string) error {
	settle := getSettle(cmd)
	contract, _ := cmd.Flags().GetString("contract")
	limit, _ := cmd.Flags().GetInt32("limit")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.GetMyDeliveryTradesOpts{}
	if contract != "" {
		opts.Contract = optional.NewString(contract)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.DeliveryAPI.GetMyDeliveryTrades(c.Context(), settle, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/delivery/"+settle+"/my_trades", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, t := range result {
		rows[i] = []string{
			fmt.Sprintf("%d", t.Id),
			t.Contract,
			t.OrderId,
			fmt.Sprintf("%d", t.Size),
			t.Price,
			t.Role,
			strconv.FormatFloat(t.CreateTime, 'f', 3, 64),
		}
	}
	return p.Table([]string{"ID", "Contract", "Order ID", "Size", "Price", "Role", "Time"}, rows)
}
