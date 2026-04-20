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

var trailCmd = &cobra.Command{
	Use:   "trail",
	Short: "Futures trailing stop order commands",
}

func init() {
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a trailing stop order",
		RunE:  runFuturesTrailCreate,
	}
	createCmd.Flags().String("contract", "", "Contract name (required)")
	createCmd.Flags().String("amount", "", "Trading quantity in contracts, positive=buy, negative=sell (required)")
	createCmd.Flags().String("activation-price", "0", "Activation price (0 = trigger immediately)")
	createCmd.Flags().String("price-offset", "", "Callback ratio or price distance, e.g. 0.1 or 0.1% (required)")
	createCmd.MarkFlagRequired("contract")
	createCmd.MarkFlagRequired("amount")
	createCmd.MarkFlagRequired("price-offset")
	addSettleFlag(createCmd)

	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop a trailing stop order by ID or text",
		RunE:  runFuturesTrailStop,
	}
	stopCmd.Flags().Int64("id", 0, "Order ID")
	stopCmd.Flags().String("text", "", "Custom text (used if ID not provided)")
	addSettleFlag(stopCmd)

	stopAllCmd := &cobra.Command{
		Use:   "stop-all",
		Short: "Stop all trailing stop orders for a contract",
		RunE:  runFuturesTrailStopAll,
	}
	stopAllCmd.Flags().String("contract", "", "Limit to this contract")
	addSettleFlag(stopAllCmd)

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List trailing stop orders",
		RunE:  runFuturesTrailList,
	}
	listCmd.Flags().String("contract", "", "Filter by contract name")
	listCmd.Flags().Bool("finished", false, "Show finished orders (default: active)")
	listCmd.Flags().Int32("limit", 0, "Number of records per page")
	addSettleFlag(listCmd)

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get a trailing stop order by ID",
		RunE:  runFuturesTrailGet,
	}
	getCmd.Flags().Int64("id", 0, "Order ID (required)")
	getCmd.MarkFlagRequired("id")
	addSettleFlag(getCmd)

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update a trailing stop order",
		RunE:  runFuturesTrailUpdate,
	}
	updateCmd.Flags().Int64("id", 0, "Order ID (required)")
	updateCmd.Flags().String("amount", "", "New amount")
	updateCmd.Flags().String("activation-price", "", "New activation price")
	updateCmd.Flags().String("price-offset", "", "New callback ratio or price distance")
	updateCmd.MarkFlagRequired("id")
	addSettleFlag(updateCmd)

	logCmd := &cobra.Command{
		Use:   "log",
		Short: "Get change log of a trailing stop order",
		RunE:  runFuturesTrailLog,
	}
	logCmd.Flags().Int64("id", 0, "Order ID (required)")
	logCmd.Flags().Int32("limit", 0, "Number of records per page")
	logCmd.MarkFlagRequired("id")
	addSettleFlag(logCmd)

	trailCmd.AddCommand(createCmd, stopCmd, stopAllCmd, listCmd, getCmd, updateCmd, logCmd)
	Cmd.AddCommand(trailCmd)
}

func runFuturesTrailCreate(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	amount, _ := cmd.Flags().GetString("amount")
	activationPrice, _ := cmd.Flags().GetString("activation-price")
	priceOffset, _ := cmd.Flags().GetString("price-offset")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	req := gateapi.CreateTrailOrder{
		Contract:        contract,
		Amount:          amount,
		ActivationPrice: activationPrice,
		PriceOffset:     priceOffset,
	}
	body, _ := json.Marshal(req)
	result, httpResp, err := c.FuturesAPI.CreateTrailOrder(c.Context(), settle, req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/futures/"+settle+"/trail_orders", string(body)))
		return nil
	}
	return p.Print(result)
}

func runFuturesTrailStop(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetInt64("id")
	text, _ := cmd.Flags().GetString("text")
	if id == 0 && text == "" {
		return fmt.Errorf("at least one of --id or --text must be provided")
	}
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	req := gateapi.StopTrailOrder{Id: id, Text: text}
	body, _ := json.Marshal(req)
	result, httpResp, err := c.FuturesAPI.StopTrailOrder(c.Context(), settle, req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "DELETE", "/api/v4/futures/"+settle+"/trail_orders", string(body)))
		return nil
	}
	return p.Print(result)
}

func runFuturesTrailStopAll(cmd *cobra.Command, args []string) error {
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

	req := gateapi.StopAllTrailOrders{Contract: contract}
	body, _ := json.Marshal(req)
	result, httpResp, err := c.FuturesAPI.StopAllTrailOrders(c.Context(), settle, req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "DELETE", "/api/v4/futures/"+settle+"/trail_orders/all", string(body)))
		return nil
	}
	return p.Print(result)
}

func runFuturesTrailList(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	finished, _ := cmd.Flags().GetBool("finished")
	limit, _ := cmd.Flags().GetInt32("limit")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.GetTrailOrdersOpts{}
	if contract != "" {
		opts.Contract = optional.NewString(contract)
	}
	if finished {
		opts.IsFinished = optional.NewBool(true)
	}
	if limit != 0 {
		opts.PageSize = optional.NewInt32(limit)
	}

	result, httpResp, err := c.FuturesAPI.GetTrailOrders(c.Context(), settle, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/trail_orders", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result.Orders))
	for i, o := range result.Orders {
		rows[i] = []string{fmt.Sprintf("%d", o.Id), o.Contract, o.Amount, o.ActivationPrice, o.PriceOffset, o.Status}
	}
	return p.Table([]string{"ID", "Contract", "Amount", "Activation Price", "Offset", "Status"}, rows)
}

func runFuturesTrailGet(cmd *cobra.Command, args []string) error {
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

	result, httpResp, err := c.FuturesAPI.GetTrailOrderDetail(c.Context(), settle, id)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", fmt.Sprintf("/api/v4/futures/%s/trail_orders/%d", settle, id), ""))
		return nil
	}
	return p.Print(result)
}

func runFuturesTrailUpdate(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetInt64("id")
	amount, _ := cmd.Flags().GetString("amount")
	activationPrice, _ := cmd.Flags().GetString("activation-price")
	priceOffset, _ := cmd.Flags().GetString("price-offset")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	req := gateapi.UpdateTrailOrder{Id: id}
	if amount != "" {
		req.Amount = amount
	}
	if activationPrice != "" {
		req.ActivationPrice = activationPrice
	}
	if priceOffset != "" {
		req.PriceOffset = priceOffset
	}
	body, _ := json.Marshal(req)
	result, httpResp, err := c.FuturesAPI.UpdateTrailOrder(c.Context(), settle, req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "PUT", "/api/v4/futures/"+settle+"/trail_orders", string(body)))
		return nil
	}
	return p.Print(result)
}

func runFuturesTrailLog(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetInt64("id")
	limit, _ := cmd.Flags().GetInt32("limit")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.GetTrailOrderChangeLogOpts{}
	if limit != 0 {
		opts.PageSize = optional.NewInt32(limit)
	}

	result, httpResp, err := c.FuturesAPI.GetTrailOrderChangeLog(c.Context(), settle, id, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", fmt.Sprintf("/api/v4/futures/%s/trail_orders/%d/change_log", settle, id), ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result.ChangeLog))
	for i, r := range result.ChangeLog {
		rows[i] = []string{fmt.Sprintf("%d", r.UpdatedAt), r.Amount, r.ActivationPrice, r.PriceOffset}
	}
	return p.Table([]string{"Updated At", "Amount", "Activation Price", "Offset"}, rows)
}
