package spot

import (
	"fmt"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
)

var marketCmd = &cobra.Command{
	Use:   "market",
	Short: "Spot market data (public, no authentication required)",
}

func init() {
	tickerCmd := &cobra.Command{
		Use:   "ticker",
		Short: "Get ticker for a currency pair",
		RunE:  runSpotTicker,
	}
	tickerCmd.Flags().String("pair", "", "Currency pair, e.g. BTC_USDT (required)")
	tickerCmd.MarkFlagRequired("pair")

	tickersCmd := &cobra.Command{
		Use:   "tickers",
		Short: "Get all spot tickers",
		RunE:  runSpotTickers,
	}

	orderbookCmd := &cobra.Command{
		Use:   "orderbook",
		Short: "Get order book for a currency pair",
		RunE:  runSpotOrderbook,
	}
	orderbookCmd.Flags().String("pair", "", "Currency pair (required)")
	orderbookCmd.Flags().Int32("depth", 20, "Order book depth")
	orderbookCmd.MarkFlagRequired("pair")

	tradesCmd := &cobra.Command{
		Use:   "trades",
		Short: "Get recent trades for a currency pair",
		RunE:  runSpotTrades,
	}
	tradesCmd.Flags().String("pair", "", "Currency pair (required)")
	tradesCmd.Flags().Int32("limit", 20, "Number of trades to return")
	tradesCmd.MarkFlagRequired("pair")

	candlesticksCmd := &cobra.Command{
		Use:   "candlesticks",
		Short: "Get candlestick data for a currency pair",
		RunE:  runSpotCandlesticks,
	}
	candlesticksCmd.Flags().String("pair", "", "Currency pair (required)")
	candlesticksCmd.Flags().String("interval", "1h", "Interval: 10s, 1m, 5m, 15m, 30m, 1h, 4h, 8h, 1d, 7d, 30d")
	candlesticksCmd.Flags().Int32("limit", 100, "Number of candlesticks to return")
	candlesticksCmd.MarkFlagRequired("pair")

	marketCmd.AddCommand(tickerCmd, tickersCmd, orderbookCmd, tradesCmd, candlesticksCmd)
	Cmd.AddCommand(marketCmd)
}

func runSpotTicker(cmd *cobra.Command, args []string) error {
	pair, _ := cmd.Flags().GetString("pair")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	tickers, httpResp, err := c.SpotAPI.ListTickers(c.Context(), &gateapi.ListTickersOpts{
		CurrencyPair: optional.NewString(pair),
	})
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/spot/tickers", ""))
		return nil
	}
	if len(tickers) == 0 {
		return fmt.Errorf("no ticker found for %s", pair)
	}
	t := tickers[0]
	if p.IsJSON() {
		return p.Print(t)
	}
	return p.Table(
		[]string{"Pair", "Last", "Change %", "High 24h", "Low 24h", "Volume"},
		[][]string{{t.CurrencyPair, t.Last, t.ChangePercentage, t.High24h, t.Low24h, t.BaseVolume}},
	)
}

func runSpotTickers(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	tickers, httpResp, err := c.SpotAPI.ListTickers(c.Context(), nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/spot/tickers", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(tickers)
	}
	rows := make([][]string, len(tickers))
	for i, t := range tickers {
		rows[i] = []string{t.CurrencyPair, t.Last, t.ChangePercentage, t.BaseVolume}
	}
	return p.Table([]string{"Pair", "Last", "Change %", "Volume"}, rows)
}

func runSpotOrderbook(cmd *cobra.Command, args []string) error {
	pair, _ := cmd.Flags().GetString("pair")
	depth, _ := cmd.Flags().GetInt32("depth")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	ob, httpResp, err := c.SpotAPI.ListOrderBook(c.Context(), pair, &gateapi.ListOrderBookOpts{
		Limit: optional.NewInt32(depth),
	})
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/spot/order_book", ""))
		return nil
	}
	return p.Print(ob)
}

func runSpotTrades(cmd *cobra.Command, args []string) error {
	pair, _ := cmd.Flags().GetString("pair")
	limit, _ := cmd.Flags().GetInt32("limit")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	trades, httpResp, err := c.SpotAPI.ListTrades(c.Context(), pair, &gateapi.ListTradesOpts{
		Limit: optional.NewInt32(limit),
	})
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/spot/trades", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(trades)
	}
	rows := make([][]string, len(trades))
	for i, t := range trades {
		rows[i] = []string{t.Id, t.Side, t.Amount, t.Price, t.CreateTimeMs}
	}
	return p.Table([]string{"ID", "Side", "Amount", "Price", "Time(ms)"}, rows)
}

func runSpotCandlesticks(cmd *cobra.Command, args []string) error {
	pair, _ := cmd.Flags().GetString("pair")
	interval, _ := cmd.Flags().GetString("interval")
	limit, _ := cmd.Flags().GetInt32("limit")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	candles, httpResp, err := c.SpotAPI.ListCandlesticks(c.Context(), pair, &gateapi.ListCandlesticksOpts{
		Interval: optional.NewString(interval),
		Limit:    optional.NewInt32(limit),
	})
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/spot/candlesticks", ""))
		return nil
	}
	// Candlesticks returns [][]string: [timestamp, volume, close, high, low, open, ...]
	if p.IsJSON() {
		return p.Print(candles)
	}
	rows := make([][]string, len(candles))
	for i, c := range candles {
		// Gate candlestick format: [timestamp, volume, close, high, low, open]
		if len(c) >= 6 {
			rows[i] = []string{c[0], c[5], c[2], c[3], c[4], c[1]}
		}
	}
	return p.Table([]string{"Timestamp", "Open", "Close", "High", "Low", "Volume"}, rows)
}
