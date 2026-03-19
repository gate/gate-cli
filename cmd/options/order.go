package options

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
)

var orderCmd = &cobra.Command{
	Use:   "order",
	Short: "Options order commands",
}

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List options orders",
		RunE:  runOptionsOrders,
	}
	listCmd.Flags().String("status", "open", "Order status: open or finished (required)")
	listCmd.Flags().String("contract", "", "Filter by contract name")
	listCmd.Flags().String("underlying", "", "Filter by underlying")
	listCmd.Flags().Int32("limit", 0, "Number of records to return")
	listCmd.Flags().Int32("offset", 0, "Number of records to skip")
	listCmd.MarkFlagRequired("status")

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create an options order",
		RunE:  runOptionsCreateOrder,
	}
	createCmd.Flags().String("contract", "", "Options contract name (required)")
	createCmd.Flags().Int64("size", 0, "Order size, positive for buy, negative for sell (required)")
	createCmd.Flags().String("price", "", "Order price (0 for market order with ioc)")
	createCmd.Flags().String("tif", "", "Time in force: gtc, ioc, poc, fok")
	createCmd.MarkFlagRequired("contract")
	createCmd.MarkFlagRequired("size")

	cancelAllCmd := &cobra.Command{
		Use:   "cancel-all",
		Short: "Cancel all open options orders",
		RunE:  runOptionsCancelOrders,
	}
	cancelAllCmd.Flags().String("contract", "", "Filter by contract name")
	cancelAllCmd.Flags().String("underlying", "", "Filter by underlying")
	cancelAllCmd.Flags().String("side", "", "Filter by side: ask or bid")

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get details of an options order",
		RunE:  runOptionsGetOrder,
	}
	getCmd.Flags().Int64("id", 0, "Order ID (required)")
	getCmd.MarkFlagRequired("id")

	amendCmd := &cobra.Command{
		Use:   "amend",
		Short: "Amend an options order",
		RunE:  runOptionsAmendOrder,
	}
	amendCmd.Flags().Int64("id", 0, "Order ID (required)")
	amendCmd.Flags().String("contract", "", "Options contract name (required)")
	amendCmd.Flags().String("price", "", "New price (required)")
	amendCmd.Flags().Int64("size", 0, "New size")
	amendCmd.MarkFlagRequired("id")
	amendCmd.MarkFlagRequired("contract")
	amendCmd.MarkFlagRequired("price")

	cancelCmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel a single options order",
		RunE:  runOptionsCancelOrder,
	}
	cancelCmd.Flags().Int64("id", 0, "Order ID (required)")
	cancelCmd.MarkFlagRequired("id")

	countdownCmd := &cobra.Command{
		Use:   "countdown-cancel-all",
		Short: "Countdown to cancel all options orders",
		RunE:  runOptionsCountdownCancelAll,
	}
	countdownCmd.Flags().Int32("timeout", 0, "Countdown timeout in seconds (required); 0 to cancel the countdown")
	countdownCmd.Flags().String("underlying", "", "Filter by underlying")
	countdownCmd.MarkFlagRequired("timeout")

	orderCmd.AddCommand(listCmd, createCmd, cancelAllCmd, getCmd, amendCmd, cancelCmd, countdownCmd)
	Cmd.AddCommand(orderCmd)
}

func runOptionsOrders(cmd *cobra.Command, args []string) error {
	status, _ := cmd.Flags().GetString("status")
	contract, _ := cmd.Flags().GetString("contract")
	underlying, _ := cmd.Flags().GetString("underlying")
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

	opts := &gateapi.ListOptionsOrdersOpts{}
	if contract != "" {
		opts.Contract = optional.NewString(contract)
	}
	if underlying != "" {
		opts.Underlying = optional.NewString(underlying)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if offset != 0 {
		opts.Offset = optional.NewInt32(offset)
	}

	result, httpResp, err := c.OptionsAPI.ListOptionsOrders(c.Context(), status, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/options/orders", ""))
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
			fmt.Sprintf("%d", o.Left),
			o.Price,
			o.Status,
			strconv.FormatFloat(o.CreateTime, 'f', 3, 64),
		}
	}
	return p.Table([]string{"ID", "Contract", "Size", "Left", "Price", "Status", "Created"}, rows)
}

func runOptionsCreateOrder(cmd *cobra.Command, args []string) error {
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

	req := gateapi.OptionsOrder{
		Contract: contract,
		Size:     size,
		Price:    price,
		Tif:      tif,
	}
	body, _ := json.Marshal(req)
	result, httpResp, err := c.OptionsAPI.CreateOptionsOrder(c.Context(), req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/options/orders", string(body)))
		return nil
	}
	return p.Print(result)
}

func runOptionsCancelOrders(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	underlying, _ := cmd.Flags().GetString("underlying")
	side, _ := cmd.Flags().GetString("side")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.CancelOptionsOrdersOpts{}
	if contract != "" {
		opts.Contract = optional.NewString(contract)
	}
	if underlying != "" {
		opts.Underlying = optional.NewString(underlying)
	}
	if side != "" {
		opts.Side = optional.NewString(side)
	}

	result, httpResp, err := c.OptionsAPI.CancelOptionsOrders(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "DELETE", "/api/v4/options/orders", ""))
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

func runOptionsGetOrder(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetInt64("id")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.OptionsAPI.GetOptionsOrder(c.Context(), id)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", fmt.Sprintf("/api/v4/options/orders/%d", id), ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"ID", "Contract", "Size", "Left", "Price", "Status", "Created"},
		[][]string{{
			fmt.Sprintf("%d", result.Id),
			result.Contract,
			fmt.Sprintf("%d", result.Size),
			fmt.Sprintf("%d", result.Left),
			result.Price,
			result.Status,
			strconv.FormatFloat(result.CreateTime, 'f', 3, 64),
		}},
	)
}

func runOptionsAmendOrder(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetInt64("id")
	contract, _ := cmd.Flags().GetString("contract")
	price, _ := cmd.Flags().GetString("price")
	size, _ := cmd.Flags().GetInt64("size")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	req := gateapi.InlineObject1{
		Contract: contract,
		Price:    price,
		Size:     size,
	}
	body, _ := json.Marshal(req)
	result, httpResp, err := c.OptionsAPI.AmendOptionsOrder(c.Context(), id, req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "PUT", fmt.Sprintf("/api/v4/options/orders/%d", id), string(body)))
		return nil
	}
	return p.Print(result)
}

func runOptionsCancelOrder(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetInt64("id")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.OptionsAPI.CancelOptionsOrder(c.Context(), id)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "DELETE", fmt.Sprintf("/api/v4/options/orders/%d", id), ""))
		return nil
	}
	return p.Print(result)
}

func runOptionsCountdownCancelAll(cmd *cobra.Command, args []string) error {
	timeout, _ := cmd.Flags().GetInt32("timeout")
	underlying, _ := cmd.Flags().GetString("underlying")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	req := gateapi.CountdownCancelAllOptionsTask{
		Timeout:    timeout,
		Underlying: underlying,
	}
	body, _ := json.Marshal(req)
	result, httpResp, err := c.OptionsAPI.CountdownCancelAllOptions(c.Context(), req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/options/countdown_cancel_all", string(body)))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table([]string{"Trigger Time"}, [][]string{{fmt.Sprintf("%d", result.TriggerTime)}})
}
