package futures

import (
	"fmt"

	"github.com/antihax/optional"
	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/spf13/cobra"
)

var chaseCmd = &cobra.Command{Use: "chase", Short: "Futures chase orders"}

// SDK-backed commands share the existing transport and authentication boundary.
func init() {
	Cmd.AddCommand(chaseCmd)
	futuresChaseCreateCmd := &cobra.Command{Use: "create", Short: "Create a chase order from SDK request JSON", Args: cobra.NoArgs, RunE: runFuturesChaseCreate}
	futuresChaseCreateCmd.Flags().String("json", "", "CreateChaseOrderReq JSON; see cex-gap-commands.md for all fields")
	cmdutil.MustRequireFlags(futuresChaseCreateCmd, "json")
	addSettleFlag(futuresChaseCreateCmd)
	chaseCmd.AddCommand(futuresChaseCreateCmd)
	futuresChaseGetCmd := &cobra.Command{Use: "get", Short: "Get a chase order", Args: cobra.NoArgs, RunE: runFuturesChaseGet}
	futuresChaseGetCmd.Flags().String("id", "", "Order ID")
	cmdutil.MustRequireFlags(futuresChaseGetCmd, "id")
	addSettleFlag(futuresChaseGetCmd)
	chaseCmd.AddCommand(futuresChaseGetCmd)
	futuresChaseListCmd := &cobra.Command{Use: "list", Short: "List active or finished chase orders", Args: cobra.NoArgs, RunE: runFuturesChaseList}
	futuresChaseListCmd.Flags().String("contract", "", "Contract name, e.g. BTC_USDT")
	futuresChaseListCmd.Flags().Bool("finished", false, "Query finished orders; requires --from and --to")
	futuresChaseListCmd.Flags().Int64("from", 0, "Start Unix timestamp")
	futuresChaseListCmd.Flags().Int64("to", 0, "End Unix timestamp")
	futuresChaseListCmd.Flags().Int32("page", 1, "Page number, starting at 1")
	futuresChaseListCmd.Flags().Int32("limit", 20, "Records per page, 1 to 100")
	futuresChaseListCmd.Flags().Int32("sort-by", 1, "1=created, 2=finished")
	futuresChaseListCmd.Flags().Bool("hide-cancel", false, "Hide cancelled orders")
	futuresChaseListCmd.Flags().Int32("reduce-only", 0, "0=all, 1=reduce-only, 2=not reduce-only")
	futuresChaseListCmd.Flags().Int32("side", 0, "0=all, 1=long, 2=short")
	addSettleFlag(futuresChaseListCmd)
	chaseCmd.AddCommand(futuresChaseListCmd)
	futuresChaseStopCmd := &cobra.Command{Use: "stop", Short: "Stop a chase order by ID or custom text", Args: cobra.NoArgs, RunE: runFuturesChaseStop}
	futuresChaseStopCmd.Flags().String("id", "", "Order ID")
	futuresChaseStopCmd.Flags().String("text", "", "text")
	addSettleFlag(futuresChaseStopCmd)
	chaseCmd.AddCommand(futuresChaseStopCmd)
	futuresChaseStopAllCmd := &cobra.Command{Use: "stop-all", Short: "Stop all chase orders, optionally filtered by contract and margin mode", Args: cobra.NoArgs, RunE: runFuturesChaseStopAll}
	futuresChaseStopAllCmd.Flags().String("contract", "", "Contract name, e.g. BTC_USDT")
	futuresChaseStopAllCmd.Flags().String("pos-margin-mode", "", "isolated or cross")
	addSettleFlag(futuresChaseStopAllCmd)
	chaseCmd.AddCommand(futuresChaseStopAllCmd)
}

func runFuturesChaseCreate(cmd *cobra.Command, args []string) error {
	json, _ := cmd.Flags().GetString("json")
	settle := cmdutil.GetSettle(cmd)
	if err := cmdutil.TradingToken("--settle", settle); err != nil {
		return err
	}
	var req gateapi.CreateChaseOrderReq
	if err := cmdutil.DecodeTradingObject("--json", json, &req); err != nil {
		return err
	}
	if err := validateChaseCreate(settle, req); err != nil {
		return err
	}

	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}
	result, resp, err := c.FuturesAPI.CreateChaseOrder(c.Context(), settle, req)
	if err != nil {
		return client.CommandError(err, resp, "POST", "/api/v4/futures/"+settle+"/autoorder/v1/chase/create", "")
	}
	return cmdutil.GetPrinter(cmd).Print(result)
}

func runFuturesChaseGet(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetString("id")
	settle := cmdutil.GetSettle(cmd)
	if err := cmdutil.TradingToken("--settle", settle); err != nil {
		return err
	}
	if err := cmdutil.TradingToken("--id", id); err != nil {
		return err
	}

	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}
	result, resp, err := c.FuturesAPI.GetChaseOrderDetail(c.Context(), settle, id)
	if err != nil {
		return client.CommandError(err, resp, "GET", "/api/v4/futures/"+settle+"/autoorder/v1/chase/detail", "")
	}
	return cmdutil.GetPrinter(cmd).Print(result)
}

func runFuturesChaseList(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	finished, _ := cmd.Flags().GetBool("finished")
	from, _ := cmd.Flags().GetInt64("from")
	to, _ := cmd.Flags().GetInt64("to")
	page, _ := cmd.Flags().GetInt32("page")
	limit, _ := cmd.Flags().GetInt32("limit")
	sortBy, _ := cmd.Flags().GetInt32("sort-by")
	hideCancel, _ := cmd.Flags().GetBool("hide-cancel")
	reduceOnly, _ := cmd.Flags().GetInt32("reduce-only")
	side, _ := cmd.Flags().GetInt32("side")
	settle := cmdutil.GetSettle(cmd)
	if err := cmdutil.TradingToken("--settle", settle); err != nil {
		return err
	}
	if page < 1 || page > 100000 {
		return fmt.Errorf("--page must be between 1 and 100000")
	}
	if limit < 1 || limit > 100 {
		return fmt.Errorf("--limit must be between 1 and 100")
	}
	if sortBy < 1 || sortBy > 2 {
		return fmt.Errorf("--sort-by must be between 1 and 2")
	}
	if reduceOnly < 0 || reduceOnly > 2 {
		return fmt.Errorf("--reduce-only must be between 0 and 2")
	}
	if side < 0 || side > 2 {
		return fmt.Errorf("--side must be between 0 and 2")
	}
	if from < 0 || to < 0 || (from == 0) != (to == 0) || (from > 0 && to < from) || (finished && (from == 0 || to == 0)) {
		return fmt.Errorf("provide a valid --from/--to range; both are required for --finished")
	}
	// The live service requires an explicit active/finished selector.
	opts := &gateapi.GetChaseOrdersOpts{PageNum: optional.NewInt32(page), PageSize: optional.NewInt32(limit), IsFinished: optional.NewBool(finished)}
	if contract != "" {
		opts.Contract = optional.NewString(contract)
	}

	if from > 0 {
		opts.StartAt = optional.NewInt64(from)
		opts.EndAt = optional.NewInt64(to)
	}
	if cmd.Flags().Changed("hide-cancel") {
		opts.HideCancel = optional.NewBool(hideCancel)
	}
	if cmd.Flags().Changed("reduce-only") {
		opts.ReduceOnly = optional.NewInt32(reduceOnly)
	}
	if cmd.Flags().Changed("side") {
		opts.Side = optional.NewInt32(side)
	}

	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}
	result, resp, err := c.FuturesAPI.GetChaseOrders(c.Context(), settle, sortBy, opts)
	if err != nil {
		return client.CommandError(err, resp, "GET", "/api/v4/futures/"+settle+"/autoorder/v1/chase/list", "")
	}
	if result.Orders == nil {
		result.Orders = []gateapi.ChaseOrder{}
	}
	return cmdutil.GetPrinter(cmd).Print(map[string]any{"orders": result.Orders})
}

func runFuturesChaseStop(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetString("id")
	text, _ := cmd.Flags().GetString("text")
	settle := cmdutil.GetSettle(cmd)
	if err := cmdutil.TradingToken("--settle", settle); err != nil {
		return err
	}
	if (id == "") == (text == "") {
		return fmt.Errorf("provide exactly one of --id or --text")
	}
	if id != "" {
		if err := cmdutil.TradingToken("--id", id); err != nil {
			return err
		}
	}
	if text != "" {
		if err := cmdutil.TradingToken("--text", text); err != nil {
			return err
		}
	}
	req := gateapi.StopChaseOrderReq{Id: id, Text: text}

	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}
	result, resp, err := c.FuturesAPI.StopChaseOrder(c.Context(), settle, req)
	if err != nil {
		return client.CommandError(err, resp, "POST", "/api/v4/futures/"+settle+"/autoorder/v1/chase/stop", "")
	}
	return cmdutil.GetPrinter(cmd).Print(result)
}

func runFuturesChaseStopAll(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	posMarginMode, _ := cmd.Flags().GetString("pos-margin-mode")
	settle := cmdutil.GetSettle(cmd)
	if err := cmdutil.TradingToken("--settle", settle); err != nil {
		return err
	}
	if posMarginMode != "" && posMarginMode != "isolated" && posMarginMode != "cross" {
		return fmt.Errorf("--pos-margin-mode must be isolated or cross")
	}
	req := gateapi.StopAllChaseOrdersReq{Contract: contract, PosMarginMode: posMarginMode}

	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}
	result, resp, err := c.FuturesAPI.StopAllChaseOrders(c.Context(), settle, req)
	if err != nil {
		return client.CommandError(err, resp, "POST", "/api/v4/futures/"+settle+"/autoorder/v1/chase/stop_all", "")
	}
	if result.Orders == nil {
		result.Orders = []gateapi.ChaseOrder{}
	}
	return cmdutil.GetPrinter(cmd).Print(map[string]any{"orders": result.Orders})
}

// Validate execution-affecting fields before authentication or network I/O.
func validateChaseCreate(settle string, req gateapi.CreateChaseOrderReq) error {
	if err := cmdutil.TradingToken("contract", req.Contract); err != nil {
		return err
	}
	if req.Settle != "" && req.Settle != settle {
		return fmt.Errorf("JSON settle must match --settle")
	}
	amount, err := cmdutil.TradingDecimal("amount", req.Amount, false)
	if err != nil {
		return err
	}
	if amount.Sign() == 0 {
		return fmt.Errorf("amount must not be zero")
	}
	price, err := cmdutil.TradingDecimal("price_limit", req.PriceLimit, false)
	if err != nil {
		return err
	}
	if price.Sign() < 0 {
		return fmt.Errorf("price_limit must be non-negative")
	}
	if req.OffsetLimit != "" {
		if _, err := cmdutil.TradingDecimal("offset_limit", req.OffsetLimit, true); err != nil {
			return err
		}
		if price.Sign() != 0 {
			return fmt.Errorf("offset_limit and nonzero price_limit are mutually exclusive")
		}
	}
	if req.PriceType != 0 && req.PriceType != 1 && req.PriceType != 2 {
		return fmt.Errorf("price_type must be 1 or 2")
	}
	if req.PriceType == 2 {
		if req.PriceGapType != 1 && req.PriceGapType != 2 {
			return fmt.Errorf("price_type 2 requires price_gap_type 1 or 2")
		}
		if _, err := cmdutil.TradingDecimal("price_gap_value", req.PriceGapValue, true); err != nil {
			return err
		}
	} else if req.PriceGapType != 0 || req.PriceGapValue != "" {
		return fmt.Errorf("price_gap_type and price_gap_value require price_type 2")
	}
	if req.PosMarginMode != "" && req.PosMarginMode != "isolated" && req.PosMarginMode != "cross" {
		return fmt.Errorf("pos_margin_mode must be isolated or cross")
	}
	if req.PositionMode != "" && req.PositionMode != "single" && req.PositionMode != "dual" && req.PositionMode != "dual_plus" {
		return fmt.Errorf("invalid position_mode")
	}
	return nil
}
