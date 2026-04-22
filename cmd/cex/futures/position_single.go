package futures

import (
	"encoding/json"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

// registerSinglePositionCommands attaches single (one-way) position-mode
// commands. These mirror the dual (hedge) variants and are introduced
// alongside SDK v7.2.71 adoption to expose the bare UpdatePosition* /
// GetPosition methods that were previously unavailable in the CLI.
func registerSinglePositionCommands(parent *cobra.Command) {
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get one-way (single-mode) position for a contract",
		RunE:  runFuturesPositionGetSingle,
	}
	getCmd.Flags().String("contract", "", "Contract name, e.g. BTC_USDT (required)")
	getCmd.MarkFlagRequired("contract")
	addSettleFlag(getCmd)

	updateMarginCmd := &cobra.Command{
		Use:   "update-margin",
		Short: "Update one-way (single-mode) position margin",
		RunE:  runFuturesUpdateSinglePositionMargin,
	}
	updateMarginCmd.Flags().String("contract", "", "Contract name (required)")
	updateMarginCmd.Flags().String("change", "", "Margin change amount (required)")
	updateMarginCmd.MarkFlagRequired("contract")
	updateMarginCmd.MarkFlagRequired("change")
	addSettleFlag(updateMarginCmd)

	updateLeverageCmd := &cobra.Command{
		Use:   "update-leverage",
		Short: "Update one-way (single-mode) position leverage",
		RunE:  runFuturesUpdateSinglePositionLeverage,
	}
	updateLeverageCmd.Flags().String("contract", "", "Contract name (required)")
	updateLeverageCmd.Flags().String("leverage", "", "New leverage (required)")
	updateLeverageCmd.Flags().String("cross-leverage-limit", "", "Cross margin leverage limit (cross mode only)")
	updateLeverageCmd.MarkFlagRequired("contract")
	updateLeverageCmd.MarkFlagRequired("leverage")
	addSettleFlag(updateLeverageCmd)

	updateCrossCmd := &cobra.Command{
		Use:   "update-cross-mode",
		Short: "Update one-way (single-mode) position cross/isolated margin mode",
		RunE:  runFuturesUpdateSinglePositionCrossMode,
	}
	updateCrossCmd.Flags().String("contract", "", "Contract name (required)")
	updateCrossCmd.Flags().String("mode", "", "Margin mode: ISOLATED or CROSS (required)")
	updateCrossCmd.MarkFlagRequired("contract")
	updateCrossCmd.MarkFlagRequired("mode")
	addSettleFlag(updateCrossCmd)

	updateRiskLimitCmd := &cobra.Command{
		Use:   "update-risk-limit",
		Short: "Update one-way (single-mode) position risk limit",
		RunE:  runFuturesUpdateSinglePositionRiskLimit,
	}
	updateRiskLimitCmd.Flags().String("contract", "", "Contract name (required)")
	updateRiskLimitCmd.Flags().String("risk-limit", "", "New risk limit (required)")
	updateRiskLimitCmd.MarkFlagRequired("contract")
	updateRiskLimitCmd.MarkFlagRequired("risk-limit")
	addSettleFlag(updateRiskLimitCmd)

	parent.AddCommand(getCmd, updateMarginCmd, updateLeverageCmd, updateCrossCmd, updateRiskLimitCmd)
}

func runFuturesPositionGetSingle(cmd *cobra.Command, args []string) error {
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

	result, httpResp, err := c.FuturesAPI.GetPosition(c.Context(), settle, contract)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/positions/"+contract, ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"Contract", "Mode", "Size", "Entry Price", "Mark Price", "Unrealised PNL", "Leverage", "Liq Price"},
		[][]string{{result.Contract, result.Mode, result.Size, result.EntryPrice, result.MarkPrice, result.UnrealisedPnl, result.Leverage, result.LiqPrice}},
	)
}

func runFuturesUpdateSinglePositionMargin(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	change, _ := cmd.Flags().GetString("change")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.FuturesAPI.UpdatePositionMargin(c.Context(), settle, contract, change)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/futures/"+settle+"/positions/"+contract+"/margin", ""))
		return nil
	}
	return p.Print(result)
}

func runFuturesUpdateSinglePositionLeverage(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	leverage, _ := cmd.Flags().GetString("leverage")
	crossLeverageLimit, _ := cmd.Flags().GetString("cross-leverage-limit")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var opts *gateapi.UpdatePositionLeverageOpts
	if crossLeverageLimit != "" {
		opts = &gateapi.UpdatePositionLeverageOpts{
			CrossLeverageLimit: optional.NewString(crossLeverageLimit),
		}
	}
	result, httpResp, err := c.FuturesAPI.UpdatePositionLeverage(c.Context(), settle, contract, leverage, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/futures/"+settle+"/positions/"+contract+"/leverage", ""))
		return nil
	}
	return p.Print(result)
}

func runFuturesUpdateSinglePositionCrossMode(cmd *cobra.Command, args []string) error {
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

	req := gateapi.FuturesPositionCrossMode{Contract: contract, Mode: mode}
	body, _ := json.Marshal(req)
	result, httpResp, err := c.FuturesAPI.UpdatePositionCrossMode(c.Context(), settle, req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/futures/"+settle+"/positions/cross_mode", string(body)))
		return nil
	}
	return p.Print(result)
}

func runFuturesUpdateSinglePositionRiskLimit(cmd *cobra.Command, args []string) error {
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

	result, httpResp, err := c.FuturesAPI.UpdatePositionRiskLimit(c.Context(), settle, contract, riskLimit)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/futures/"+settle+"/positions/"+contract+"/risk_limit", ""))
		return nil
	}
	return p.Print(result)
}
