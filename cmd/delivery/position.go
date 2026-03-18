package delivery

import (
	"fmt"
	"strconv"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
)

func init() {
	positionsCmd := &cobra.Command{
		Use:   "positions",
		Short: "List all delivery positions",
		RunE:  runDeliveryPositions,
	}
	positionsCmd.Flags().String("settle", "usdt", "Settlement currency")

	positionCmd := &cobra.Command{
		Use:   "position",
		Short: "Get details of a delivery position",
		RunE:  runDeliveryPosition,
	}
	positionCmd.Flags().String("settle", "usdt", "Settlement currency")
	positionCmd.Flags().String("contract", "", "Futures contract name (required)")
	positionCmd.MarkFlagRequired("contract")

	updateMarginCmd := &cobra.Command{
		Use:   "update-margin",
		Short: "Update delivery position margin",
		RunE:  runDeliveryUpdateMargin,
	}
	updateMarginCmd.Flags().String("settle", "usdt", "Settlement currency")
	updateMarginCmd.Flags().String("contract", "", "Futures contract name (required)")
	updateMarginCmd.Flags().String("change", "", "Margin change amount (required)")
	updateMarginCmd.MarkFlagRequired("contract")
	updateMarginCmd.MarkFlagRequired("change")

	updateLeverageCmd := &cobra.Command{
		Use:   "update-leverage",
		Short: "Update delivery position leverage",
		RunE:  runDeliveryUpdateLeverage,
	}
	updateLeverageCmd.Flags().String("settle", "usdt", "Settlement currency")
	updateLeverageCmd.Flags().String("contract", "", "Futures contract name (required)")
	updateLeverageCmd.Flags().String("leverage", "", "New leverage value (required)")
	updateLeverageCmd.MarkFlagRequired("contract")
	updateLeverageCmd.MarkFlagRequired("leverage")

	updateRiskLimitCmd := &cobra.Command{
		Use:   "update-risk-limit",
		Short: "Update delivery position risk limit",
		RunE:  runDeliveryUpdateRiskLimit,
	}
	updateRiskLimitCmd.Flags().String("settle", "usdt", "Settlement currency")
	updateRiskLimitCmd.Flags().String("contract", "", "Futures contract name (required)")
	updateRiskLimitCmd.Flags().String("risk-limit", "", "New risk limit value (required)")
	updateRiskLimitCmd.MarkFlagRequired("contract")
	updateRiskLimitCmd.MarkFlagRequired("risk-limit")

	positionCloseCmd := &cobra.Command{
		Use:   "position-close",
		Short: "List position close history",
		RunE:  runDeliveryPositionClose,
	}
	positionCloseCmd.Flags().String("settle", "usdt", "Settlement currency")
	positionCloseCmd.Flags().String("contract", "", "Filter by contract name")
	positionCloseCmd.Flags().Int32("limit", 0, "Number of records to return")

	liquidatesCmd := &cobra.Command{
		Use:   "liquidates",
		Short: "List liquidation history",
		RunE:  runDeliveryLiquidates,
	}
	liquidatesCmd.Flags().String("settle", "usdt", "Settlement currency")
	liquidatesCmd.Flags().String("contract", "", "Filter by contract name")
	liquidatesCmd.Flags().Int32("limit", 0, "Number of records to return")

	settlementsCmd := &cobra.Command{
		Use:   "settlements",
		Short: "List settlement records",
		RunE:  runDeliverySettlements,
	}
	settlementsCmd.Flags().String("settle", "usdt", "Settlement currency")
	settlementsCmd.Flags().String("contract", "", "Filter by contract name")
	settlementsCmd.Flags().Int32("limit", 0, "Number of records to return")

	Cmd.AddCommand(positionsCmd, positionCmd, updateMarginCmd, updateLeverageCmd, updateRiskLimitCmd, positionCloseCmd, liquidatesCmd, settlementsCmd)
}

func printDeliveryPosition(p interface{ Table([]string, [][]string) error }, pos gateapi.DeliveryPosition) error {
	return p.Table(
		[]string{"Contract", "Size", "Leverage", "Entry Price", "Mark Price", "Unrealised PnL"},
		[][]string{{pos.Contract, fmt.Sprintf("%d", pos.Size), pos.Leverage, pos.EntryPrice, pos.MarkPrice, pos.UnrealisedPnl}},
	)
}

func runDeliveryPositions(cmd *cobra.Command, args []string) error {
	settle := getSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.DeliveryAPI.ListDeliveryPositions(c.Context(), settle)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/delivery/"+settle+"/positions", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, pos := range result {
		rows[i] = []string{pos.Contract, fmt.Sprintf("%d", pos.Size), pos.Leverage, pos.EntryPrice, pos.MarkPrice, pos.UnrealisedPnl}
	}
	return p.Table([]string{"Contract", "Size", "Leverage", "Entry Price", "Mark Price", "Unrealised PnL"}, rows)
}

func runDeliveryPosition(cmd *cobra.Command, args []string) error {
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

	result, httpResp, err := c.DeliveryAPI.GetDeliveryPosition(c.Context(), settle, contract)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/delivery/"+settle+"/positions/"+contract, ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return printDeliveryPosition(p, result)
}

func runDeliveryUpdateMargin(cmd *cobra.Command, args []string) error {
	settle := getSettle(cmd)
	contract, _ := cmd.Flags().GetString("contract")
	change, _ := cmd.Flags().GetString("change")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.DeliveryAPI.UpdateDeliveryPositionMargin(c.Context(), settle, contract, change)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/delivery/"+settle+"/positions/"+contract+"/margin", ""))
		return nil
	}
	return p.Print(result)
}

func runDeliveryUpdateLeverage(cmd *cobra.Command, args []string) error {
	settle := getSettle(cmd)
	contract, _ := cmd.Flags().GetString("contract")
	leverage, _ := cmd.Flags().GetString("leverage")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.DeliveryAPI.UpdateDeliveryPositionLeverage(c.Context(), settle, contract, leverage)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/delivery/"+settle+"/positions/"+contract+"/leverage", ""))
		return nil
	}
	return p.Print(result)
}

func runDeliveryUpdateRiskLimit(cmd *cobra.Command, args []string) error {
	settle := getSettle(cmd)
	contract, _ := cmd.Flags().GetString("contract")
	riskLimit, _ := cmd.Flags().GetString("risk-limit")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.DeliveryAPI.UpdateDeliveryPositionRiskLimit(c.Context(), settle, contract, riskLimit)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/delivery/"+settle+"/positions/"+contract+"/risk_limit", ""))
		return nil
	}
	return p.Print(result)
}

func runDeliveryPositionClose(cmd *cobra.Command, args []string) error {
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

	opts := &gateapi.ListDeliveryPositionCloseOpts{}
	if contract != "" {
		opts.Contract = optional.NewString(contract)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.DeliveryAPI.ListDeliveryPositionClose(c.Context(), settle, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/delivery/"+settle+"/position_close", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, pc := range result {
		rows[i] = []string{pc.Contract, pc.Side, pc.Pnl, strconv.FormatFloat(pc.Time, 'f', 3, 64)}
	}
	return p.Table([]string{"Contract", "Side", "PnL", "Time"}, rows)
}

func runDeliveryLiquidates(cmd *cobra.Command, args []string) error {
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

	opts := &gateapi.ListDeliveryLiquidatesOpts{}
	if contract != "" {
		opts.Contract = optional.NewString(contract)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.DeliveryAPI.ListDeliveryLiquidates(c.Context(), settle, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/delivery/"+settle+"/liquidates", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, liq := range result {
		rows[i] = []string{fmt.Sprintf("%d", liq.Time), liq.Contract, fmt.Sprintf("%d", liq.Size), liq.LiqPrice, liq.Margin}
	}
	return p.Table([]string{"Time", "Contract", "Size", "Liq Price", "Margin"}, rows)
}

func runDeliverySettlements(cmd *cobra.Command, args []string) error {
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

	opts := &gateapi.ListDeliverySettlementsOpts{}
	if contract != "" {
		opts.Contract = optional.NewString(contract)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.DeliveryAPI.ListDeliverySettlements(c.Context(), settle, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/delivery/"+settle+"/settlements", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, s := range result {
		rows[i] = []string{fmt.Sprintf("%d", s.Time), s.Contract, fmt.Sprintf("%d", s.Size), s.SettlePrice, s.Profit}
	}
	return p.Table([]string{"Time", "Contract", "Size", "Settle Price", "Profit"}, rows)
}
