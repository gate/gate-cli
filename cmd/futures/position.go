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

var positionCmd = &cobra.Command{
	Use:   "position",
	Short: "Futures position commands",
}

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all open futures positions",
		RunE:  runFuturesPositionList,
	}
	addSettleFlag(listCmd)

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get position(s) for a contract (works in both single and dual mode)",
		RunE:  runFuturesPositionGet,
	}
	getCmd.Flags().String("contract", "", "Contract name, e.g. BTC_USDT (required)")
	getCmd.MarkFlagRequired("contract")
	addSettleFlag(getCmd)

	listTimerangeCmd := &cobra.Command{
		Use:   "list-timerange",
		Short: "List position history by time range",
		RunE:  runFuturesPositionListTimerange,
	}
	listTimerangeCmd.Flags().String("contract", "", "Contract name (required)")
	listTimerangeCmd.Flags().Int64("from", 0, "Start Unix timestamp")
	listTimerangeCmd.Flags().Int64("to", 0, "End Unix timestamp")
	listTimerangeCmd.Flags().Int32("limit", 0, "Number of records to return")
	listTimerangeCmd.MarkFlagRequired("contract")
	addSettleFlag(listTimerangeCmd)

	leverageCmd := &cobra.Command{
		Use:   "leverage",
		Short: "Get leverage settings for a contract",
		RunE:  runFuturesPositionLeverage,
	}
	leverageCmd.Flags().String("contract", "", "Contract name (required)")
	leverageCmd.MarkFlagRequired("contract")
	addSettleFlag(leverageCmd)

	updateMarginCmd := &cobra.Command{
		Use:   "update-margin",
		Short: "Update position margin",
		RunE:  runFuturesUpdatePositionMargin,
	}
	updateMarginCmd.Flags().String("contract", "", "Contract name (required)")
	updateMarginCmd.Flags().String("change", "", "Margin change amount (required)")
	updateMarginCmd.Flags().String("dual-side", "", "Position side for dual mode: dual_long or dual_short (omit for single mode)")
	updateMarginCmd.MarkFlagRequired("contract")
	updateMarginCmd.MarkFlagRequired("change")
	addSettleFlag(updateMarginCmd)

	updateLeverageCmd := &cobra.Command{
		Use:   "update-leverage",
		Short: "Update position leverage",
		RunE:  runFuturesUpdatePositionLeverage,
	}
	updateLeverageCmd.Flags().String("contract", "", "Contract name (required)")
	updateLeverageCmd.Flags().String("leverage", "", "New leverage (required)")
	updateLeverageCmd.MarkFlagRequired("contract")
	updateLeverageCmd.MarkFlagRequired("leverage")
	addSettleFlag(updateLeverageCmd)

	updateContractLeverageCmd := &cobra.Command{
		Use:   "update-contract-leverage",
		Short: "Update leverage for specified mode",
		RunE:  runFuturesUpdateContractPositionLeverage,
	}
	updateContractLeverageCmd.Flags().String("contract", "", "Contract name (required)")
	updateContractLeverageCmd.Flags().String("leverage", "", "New leverage (required)")
	updateContractLeverageCmd.Flags().String("margin-mode", "", "Margin mode: isolated or cross (required)")
	updateContractLeverageCmd.MarkFlagRequired("contract")
	updateContractLeverageCmd.MarkFlagRequired("leverage")
	updateContractLeverageCmd.MarkFlagRequired("margin-mode")
	addSettleFlag(updateContractLeverageCmd)

	updateCrossCmd := &cobra.Command{
		Use:   "update-cross-mode",
		Short: "Update position cross/isolated margin mode (works in both single and dual mode)",
		RunE:  runFuturesUpdatePositionCrossMode,
	}
	updateCrossCmd.Flags().String("contract", "", "Contract name (required)")
	updateCrossCmd.Flags().String("mode", "", "Margin mode: ISOLATED or CROSS (required)")
	updateCrossCmd.MarkFlagRequired("contract")
	updateCrossCmd.MarkFlagRequired("mode")
	addSettleFlag(updateCrossCmd)

	updateRiskLimitCmd := &cobra.Command{
		Use:   "update-risk-limit",
		Short: "Update position risk limit",
		RunE:  runFuturesUpdatePositionRiskLimit,
	}
	updateRiskLimitCmd.Flags().String("contract", "", "Contract name (required)")
	updateRiskLimitCmd.Flags().String("risk-limit", "", "New risk limit (required)")
	updateRiskLimitCmd.MarkFlagRequired("contract")
	updateRiskLimitCmd.MarkFlagRequired("risk-limit")
	addSettleFlag(updateRiskLimitCmd)

	closeHistoryCmd := &cobra.Command{
		Use:   "close-history",
		Short: "List position close history",
		RunE:  runFuturesPositionCloseHistory,
	}
	closeHistoryCmd.Flags().String("contract", "", "Filter by contract name")
	closeHistoryCmd.Flags().Int32("limit", 0, "Number of records to return")
	closeHistoryCmd.Flags().Int64("from", 0, "Start Unix timestamp")
	closeHistoryCmd.Flags().Int64("to", 0, "End Unix timestamp")
	addSettleFlag(closeHistoryCmd)

	liquidatesCmd := &cobra.Command{
		Use:   "liquidates",
		Short: "List personal liquidation history",
		RunE:  runFuturesPositionLiquidates,
	}
	liquidatesCmd.Flags().String("contract", "", "Filter by contract name")
	liquidatesCmd.Flags().Int32("limit", 0, "Number of records to return")
	addSettleFlag(liquidatesCmd)

	adlCmd := &cobra.Command{
		Use:   "adl",
		Short: "List auto-deleveraging history",
		RunE:  runFuturesPositionADL,
	}
	adlCmd.Flags().String("contract", "", "Filter by contract name")
	adlCmd.Flags().Int32("limit", 0, "Number of records to return")
	addSettleFlag(adlCmd)

	positionCmd.AddCommand(listCmd, getCmd,
		listTimerangeCmd, leverageCmd,
		updateMarginCmd, updateLeverageCmd, updateContractLeverageCmd,
		updateCrossCmd, updateRiskLimitCmd,
		closeHistoryCmd, liquidatesCmd, adlCmd)
	Cmd.AddCommand(positionCmd)
}

func runFuturesPositionList(cmd *cobra.Command, args []string) error {
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	positions, httpResp, err := c.FuturesAPI.ListPositions(c.Context(), settle, nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/positions", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(positions)
	}
	rows := make([][]string, 0, len(positions))
	for _, pos := range positions {
		if pos.Size == "0" || pos.Size == "" {
			continue
		}
		rows = append(rows, []string{
			pos.Contract, pos.Size, pos.EntryPrice,
			pos.MarkPrice, pos.UnrealisedPnl, pos.Leverage,
		})
	}
	return p.Table([]string{"Contract", "Size", "Entry Price", "Mark Price", "Unrealised PNL", "Leverage"}, rows)
}

func runFuturesPositionGet(cmd *cobra.Command, args []string) error {
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

	positions, httpResp, err := c.FuturesAPI.GetDualModePosition(c.Context(), settle, contract)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/dual_comp/positions/"+contract, ""))
		return nil
	}
	if p.IsJSON() {
		if len(positions) == 1 {
			return p.Print(positions[0])
		}
		return p.Print(positions)
	}
	rows := make([][]string, 0, len(positions))
	for _, pos := range positions {
		rows = append(rows, []string{
			pos.Contract, pos.Mode, pos.Size, pos.EntryPrice,
			pos.MarkPrice, pos.UnrealisedPnl, pos.Leverage, pos.LiqPrice,
		})
	}
	return p.Table(
		[]string{"Contract", "Mode", "Size", "Entry Price", "Mark Price", "Unrealised PNL", "Leverage", "Liq Price"},
		rows,
	)
}

func runFuturesPositionListTimerange(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	from, _ := cmd.Flags().GetInt64("from")
	to, _ := cmd.Flags().GetInt64("to")
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

	opts := &gateapi.ListPositionsTimerangeOpts{}
	if from != 0 {
		opts.From = optional.NewInt64(from)
	}
	if to != 0 {
		opts.To = optional.NewInt64(to)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.FuturesAPI.ListPositionsTimerange(c.Context(), settle, contract, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/positions/"+contract+"/timerange", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{r.Contract, r.Size, r.Leverage, r.RiskLimit}
	}
	return p.Table([]string{"Contract", "Size", "Leverage", "Risk Limit"}, rows)
}

func runFuturesPositionLeverage(cmd *cobra.Command, args []string) error {
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

	result, httpResp, err := c.FuturesAPI.GetLeverage(c.Context(), settle, contract, nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/positions/"+contract+"/leverage", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"Leverage"},
		[][]string{{result.Lever}},
	)
}

func runFuturesUpdatePositionMargin(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	change, _ := cmd.Flags().GetString("change")
	dualSide, _ := cmd.Flags().GetString("dual-side")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}
	if dualSide == "" && c.IsDualMode(settle) {
		return fmt.Errorf("--dual-side is required in dual position mode (dual_long or dual_short)")
	}

	result, httpResp, err := c.FuturesAPI.UpdateDualModePositionMargin(c.Context(), settle, contract, change, dualSide)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/futures/"+settle+"/dual_comp/positions/"+contract+"/margin", ""))
		return nil
	}
	return p.Print(result)
}

func runFuturesUpdatePositionLeverage(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	leverage, _ := cmd.Flags().GetString("leverage")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.FuturesAPI.UpdateDualModePositionLeverage(c.Context(), settle, contract, leverage, nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/futures/"+settle+"/dual_comp/positions/"+contract+"/leverage", ""))
		return nil
	}
	return p.Print(result)
}

func runFuturesUpdateContractPositionLeverage(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	leverage, _ := cmd.Flags().GetString("leverage")
	marginMode, _ := cmd.Flags().GetString("margin-mode")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.FuturesAPI.UpdateContractPositionLeverage(c.Context(), settle, contract, leverage, marginMode, nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/futures/"+settle+"/contracts/"+contract+"/leverage", ""))
		return nil
	}
	return p.Print(result)
}

func runFuturesUpdatePositionCrossMode(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	mode, _ := cmd.Flags().GetString("mode")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	req := gateapi.InlineObject{Contract: contract, Mode: mode}
	body, _ := json.Marshal(req)
	result, httpResp, err := c.FuturesAPI.UpdateDualCompPositionCrossMode(c.Context(), settle, req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/futures/"+settle+"/dual_comp/positions/cross_mode", string(body)))
		return nil
	}
	return p.Print(result)
}

func runFuturesUpdatePositionRiskLimit(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	riskLimit, _ := cmd.Flags().GetString("risk-limit")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.FuturesAPI.UpdateDualModePositionRiskLimit(c.Context(), settle, contract, riskLimit)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/futures/"+settle+"/dual_comp/positions/"+contract+"/risk_limit", ""))
		return nil
	}
	return p.Print(result)
}

func runFuturesPositionCloseHistory(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	limit, _ := cmd.Flags().GetInt32("limit")
	from, _ := cmd.Flags().GetInt64("from")
	to, _ := cmd.Flags().GetInt64("to")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListPositionCloseOpts{}
	if contract != "" {
		opts.Contract = optional.NewString(contract)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if from != 0 {
		opts.From = optional.NewInt64(from)
	}
	if to != 0 {
		opts.To = optional.NewInt64(to)
	}

	result, httpResp, err := c.FuturesAPI.ListPositionClose(c.Context(), settle, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/position_close", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{fmt.Sprintf("%g", r.Time), r.Contract, r.Side, r.Pnl}
	}
	return p.Table([]string{"Time", "Contract", "Side", "PNL"}, rows)
}

func runFuturesPositionLiquidates(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
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

	opts := &gateapi.ListLiquidatesOpts{}
	if contract != "" {
		opts.Contract = optional.NewString(contract)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.FuturesAPI.ListLiquidates(c.Context(), settle, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/liquidates", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{fmt.Sprintf("%d", r.Time), r.Contract, r.Size, r.Leverage}
	}
	return p.Table([]string{"Time", "Contract", "Size", "Leverage"}, rows)
}

func runFuturesPositionADL(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
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

	opts := &gateapi.ListAutoDeleveragesOpts{}
	if contract != "" {
		opts.Contract = optional.NewString(contract)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.FuturesAPI.ListAutoDeleverages(c.Context(), settle, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/auto_deleverages", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{fmt.Sprintf("%d", r.Time), r.Contract, r.TradeSize, r.PositionSize}
	}
	return p.Table([]string{"Time", "Contract", "Trade Size", "Position Size"}, rows)
}
