package spot

import (
	"fmt"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
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

	currenciesCmd := &cobra.Command{
		Use:   "currencies",
		Short: "List all currencies (public, no authentication required)",
		RunE:  runSpotCurrencies,
	}

	currencyCmd := &cobra.Command{
		Use:   "currency",
		Short: "Get details for a currency (public, no authentication required)",
		RunE:  runSpotCurrency,
	}
	currencyCmd.Flags().String("currency", "", "Currency symbol, e.g. BTC (required)")
	currencyCmd.MarkFlagRequired("currency")

	pairsCmd := &cobra.Command{
		Use:   "pairs",
		Short: "List all currency pairs (public, no authentication required)",
		RunE:  runSpotPairs,
	}

	pairCmd := &cobra.Command{
		Use:   "pair",
		Short: "Get details for a currency pair (public, no authentication required)",
		RunE:  runSpotPair,
	}
	pairCmd.Flags().String("pair", "", "Currency pair, e.g. BTC_USDT (required)")
	pairCmd.MarkFlagRequired("pair")

	insuranceCmd := &cobra.Command{
		Use:   "insurance",
		Short: "Get spot insurance fund historical data (public, no authentication required)",
		RunE:  runSpotInsurance,
	}
	insuranceCmd.Flags().String("business", "margin", "Leverage business: margin, unified")
	insuranceCmd.Flags().String("currency", "", "Currency name (required)")
	insuranceCmd.Flags().Int64("from", 0, "Start Unix timestamp (seconds)")
	insuranceCmd.Flags().Int64("to", 0, "End Unix timestamp (seconds)")
	insuranceCmd.Flags().Int32("limit", 0, "Number of records to return")
	insuranceCmd.MarkFlagRequired("currency")

	timeCmd := &cobra.Command{
		Use:   "time",
		Short: "Get server time (public, no authentication required)",
		RunE:  runSpotServerTime,
	}

	marketCmd.AddCommand(tickerCmd, tickersCmd, orderbookCmd, tradesCmd, candlesticksCmd,
		currenciesCmd, currencyCmd, pairsCmd, pairCmd, insuranceCmd, timeCmd)
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

func runSpotCurrencies(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.SpotAPI.ListCurrencies(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/spot/currencies", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, cur := range result {
		delisted := "false"
		if cur.Delisted {
			delisted = "true"
		}
		rows[i] = []string{cur.Currency, cur.Name, delisted}
	}
	return p.Table([]string{"Currency", "Name", "Delisted"}, rows)
}

func runSpotCurrency(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.SpotAPI.GetCurrency(c.Context(), currency)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/spot/currencies/"+currency, ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	delisted := "false"
	if result.Delisted {
		delisted = "true"
	}
	return p.Table(
		[]string{"Currency", "Name", "Delisted"},
		[][]string{{result.Currency, result.Name, delisted}},
	)
}

func runSpotPairs(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.SpotAPI.ListCurrencyPairs(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/spot/currency_pairs", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, cp := range result {
		rows[i] = []string{cp.Id, cp.Base, cp.Quote, cp.TradeStatus}
	}
	return p.Table([]string{"ID", "Base", "Quote", "Status"}, rows)
}

func runSpotPair(cmd *cobra.Command, args []string) error {
	pair, _ := cmd.Flags().GetString("pair")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.SpotAPI.GetCurrencyPair(c.Context(), pair)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/spot/currency_pairs/"+pair, ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"ID", "Base", "Quote", "Status"},
		[][]string{{result.Id, result.Base, result.Quote, result.TradeStatus}},
	)
}

func runSpotInsurance(cmd *cobra.Command, args []string) error {
	business, _ := cmd.Flags().GetString("business")
	currency, _ := cmd.Flags().GetString("currency")
	from, _ := cmd.Flags().GetInt64("from")
	to, _ := cmd.Flags().GetInt64("to")
	limit, _ := cmd.Flags().GetInt32("limit")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.GetSpotInsuranceHistoryOpts{}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.SpotAPI.GetSpotInsuranceHistory(c.Context(), business, currency, from, to, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/spot/insurance_history", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{r.Currency, r.Balance, fmt.Sprintf("%d", r.Time)}
	}
	return p.Table([]string{"Currency", "Balance", "Time(ms)"}, rows)
}

func runSpotServerTime(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.SpotAPI.GetSystemTime(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/spot/time", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"Server Time (ms)"},
		[][]string{{fmt.Sprintf("%d", result.ServerTime)}},
	)
}
