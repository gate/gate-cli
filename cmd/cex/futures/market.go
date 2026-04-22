package futures

import (
	"encoding/json"
	"fmt"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

var marketCmd = &cobra.Command{
	Use:   "market",
	Short: "Futures market data (public, no authentication required)",
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

	contractsCmd := &cobra.Command{
		Use:   "contracts",
		Short: "List all futures contracts (public, no authentication required)",
		RunE:  runFuturesContracts,
	}
	contractsCmd.Flags().Int32("limit", 0, "Number of records to return")
	contractsCmd.Flags().Int32("offset", 0, "Records to skip")
	addSettleFlag(contractsCmd)

	contractCmd := &cobra.Command{
		Use:   "contract",
		Short: "Get details for a futures contract (public, no authentication required)",
		RunE:  runFuturesContract,
	}
	contractCmd.Flags().String("contract", "", "Contract name, e.g. BTC_USDT (required)")
	contractCmd.MarkFlagRequired("contract")
	addSettleFlag(contractCmd)

	premiumCmd := &cobra.Command{
		Use:   "premium",
		Short: "Get premium index K-line data (public, no authentication required)",
		RunE:  runFuturesPremium,
	}
	premiumCmd.Flags().String("contract", "", "Contract name (required)")
	premiumCmd.Flags().String("interval", "1h", "Interval: 10s, 1m, 5m, 15m, 30m, 1h, 4h, 8h, 1d, 7d, 30d")
	premiumCmd.Flags().Int32("limit", 100, "Number of records to return")
	premiumCmd.MarkFlagRequired("contract")
	addSettleFlag(premiumCmd)

	statsCmd := &cobra.Command{
		Use:   "stats",
		Short: "Get futures contract statistics (public, no authentication required)",
		RunE:  runFuturesStats,
	}
	statsCmd.Flags().String("contract", "", "Contract name (required)")
	statsCmd.Flags().String("interval", "1h", "Stat interval: 5m, 15m, 30m, 1h, 4h, 1d")
	statsCmd.Flags().Int32("limit", 30, "Number of records to return")
	statsCmd.MarkFlagRequired("contract")
	addSettleFlag(statsCmd)

	indexCmd := &cobra.Command{
		Use:   "index-constituents",
		Short: "Get index constituents (public, no authentication required)",
		RunE:  runFuturesIndexConstituents,
	}
	indexCmd.Flags().String("index", "", "Index name (required)")
	indexCmd.MarkFlagRequired("index")
	addSettleFlag(indexCmd)

	riskLimitTiersCmd := &cobra.Command{
		Use:   "risk-limit-tiers",
		Short: "Query risk limit tiers (public, no authentication required)",
		RunE:  runFuturesRiskLimitTiers,
	}
	riskLimitTiersCmd.Flags().String("contract", "", "Filter by contract name")
	riskLimitTiersCmd.Flags().Int32("limit", 0, "Number of records to return")
	addSettleFlag(riskLimitTiersCmd)

	riskLimitTableCmd := &cobra.Command{
		Use:   "risk-limit-table <table-id>",
		Short: "Query a specific risk limit table (public, no authentication required)",
		Args:  cobra.ExactArgs(1),
		RunE:  runFuturesRiskLimitTable,
	}
	addSettleFlag(riskLimitTableCmd)

	liquidationsCmd := &cobra.Command{
		Use:   "liquidations",
		Short: "List recently liquidated orders (public, no authentication required)",
		RunE:  runFuturesLiquidations,
	}
	liquidationsCmd.Flags().String("contract", "", "Filter by contract name")
	liquidationsCmd.Flags().Int32("limit", 0, "Number of records to return")
	addSettleFlag(liquidationsCmd)

	insuranceCmd := &cobra.Command{
		Use:   "insurance",
		Short: "Get futures insurance fund history (public, no authentication required)",
		RunE:  runFuturesInsurance,
	}
	insuranceCmd.Flags().Int32("limit", 0, "Number of records to return")
	addSettleFlag(insuranceCmd)

	batchFundingRatesCmd := &cobra.Command{
		Use:   "batch-funding-rates",
		Short: "Get batch funding rates for multiple contracts (public, no authentication required)",
		RunE:  runFuturesBatchFundingRates,
	}
	batchFundingRatesCmd.Flags().StringSlice("contracts", nil, "Contract names (required)")
	batchFundingRatesCmd.MarkFlagRequired("contracts")
	addSettleFlag(batchFundingRatesCmd)

	marketCmd.AddCommand(tickerCmd, tickersCmd, orderbookCmd, tradesCmd, candlesticksCmd, fundingRateCmd,
		contractsCmd, contractCmd, premiumCmd, statsCmd, indexCmd, riskLimitTiersCmd, riskLimitTableCmd,
		liquidationsCmd, insuranceCmd, batchFundingRatesCmd)
	Cmd.AddCommand(marketCmd)
}

func runFuturesTicker(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	settle := cmdutil.GetSettle(cmd)
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
	settle := cmdutil.GetSettle(cmd)
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
	settle := cmdutil.GetSettle(cmd)
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
	settle := cmdutil.GetSettle(cmd)
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
	settle := cmdutil.GetSettle(cmd)
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
	settle := cmdutil.GetSettle(cmd)
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

func runFuturesContracts(cmd *cobra.Command, args []string) error {
	settle := cmdutil.GetSettle(cmd)
	limit, _ := cmd.Flags().GetInt32("limit")
	offset, _ := cmd.Flags().GetInt32("offset")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.ListFuturesContractsOpts{}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if offset != 0 {
		opts.Offset = optional.NewInt32(offset)
	}

	result, httpResp, err := c.FuturesAPI.ListFuturesContracts(c.Context(), settle, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/contracts", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, ct := range result {
		rows[i] = []string{ct.Name, ct.Type, ct.LeverageMin, ct.LeverageMax, ct.FundingRate, ct.MarkPrice}
	}
	return p.Table([]string{"Name", "Type", "Lev Min", "Lev Max", "Funding Rate", "Mark Price"}, rows)
}

func runFuturesContract(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.FuturesAPI.GetFuturesContract(c.Context(), settle, contract)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/contracts/"+contract, ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"Name", "Type", "Lev Min", "Lev Max", "Funding Rate", "Mark Price"},
		[][]string{{result.Name, result.Type, result.LeverageMin, result.LeverageMax, result.FundingRate, result.MarkPrice}},
	)
}

func runFuturesPremium(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	interval, _ := cmd.Flags().GetString("interval")
	limit, _ := cmd.Flags().GetInt32("limit")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.ListFuturesPremiumIndexOpts{
		Interval: optional.NewString(interval),
		Limit:    optional.NewInt32(limit),
	}
	result, httpResp, err := c.FuturesAPI.ListFuturesPremiumIndex(c.Context(), settle, contract, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/premium_index", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{fmt.Sprintf("%g", r.T), r.O, r.C, r.H, r.L}
	}
	return p.Table([]string{"Timestamp", "Open", "Close", "High", "Low"}, rows)
}

func runFuturesStats(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	interval, _ := cmd.Flags().GetString("interval")
	limit, _ := cmd.Flags().GetInt32("limit")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.ListContractStatsOpts{
		Interval: optional.NewString(interval),
		Limit:    optional.NewInt32(limit),
	}
	result, httpResp, err := c.FuturesAPI.ListContractStats(c.Context(), settle, contract, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/contract_stats", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{
			fmt.Sprintf("%d", r.Time),
			fmt.Sprintf("%g", r.LsrTaker),
			fmt.Sprintf("%g", r.LsrAccount),
			r.LongLiqSize,
		}
	}
	return p.Table([]string{"Time", "LSR Taker", "LSR Account", "Long Liq Size"}, rows)
}

func runFuturesIndexConstituents(cmd *cobra.Command, args []string) error {
	index, _ := cmd.Flags().GetString("index")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.FuturesAPI.GetIndexConstituents(c.Context(), settle, index)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/index_constituents/"+index, ""))
		return nil
	}
	return p.Print(result)
}

func runFuturesRiskLimitTiers(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	limit, _ := cmd.Flags().GetInt32("limit")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.ListFuturesRiskLimitTiersOpts{}
	if contract != "" {
		opts.Contract = optional.NewString(contract)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.FuturesAPI.ListFuturesRiskLimitTiers(c.Context(), settle, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/risk_limit_tiers", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{fmt.Sprintf("%d", r.Tier), r.RiskLimit, r.InitialRate, r.MaintenanceRate}
	}
	return p.Table([]string{"Tier", "Risk Limit", "Initial Rate", "Maintenance Rate"}, rows)
}

func runFuturesRiskLimitTable(cmd *cobra.Command, args []string) error {
	tableID := args[0]
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.FuturesAPI.GetFuturesRiskLimitTable(c.Context(), settle, tableID)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/risk_limit_table/"+tableID, ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{fmt.Sprintf("%d", r.Tier), r.RiskLimit, r.InitialRate, r.MaintenanceRate}
	}
	return p.Table([]string{"Tier", "Risk Limit", "Initial Rate", "Maintenance Rate"}, rows)
}

func runFuturesLiquidations(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	limit, _ := cmd.Flags().GetInt32("limit")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.ListLiquidatedOrdersOpts{}
	if contract != "" {
		opts.Contract = optional.NewString(contract)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.FuturesAPI.ListLiquidatedOrders(c.Context(), settle, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/liq_orders", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{fmt.Sprintf("%d", r.Time), r.Contract, r.Size, r.OrderSize}
	}
	return p.Table([]string{"Time", "Contract", "Size", "Order Size"}, rows)
}

func runFuturesInsurance(cmd *cobra.Command, args []string) error {
	limit, _ := cmd.Flags().GetInt32("limit")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.ListFuturesInsuranceLedgerOpts{}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.FuturesAPI.ListFuturesInsuranceLedger(c.Context(), settle, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/insurance", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{fmt.Sprintf("%d", r.T), r.B}
	}
	return p.Table([]string{"Timestamp", "Balance"}, rows)
}

func runFuturesBatchFundingRates(cmd *cobra.Command, args []string) error {
	contracts, _ := cmd.Flags().GetStringSlice("contracts")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	req := gateapi.BatchFundingRatesRequest{Contracts: contracts}
	body, _ := json.Marshal(req)
	result, httpResp, err := c.FuturesAPI.ListBatchFuturesFundingRates(c.Context(), settle, req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/futures/"+settle+"/batch_funding_rate", string(body)))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, 0)
	for _, r := range result {
		for _, d := range r.Data {
			rows = append(rows, []string{r.Contract, fmt.Sprintf("%d", d.T), d.R})
		}
	}
	return p.Table([]string{"Contract", "Timestamp", "Rate"}, rows)
}
