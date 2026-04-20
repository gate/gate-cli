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

func getSettle(_ *cobra.Command) string {
	return "usdt"
}

var marketCmd = &cobra.Command{
	Use:   "market",
	Short: "Delivery market data (public, no authentication required)",
}

func init() {
	contractsCmd := &cobra.Command{
		Use:   "contracts",
		Short: "List delivery contracts (public, no authentication required)",
		RunE:  runDeliveryContracts,
	}

	contractCmd := &cobra.Command{
		Use:   "contract",
		Short: "Get details of a delivery contract (public, no authentication required)",
		RunE:  runDeliveryContract,
	}
	contractCmd.Flags().String("contract", "", "Futures contract name (required)")
	contractCmd.MarkFlagRequired("contract")

	orderBookCmd := &cobra.Command{
		Use:   "order-book",
		Short: "Get order book for a delivery contract (public, no authentication required)",
		RunE:  runDeliveryOrderBook,
	}
	orderBookCmd.Flags().String("contract", "", "Futures contract name (required)")
	orderBookCmd.Flags().Int32("limit", 0, "Order book depth limit")
	orderBookCmd.MarkFlagRequired("contract")

	tradesCmd := &cobra.Command{
		Use:   "trades",
		Short: "List recent trades for a delivery contract (public, no authentication required)",
		RunE:  runDeliveryTrades,
	}
	tradesCmd.Flags().String("contract", "", "Futures contract name (required)")
	tradesCmd.Flags().Int32("limit", 0, "Number of records to return")
	tradesCmd.MarkFlagRequired("contract")

	candlesticksCmd := &cobra.Command{
		Use:   "candlesticks",
		Short: "List candlesticks for a delivery contract (public, no authentication required)",
		RunE:  runDeliveryCandlesticks,
	}
	candlesticksCmd.Flags().String("contract", "", "Futures contract name (required)")
	candlesticksCmd.Flags().String("interval", "", "Candlestick interval")
	candlesticksCmd.Flags().Int32("limit", 0, "Number of records")
	candlesticksCmd.MarkFlagRequired("contract")

	tickersCmd := &cobra.Command{
		Use:   "tickers",
		Short: "List delivery tickers (public, no authentication required)",
		RunE:  runDeliveryTickers,
	}
	tickersCmd.Flags().String("contract", "", "Filter by contract name")

	insuranceCmd := &cobra.Command{
		Use:   "insurance",
		Short: "List insurance fund records (public, no authentication required)",
		RunE:  runDeliveryInsurance,
	}
	insuranceCmd.Flags().Int32("limit", 0, "Number of records to return")

	riskLimitTiersCmd := &cobra.Command{
		Use:   "risk-limit-tiers",
		Short: "List risk limit tiers for a contract (public, no authentication required)",
		RunE:  runDeliveryRiskLimitTiers,
	}
	riskLimitTiersCmd.Flags().String("contract", "", "Futures contract name (required)")
	riskLimitTiersCmd.MarkFlagRequired("contract")

	marketCmd.AddCommand(contractsCmd, contractCmd, orderBookCmd, tradesCmd, candlesticksCmd, tickersCmd, insuranceCmd, riskLimitTiersCmd)
	Cmd.AddCommand(marketCmd)
}

func runDeliveryContracts(cmd *cobra.Command, args []string) error {
	settle := getSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.DeliveryAPI.ListDeliveryContracts(c.Context(), settle)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/delivery/"+settle+"/contracts", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, ct := range result {
		rows[i] = []string{ct.Name, ct.Underlying, ct.Cycle, ct.Type, ct.MarkPrice, ct.LastPrice}
	}
	return p.Table([]string{"Name", "Underlying", "Cycle", "Type", "Mark Price", "Last Price"}, rows)
}

func runDeliveryContract(cmd *cobra.Command, args []string) error {
	settle := getSettle(cmd)
	contract, _ := cmd.Flags().GetString("contract")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.DeliveryAPI.GetDeliveryContract(c.Context(), settle, contract)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/delivery/"+settle+"/contracts/"+contract, ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"Name", "Underlying", "Cycle", "Type", "Mark Price", "Last Price"},
		[][]string{{result.Name, result.Underlying, result.Cycle, result.Type, result.MarkPrice, result.LastPrice}},
	)
}

func runDeliveryOrderBook(cmd *cobra.Command, args []string) error {
	settle := getSettle(cmd)
	contract, _ := cmd.Flags().GetString("contract")
	limit, _ := cmd.Flags().GetInt32("limit")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.ListDeliveryOrderBookOpts{}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.DeliveryAPI.ListDeliveryOrderBook(c.Context(), settle, contract, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/delivery/"+settle+"/order_book", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	size := len(result.Asks)
	if len(result.Bids) > size {
		size = len(result.Bids)
	}
	rows := make([][]string, size)
	for i := range rows {
		askP, askS := "", ""
		bidP, bidS := "", ""
		if i < len(result.Asks) {
			askP, askS = result.Asks[i].P, fmt.Sprintf("%d", result.Asks[i].S)
		}
		if i < len(result.Bids) {
			bidP, bidS = result.Bids[i].P, fmt.Sprintf("%d", result.Bids[i].S)
		}
		rows[i] = []string{bidP, bidS, askP, askS}
	}
	return p.Table([]string{"Bid Price", "Bid Size", "Ask Price", "Ask Size"}, rows)
}

func runDeliveryTrades(cmd *cobra.Command, args []string) error {
	settle := getSettle(cmd)
	contract, _ := cmd.Flags().GetString("contract")
	limit, _ := cmd.Flags().GetInt32("limit")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.ListDeliveryTradesOpts{}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.DeliveryAPI.ListDeliveryTrades(c.Context(), settle, contract, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/delivery/"+settle+"/trades", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, t := range result {
		rows[i] = []string{fmt.Sprintf("%d", t.Id), fmt.Sprintf("%d", t.Size), t.Price, strconv.FormatFloat(t.CreateTime, 'f', 3, 64)}
	}
	return p.Table([]string{"ID", "Size", "Price", "Time"}, rows)
}

func runDeliveryCandlesticks(cmd *cobra.Command, args []string) error {
	settle := getSettle(cmd)
	contract, _ := cmd.Flags().GetString("contract")
	interval, _ := cmd.Flags().GetString("interval")
	limit, _ := cmd.Flags().GetInt32("limit")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.ListDeliveryCandlesticksOpts{}
	if interval != "" {
		opts.Interval = optional.NewString(interval)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.DeliveryAPI.ListDeliveryCandlesticks(c.Context(), settle, contract, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/delivery/"+settle+"/candlesticks", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, k := range result {
		rows[i] = []string{strconv.FormatFloat(k.T, 'f', 0, 64), k.O, k.H, k.L, k.C, fmt.Sprintf("%d", k.V)}
	}
	return p.Table([]string{"Time", "Open", "High", "Low", "Close", "Volume"}, rows)
}

func runDeliveryTickers(cmd *cobra.Command, args []string) error {
	settle := getSettle(cmd)
	contract, _ := cmd.Flags().GetString("contract")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.ListDeliveryTickersOpts{}
	if contract != "" {
		opts.Contract = optional.NewString(contract)
	}

	result, httpResp, err := c.DeliveryAPI.ListDeliveryTickers(c.Context(), settle, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/delivery/"+settle+"/tickers", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, t := range result {
		rows[i] = []string{t.Contract, t.Last, t.ChangePercentage, t.High24h, t.Low24h, t.Volume24h}
	}
	return p.Table([]string{"Contract", "Last", "Change %", "High 24h", "Low 24h", "Volume 24h"}, rows)
}

func runDeliveryInsurance(cmd *cobra.Command, args []string) error {
	settle := getSettle(cmd)
	limit, _ := cmd.Flags().GetInt32("limit")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.ListDeliveryInsuranceLedgerOpts{}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.DeliveryAPI.ListDeliveryInsuranceLedger(c.Context(), settle, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/delivery/"+settle+"/insurance", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{fmt.Sprintf("%d", r.T), r.B}
	}
	return p.Table([]string{"Time", "Balance"}, rows)
}

func runDeliveryRiskLimitTiers(cmd *cobra.Command, args []string) error {
	settle := getSettle(cmd)
	contract, _ := cmd.Flags().GetString("contract")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.ListDeliveryRiskLimitTiersOpts{}
	if contract != "" {
		opts.Contract = optional.NewString(contract)
	}
	result, httpResp, err := c.DeliveryAPI.ListDeliveryRiskLimitTiers(c.Context(), settle, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/delivery/"+settle+"/risk_limit_tiers", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, t := range result {
		rows[i] = []string{fmt.Sprintf("%d", t.Tier), t.RiskLimit, t.InitialRate, t.MaintenanceRate}
	}
	return p.Table([]string{"Tier", "Risk Limit", "Initial Rate", "Maintenance Rate"}, rows)
}
