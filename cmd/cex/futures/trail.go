package futures

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	"github.com/gate/gate-cli/internal/output"
	gateapi "github.com/gate/gateapi-go/v7"
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
	createCmd.Flags().Int32("price-type", 0, "Activation price reference: 1=latest, 2=index, 3=mark (required)")
	createCmd.Flags().Bool("is-gte", false, "Activate when market price >= activation price if true, <= if false (default false)")
	createCmd.MarkFlagRequired("contract")
	createCmd.MarkFlagRequired("amount")
	createCmd.MarkFlagRequired("price-offset")
	cmdutil.MustRequireFlags(createCmd, "price-type")
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
	listCmd.Flags().Int64("start-at", 0, "History start time in Unix seconds; required with --finished and --end-at")
	listCmd.Flags().Int64("end-at", 0, "History end time in Unix seconds; required with --finished and --start-at")
	listCmd.Flags().Int32("limit", 0, "Number of records per page: 1-100 (0 uses service default)")
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
	updateCmd.Flags().Bool("is-gte", false, "Activate at market price >= activation price if true, <= if false; omitted means unchanged")
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
	// The API requires an explicit reference price; zero/missing must not
	// silently change the caller's trading intent or reach the backend.
	priceType, err := cmd.Flags().GetInt32("price-type")
	if err != nil || !cmd.Flags().Changed("price-type") || priceType < 1 || priceType > 3 {
		return output.InvalidArgsError("--price-type is required and must be 1 (latest), 2 (index), or 3 (mark)")
	}
	contract, _ := cmd.Flags().GetString("contract")
	amount, _ := cmd.Flags().GetString("amount")
	activationPrice, _ := cmd.Flags().GetString("activation-price")
	priceOffset, _ := cmd.Flags().GetString("price-offset")
	isGte, err := cmd.Flags().GetBool("is-gte")
	if err != nil {
		return output.InvalidArgsError("--is-gte must be true or false")
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

	req := gateapi.CreateTrailOrder{
		Contract:        contract,
		Amount:          amount,
		ActivationPrice: activationPrice,
		PriceOffset:     priceOffset,
		PriceType:       priceType,
		IsGte:           isGte,
	}
	body, _ := json.Marshal(req)
	result, httpResp, err := c.FuturesAPI.CreateTrailOrder(c.Context(), settle, req)
	if err != nil {
		// 服务端拒绝必须传播为命令失败，避免Agentic鉴权错误被自动化当作执行成功。
		return client.TrailCommandError(err, httpResp, "createTrailOrder", settle, string(body))
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
		return client.TrailCommandError(err, httpResp, "stopTrailOrder", settle, string(body))
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
		return client.TrailCommandError(err, httpResp, "stopAllTrailOrders", settle, string(body))
	}
	return p.Print(result)
}

func runFuturesTrailList(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	finished, _ := cmd.Flags().GetBool("finished")
	limit, _ := cmd.Flags().GetInt32("limit")
	startAt, startErr := cmd.Flags().GetInt64("start-at")
	endAt, endErr := cmd.Flags().GetInt64("end-at")
	startSet, endSet := cmd.Flags().Changed("start-at"), cmd.Flags().Changed("end-at")
	// Keep time validation before client/auth creation: malformed input must
	// not refresh credentials or reach the backend. Calendar bounds reject
	// modern millisecond timestamps without guessing or converting units.
	const maxUnixSecond = int64(253402300799) // 9999-12-31T23:59:59Z, the RFC3339 calendar boundary.
	if startErr != nil || endErr != nil || (finished && (!startSet || !endSet || startAt <= 0 || endAt <= startAt || endAt > maxUnixSecond)) {
		return output.InvalidArgsError("--finished requires --start-at and --end-at in Unix seconds (years 1970-9999), with 0 < start-at < end-at; do not use milliseconds")
	}
	if !finished && (startSet || endSet) {
		return output.InvalidArgsError("--start-at and --end-at are only supported with --finished")
	}
	if limit < 0 || limit > 100 {
		return output.InvalidArgsError("--limit must be 0 (service default) or between 1 and 100")
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

	opts := &gateapi.GetTrailOrdersOpts{}
	if contract != "" {
		opts.Contract = optional.NewString(contract)
	}
	if finished {
		opts.IsFinished = optional.NewBool(true)
		opts.StartAt = optional.NewInt64(startAt)
		opts.EndAt = optional.NewInt64(endAt)
	}
	if limit != 0 {
		opts.PageSize = optional.NewInt32(limit)
	}

	result, httpResp, err := c.FuturesAPI.GetTrailOrders(c.Context(), settle, opts)
	if err != nil {
		return client.TrailCommandError(err, httpResp, "getTrailOrders", settle, "")
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	// The SDK owns the envelope and string IDs. Table output must preserve the
	// exact identifier, including values above JavaScript's safe integer range.
	orders := result.Data.Orders
	rows := make([][]string, len(orders))
	for i, o := range orders {
		rows[i] = []string{o.Id, o.Contract, o.Amount, o.ActivationPrice, o.PriceOffset, o.Status}
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
		return client.TrailCommandError(err, httpResp, "getTrailOrderDetail", settle, "")
	}
	return p.Print(result)
}

func runFuturesTrailUpdate(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetInt64("id")
	amount, _ := cmd.Flags().GetString("amount")
	activationPrice, _ := cmd.Flags().GetString("activation-price")
	priceOffset, _ := cmd.Flags().GetString("price-offset")
	isGte, err := cmd.Flags().GetBool("is-gte")
	if err != nil {
		return output.InvalidArgsError("--is-gte must be true or false")
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

	req := gateapi.UpdateTrailOrder{Id: id}
	// The SDK uses a string tri-state: empty preserves direction, whereas
	// "false" explicitly changes it. Never use the bool value as presence.
	if cmd.Flags().Changed("is-gte") {
		req.IsGteStr = strconv.FormatBool(isGte)
	}
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
		return client.TrailCommandError(err, httpResp, "updateTrailOrder", settle, string(body))
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

	// The API documents page 1 as the default, but the service requires it on
	// the wire. Keep pagination in the SDK instead of reconstructing responses.
	opts := &gateapi.GetTrailOrderChangeLogOpts{PageNum: optional.NewInt32(1)}
	if limit != 0 {
		opts.PageSize = optional.NewInt32(limit)
	}

	result, httpResp, err := c.FuturesAPI.GetTrailOrderChangeLog(c.Context(), settle, id, opts)
	if err != nil {
		return client.TrailCommandError(err, httpResp, "getTrailOrderChangeLog", settle, "")
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	// SDK v178 owns the envelope and the quoted timestamp representation.
	rows := make([][]string, len(result.Data.ChangeLog))
	for i, r := range result.Data.ChangeLog {
		rows[i] = []string{r.UpdatedAt, r.Amount, r.ActivationPrice, r.PriceOffset}
	}
	return p.Table([]string{"Updated At", "Amount", "Activation Price", "Offset"}, rows)
}
