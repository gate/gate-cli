package futures

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
	Short: "Futures market data (public, no authentication required)",
}

// getSettle reads --settle flag, defaulting to "usdt".
func getSettle(cmd *cobra.Command) string {
	s, _ := cmd.Flags().GetString("settle")
	if s == "" {
		return "usdt"
	}
	return s
}

func addSettleFlag(cmd *cobra.Command) {
	cmd.Flags().String("settle", "usdt", "Settlement currency: usdt, btc")
}

func init() {
	tickerCmd := &cobra.Command{
		Use:   "ticker",
		Short: "Get ticker for a futures contract",
		RunE:  runFuturesTicker,
	}
	tickerCmd.Flags().String("contract", "", "Contract name, e.g. BTC_USDT (required)")
	tickerCmd.MarkFlagRequired("contract")
	addSettleFlag(tickerCmd)

	tickersCmd := &cobra.Command{
		Use:   "tickers",
		Short: "Get all futures tickers",
		RunE:  runFuturesTickers,
	}
	addSettleFlag(tickersCmd)

	orderbookCmd := &cobra.Command{
		Use:   "orderbook",
		Short: "Get futures order book",
		RunE:  runFuturesOrderbook,
	}
	orderbookCmd.Flags().String("contract", "", "Contract name (required)")
	orderbookCmd.Flags().Int32("depth", 20, "Order book depth")
	orderbookCmd.MarkFlagRequired("contract")
	addSettleFlag(orderbookCmd)

	tradesCmd := &cobra.Command{
		Use:   "trades",
		Short: "Get recent futures trades",
		RunE:  runFuturesTrades,
	}
	tradesCmd.Flags().String("contract", "", "Contract name (required)")
	tradesCmd.Flags().Int32("limit", 20, "Number of trades to return")
	tradesCmd.MarkFlagRequired("contract")
	addSettleFlag(tradesCmd)

	candlesticksCmd := &cobra.Command{
		Use:   "candlesticks",
		Short: "Get futures candlestick data",
		RunE:  runFuturesCandlesticks,
	}
	candlesticksCmd.Flags().String("contract", "", "Contract name (required)")
	candlesticksCmd.Flags().String("interval", "1h", "Interval: 10s, 1m, 5m, 15m, 30m, 1h, 4h, 8h, 1d, 7d, 30d")
	candlesticksCmd.Flags().Int32("limit", 100, "Number of candlesticks to return")
	candlesticksCmd.MarkFlagRequired("contract")
	addSettleFlag(candlesticksCmd)

	fundingRateCmd := &cobra.Command{
		Use:   "funding-rate",
		Short: "Get historical funding rate",
		RunE:  runFuturesFundingRate,
	}
	fundingRateCmd.Flags().String("contract", "", "Contract name (required)")
	fundingRateCmd.Flags().Int32("limit", 20, "Number of records to return")
	fundingRateCmd.MarkFlagRequired("contract")
	addSettleFlag(fundingRateCmd)

	marketCmd.AddCommand(tickerCmd, tickersCmd, orderbookCmd, tradesCmd, candlesticksCmd, fundingRateCmd)
	Cmd.AddCommand(marketCmd)
}

func runFuturesTicker(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	settle := getSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	tickers, httpResp, err := c.FuturesAPI.ListFuturesTickers(c.Context(), settle, &gateapi.ListFuturesTickersOpts{
		Contract: optional.NewString(contract),
	})
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/tickers", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(tickers)
	}
	rows := make([][]string, len(tickers))
	for i, t := range tickers {
		rows[i] = []string{t.Contract, t.Last, t.ChangePercentage, t.MarkPrice, t.FundingRate, t.High24h, t.Low24h}
	}
	return p.Table([]string{"Contract", "Last", "Change %", "Mark Price", "Funding Rate", "High 24h", "Low 24h"}, rows)
}

func runFuturesTickers(cmd *cobra.Command, args []string) error {
	settle := getSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	tickers, httpResp, err := c.FuturesAPI.ListFuturesTickers(c.Context(), settle, nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/tickers", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(tickers)
	}
	rows := make([][]string, len(tickers))
	for i, t := range tickers {
		rows[i] = []string{t.Contract, t.Last, t.ChangePercentage, t.Volume24hSettle}
	}
	return p.Table([]string{"Contract", "Last", "Change %", "Volume (settle)"}, rows)
}

func runFuturesOrderbook(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	depth, _ := cmd.Flags().GetInt32("depth")
	settle := getSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	ob, httpResp, err := c.FuturesAPI.ListFuturesOrderBook(c.Context(), settle, contract, &gateapi.ListFuturesOrderBookOpts{
		Limit: optional.NewInt32(depth),
	})
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/order_book", ""))
		return nil
	}
	return p.Print(ob)
}

func runFuturesTrades(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	limit, _ := cmd.Flags().GetInt32("limit")
	settle := getSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	trades, httpResp, err := c.FuturesAPI.ListFuturesTrades(c.Context(), settle, contract, &gateapi.ListFuturesTradesOpts{
		Limit: optional.NewInt32(limit),
	})
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/trades", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(trades)
	}
	rows := make([][]string, len(trades))
	for i, t := range trades {
		rows[i] = []string{fmt.Sprintf("%d", t.Id), t.Contract, t.Price, t.Size, fmt.Sprintf("%g", t.CreateTimeMs)}
	}
	return p.Table([]string{"ID", "Contract", "Price", "Size", "Time(ms)"}, rows)
}

func runFuturesCandlesticks(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	interval, _ := cmd.Flags().GetString("interval")
	limit, _ := cmd.Flags().GetInt32("limit")
	settle := getSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	candles, httpResp, err := c.FuturesAPI.ListFuturesCandlesticks(c.Context(), settle, contract, &gateapi.ListFuturesCandlesticksOpts{
		Interval: optional.NewString(interval),
		Limit:    optional.NewInt32(limit),
	})
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/candlesticks", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(candles)
	}
	rows := make([][]string, len(candles))
	for i, c := range candles {
		rows[i] = []string{fmt.Sprintf("%g", c.T), c.O, c.C, c.H, c.L, c.V}
	}
	return p.Table([]string{"Timestamp", "Open", "Close", "High", "Low", "Volume"}, rows)
}

func runFuturesFundingRate(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	limit, _ := cmd.Flags().GetInt32("limit")
	settle := getSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	rates, httpResp, err := c.FuturesAPI.ListFuturesFundingRateHistory(c.Context(), settle, contract, &gateapi.ListFuturesFundingRateHistoryOpts{
		Limit: optional.NewInt32(limit),
	})
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/funding_rate", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(rates)
	}
	rows := make([][]string, len(rates))
	for i, r := range rates {
		rows[i] = []string{fmt.Sprintf("%d", r.T), r.R}
	}
	return p.Table([]string{"Timestamp", "Rate"}, rows)
}
