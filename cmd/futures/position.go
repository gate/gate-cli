package futures

import (
	"github.com/spf13/cobra"

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
		Short: "Get a specific futures position",
		RunE:  runFuturesPositionGet,
	}
	getCmd.Flags().String("contract", "", "Contract name, e.g. BTC_USDT (required)")
	getCmd.MarkFlagRequired("contract")
	addSettleFlag(getCmd)

	positionCmd.AddCommand(listCmd, getCmd)
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

	// GetFuturesPosition handles both single and dual mode transparently.
	positions, httpResp, err := c.GetFuturesPosition(settle, contract)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/positions/"+contract, ""))
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
