package stock

import (
	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/spf13/cobra"
)

func init() {
	group := &cobra.Command{Use: "position", Short: "Stock positions", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() }}
	list := &cobra.Command{Use: "list", Short: "List positions", RunE: runPositions}
	addPnL(list)
	list.Flags().String("symbol", "", "Filter by symbol")
	list.Flags().String("exchange", "", "Exchange: us,hk,kr,jp")
	close := &cobra.Command{Use: "close", Short: "Close a stock position once", RunE: runClosePosition}
	close.Flags().String("symbol", "", "Symbol (required)")
	if err := close.MarkFlagRequired("symbol"); err != nil {
		panic(err)
	} // 该flag必须先声明，失败是注册错误。
	close.Flags().Int32("close-type", 0, "1=partial, 2=all (required)")
	if err := close.MarkFlagRequired("close-type"); err != nil {
		panic(err)
	} // 该flag必须先声明，失败是注册错误。
	close.Flags().String("close-volume", "", "Positive volume for partial close only")
	group.AddCommand(list, close)
	Cmd.AddCommand(group)
}
func runPositions(cmd *cobra.Command, _ []string) error {
	if err := symbol(flagString(cmd, "symbol"), false); err != nil {
		return err
	}
	if err := exchange(cmd); err != nil {
		return err
	}
	cost, price, err := pnl(cmd)
	if err != nil {
		return err
	}
	c, err := sdkClient(cmd, true)
	if err != nil {
		return err
	}
	result, response, err := c.StockAPI.QueryStockPositions(c.Context(), &gateapi.QueryStockPositionsOpts{PnlCalcType: cost, PnlCalcPrice: price, Symbol: optionalString(cmd, "symbol"), Exchange: optionalString(cmd, "exchange")})
	return finish(cmd, result, response, err, "GET", "/stock/positions")
}

// runClosePosition 不使用CFD position_id；全平仓不发送close_volume，部分平仓要求正数。
func runClosePosition(cmd *cobra.Command, _ []string) error {
	value := flagString(cmd, "symbol")
	if err := symbol(value, true); err != nil {
		return err
	}
	kind := flagInt32(cmd, "close-type")
	volume := flagString(cmd, "close-volume")
	switch kind {
	case 1:
		if err := positiveDecimal(volume); err != nil {
			return err
		}
	case 2:
		if cmd.Flags().Changed("close-volume") {
			return invalid("close-volume is only allowed for partial close")
		}
	default:
		return invalid("close-type must be 1 or 2")
	}
	c, err := sdkClient(cmd, true)
	if err != nil {
		return err
	}
	result, response, err := c.StockAPI.CloseStockPosition(c.Context(), gateapi.TradFiSpotClosePositionRequest{Symbol: value, CloseType: kind, CloseVolume: volume})
	return finish(cmd, result, response, err, "POST", "/stock/positions/close")
}
