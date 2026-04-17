package options

import (
	"fmt"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

var positionCmd = &cobra.Command{
	Use:   "position",
	Short: "Options position commands",
}

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List options positions",
		RunE:  runOptionsPositions,
	}
	listCmd.Flags().String("underlying", "", "Filter by underlying")

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get details of an options position",
		RunE:  runOptionsPosition,
	}
	getCmd.Flags().String("contract", "", "Options contract name (required)")
	getCmd.MarkFlagRequired("contract")

	closeCmd := &cobra.Command{
		Use:   "close",
		Short: "List position close history for an underlying",
		RunE:  runOptionsPositionClose,
	}
	closeCmd.Flags().String("underlying", "", "Underlying name (required)")
	closeCmd.Flags().String("contract", "", "Filter by contract name")
	closeCmd.MarkFlagRequired("underlying")

	myTradesCmd := &cobra.Command{
		Use:   "my-trades",
		Short: "List personal options trading records",
		RunE:  runOptionsMyTrades,
	}
	myTradesCmd.Flags().String("underlying", "", "Underlying name (required)")
	myTradesCmd.Flags().String("contract", "", "Filter by contract name")
	myTradesCmd.Flags().Int32("limit", 0, "Number of records to return")
	myTradesCmd.Flags().Int32("offset", 0, "Number of records to skip")
	myTradesCmd.MarkFlagRequired("underlying")

	positionCmd.AddCommand(listCmd, getCmd, closeCmd, myTradesCmd)
	Cmd.AddCommand(positionCmd)
}

func runOptionsPositions(cmd *cobra.Command, args []string) error {
	underlying, _ := cmd.Flags().GetString("underlying")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListOptionsPositionsOpts{}
	if underlying != "" {
		opts.Underlying = optional.NewString(underlying)
	}

	result, httpResp, err := c.OptionsAPI.ListOptionsPositions(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/options/positions", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, pos := range result {
		rows[i] = []string{pos.Contract, fmt.Sprintf("%d", pos.Size), pos.EntryPrice, pos.MarkPrice, pos.RealisedPnl, pos.UnrealisedPnl}
	}
	return p.Table([]string{"Contract", "Size", "Entry Price", "Mark Price", "Realised PnL", "Unrealised PnL"}, rows)
}

func runOptionsPosition(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.OptionsAPI.GetOptionsPosition(c.Context(), contract)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/options/positions/"+contract, ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"Contract", "Size", "Entry Price", "Mark Price", "Realised PnL", "Unrealised PnL"},
		[][]string{{result.Contract, fmt.Sprintf("%d", result.Size), result.EntryPrice, result.MarkPrice, result.RealisedPnl, result.UnrealisedPnl}},
	)
}

func runOptionsPositionClose(cmd *cobra.Command, args []string) error {
	underlying, _ := cmd.Flags().GetString("underlying")
	contract, _ := cmd.Flags().GetString("contract")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListOptionsPositionCloseOpts{}
	if contract != "" {
		opts.Contract = optional.NewString(contract)
	}

	result, httpResp, err := c.OptionsAPI.ListOptionsPositionClose(c.Context(), underlying, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/options/position_close", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, pc := range result {
		rows[i] = []string{pc.Contract, pc.Side, pc.Pnl, pc.SettleSize, pc.Text}
	}
	return p.Table([]string{"Contract", "Side", "PnL", "Settle Size", "Text"}, rows)
}

func runOptionsMyTrades(cmd *cobra.Command, args []string) error {
	underlying, _ := cmd.Flags().GetString("underlying")
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

	opts := &gateapi.ListMyOptionsTradesOpts{}
	if contract != "" {
		opts.Contract = optional.NewString(contract)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if offset != 0 {
		opts.Offset = optional.NewInt32(offset)
	}

	result, httpResp, err := c.OptionsAPI.ListMyOptionsTrades(c.Context(), underlying, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/options/my_trades", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, t := range result {
		rows[i] = []string{fmt.Sprintf("%d", t.Id), t.Contract, fmt.Sprintf("%d", t.Size), t.Price, t.Role}
	}
	return p.Table([]string{"ID", "Contract", "Size", "Price", "Role"}, rows)
}
