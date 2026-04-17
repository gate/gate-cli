package tradfi

import (
	"encoding/json"
	"fmt"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

var positionCmd = &cobra.Command{
	Use:   "position",
	Short: "TradFi position commands",
}

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List open TradFi positions",
		RunE:  runTradfiPositionList,
	}

	historyCmd := &cobra.Command{
		Use:   "history",
		Short: "List TradFi position close history",
		RunE:  runTradfiPositionHistory,
	}
	historyCmd.Flags().Int64("begin", 0, "Begin timestamp (Unix seconds)")
	historyCmd.Flags().Int64("end", 0, "End timestamp (Unix seconds)")
	historyCmd.Flags().String("symbol", "", "Filter by symbol")
	historyCmd.Flags().String("direction", "", "Filter by position direction: long, short")

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update take-profit and/or stop-loss for a position",
		RunE:  runTradfiPositionUpdate,
	}
	updateCmd.Flags().Int32("id", 0, "Position ID (required)")
	updateCmd.Flags().String("tp", "", "New take-profit price")
	updateCmd.Flags().String("sl", "", "New stop-loss price")
	updateCmd.MarkFlagRequired("id")

	closeCmd := &cobra.Command{
		Use:   "close",
		Short: "Close a TradFi position",
		RunE:  runTradfiPositionClose,
	}
	closeCmd.Flags().Int32("id", 0, "Position ID (required)")
	closeCmd.Flags().Int32("close-type", 0, "Close type: 0=full close, 1=partial close (required)")
	closeCmd.Flags().String("volume", "", "Volume for partial close (required when close-type=1)")
	closeCmd.MarkFlagRequired("id")
	closeCmd.MarkFlagRequired("close-type")

	positionCmd.AddCommand(listCmd, historyCmd, updateCmd, closeCmd)
	Cmd.AddCommand(positionCmd)
}

func runTradfiPositionList(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.TradFiAPI.QueryPositionList(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/tradfi/positions", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	if result.Data.List == nil {
		return p.Table([]string{"ID", "Symbol", "Direction", "Volume", "Open Price", "Unrealized PNL", "Margin"}, nil)
	}
	rows := make([][]string, len(result.Data.List))
	for i, pos := range result.Data.List {
		rows[i] = []string{
			fmt.Sprintf("%d", pos.PositionId), pos.Symbol,
			pos.PositionDir, pos.Volume, pos.PriceOpen,
			pos.UnrealizedPnl, pos.Margin,
		}
	}
	return p.Table([]string{"ID", "Symbol", "Direction", "Volume", "Open Price", "Unrealized PNL", "Margin"}, rows)
}

func runTradfiPositionHistory(cmd *cobra.Command, args []string) error {
	begin, _ := cmd.Flags().GetInt64("begin")
	end, _ := cmd.Flags().GetInt64("end")
	symbol, _ := cmd.Flags().GetString("symbol")
	direction, _ := cmd.Flags().GetString("direction")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.QueryPositionHistoryListOpts{}
	if begin != 0 {
		opts.BeginTime = optional.NewInt64(begin)
	}
	if end != 0 {
		opts.EndTime = optional.NewInt64(end)
	}
	if symbol != "" {
		opts.Symbol = optional.NewString(symbol)
	}
	if direction != "" {
		opts.PositionDir = optional.NewString(direction)
	}

	result, httpResp, err := c.TradFiAPI.QueryPositionHistoryList(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/tradfi/positions/history", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	if result.Data.List == nil {
		return p.Table([]string{"ID", "Symbol", "Direction", "Volume", "Open Price", "Close Price", "PNL"}, nil)
	}
	rows := make([][]string, len(result.Data.List))
	for i, pos := range result.Data.List {
		rows[i] = []string{
			fmt.Sprintf("%d", pos.PositionId), pos.Symbol,
			pos.PositionDir, pos.Volume, pos.PriceOpen,
			pos.ClosePrice, pos.RealizedPnl,
		}
	}
	return p.Table([]string{"ID", "Symbol", "Direction", "Volume", "Open Price", "Close Price", "Realized PNL"}, rows)
}

func runTradfiPositionUpdate(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetInt32("id")
	tp, _ := cmd.Flags().GetString("tp")
	sl, _ := cmd.Flags().GetString("sl")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	req := gateapi.TradFiPositionUpdateRequest{}
	if tp != "" {
		req.PriceTp = &tp
	}
	if sl != "" {
		req.PriceSl = &sl
	}

	body, _ := json.Marshal(req)
	result, httpResp, err := c.TradFiAPI.UpdatePosition(c.Context(), id, req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "PUT", fmt.Sprintf("/tradfi/positions/%d", id), string(body)))
		return nil
	}
	return p.Print(result)
}

func runTradfiPositionClose(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetInt32("id")
	closeType, _ := cmd.Flags().GetInt32("close-type")
	volume, _ := cmd.Flags().GetString("volume")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	req := gateapi.TradFiClosePositionRequest{CloseType: closeType}
	if volume != "" {
		req.CloseVolume = &volume
	}

	body, _ := json.Marshal(req)
	result, httpResp, err := c.TradFiAPI.ClosePosition(c.Context(), id, req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", fmt.Sprintf("/tradfi/positions/%d/close", id), string(body)))
		return nil
	}
	return p.Print(result)
}
