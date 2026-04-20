package options

import (
	"fmt"
	"strconv"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
)

var marketCmd = &cobra.Command{
	Use:   "market",
	Short: "Options market data (public, no authentication required)",
}

func init() {
	underlyingsCmd := &cobra.Command{
		Use:   "underlyings",
		Short: "List options underlyings (public, no authentication required)",
		RunE:  runOptionsUnderlyings,
	}

	expirationsCmd := &cobra.Command{
		Use:   "expirations",
		Short: "List expiration dates for an underlying (public, no authentication required)",
		RunE:  runOptionsExpirations,
	}
	expirationsCmd.Flags().String("underlying", "", "Underlying name (required)")
	expirationsCmd.MarkFlagRequired("underlying")

	contractsCmd := &cobra.Command{
		Use:   "contracts",
		Short: "List options contracts for an underlying (public, no authentication required)",
		RunE:  runOptionsContracts,
	}
	contractsCmd.Flags().String("underlying", "", "Underlying name (required)")
	contractsCmd.Flags().Int64("expiration", 0, "Filter by expiration Unix timestamp")
	contractsCmd.MarkFlagRequired("underlying")

	contractCmd := &cobra.Command{
		Use:   "contract",
		Short: "Get details of an options contract (public, no authentication required)",
		RunE:  runOptionsContract,
	}
	contractCmd.Flags().String("contract", "", "Options contract name (required)")
	contractCmd.MarkFlagRequired("contract")

	settlementsCmd := &cobra.Command{
		Use:   "settlements",
		Short: "List settlement history for an underlying (public, no authentication required)",
		RunE:  runOptionsSettlements,
	}
	settlementsCmd.Flags().String("underlying", "", "Underlying name (required)")
	settlementsCmd.MarkFlagRequired("underlying")

	settlementCmd := &cobra.Command{
		Use:   "settlement",
		Short: "Get specific settlement record (public, no authentication required)",
		RunE:  runOptionsSettlement,
	}
	settlementCmd.Flags().String("contract", "", "Options contract name (required)")
	settlementCmd.Flags().String("underlying", "", "Underlying name (required)")
	settlementCmd.Flags().Int64("at", 0, "Settlement time Unix timestamp (required)")
	settlementCmd.MarkFlagRequired("contract")
	settlementCmd.MarkFlagRequired("underlying")
	settlementCmd.MarkFlagRequired("at")

	orderBookCmd := &cobra.Command{
		Use:   "order-book",
		Short: "Get order book for an options contract (public, no authentication required)",
		RunE:  runOptionsOrderBook,
	}
	orderBookCmd.Flags().String("contract", "", "Options contract name (required)")
	orderBookCmd.Flags().String("interval", "", "Order book merge interval")
	orderBookCmd.Flags().Int32("limit", 0, "Order book depth limit")
	orderBookCmd.MarkFlagRequired("contract")

	tickersCmd := &cobra.Command{
		Use:   "tickers",
		Short: "List options tickers for an underlying (public, no authentication required)",
		RunE:  runOptionsTickers,
	}
	tickersCmd.Flags().String("underlying", "", "Underlying name (required)")
	tickersCmd.MarkFlagRequired("underlying")

	underlyingTickersCmd := &cobra.Command{
		Use:   "underlying-tickers",
		Short: "Get ticker for an options underlying (public, no authentication required)",
		RunE:  runOptionsUnderlyingTickers,
	}
	underlyingTickersCmd.Flags().String("underlying", "", "Underlying name (required)")
	underlyingTickersCmd.MarkFlagRequired("underlying")

	candlesticksCmd := &cobra.Command{
		Use:   "candlesticks",
		Short: "List candlesticks for an options contract (public, no authentication required)",
		RunE:  runOptionsCandlesticks,
	}
	candlesticksCmd.Flags().String("contract", "", "Options contract name (required)")
	candlesticksCmd.Flags().String("interval", "", "Candlestick interval")
	candlesticksCmd.Flags().Int32("limit", 0, "Number of records")
	candlesticksCmd.MarkFlagRequired("contract")

	underlyingCandlesticksCmd := &cobra.Command{
		Use:   "underlying-candlesticks",
		Short: "List candlesticks for an options underlying (public, no authentication required)",
		RunE:  runOptionsUnderlyingCandlesticks,
	}
	underlyingCandlesticksCmd.Flags().String("underlying", "", "Underlying name (required)")
	underlyingCandlesticksCmd.Flags().String("interval", "", "Candlestick interval")
	underlyingCandlesticksCmd.Flags().Int32("limit", 0, "Number of records")
	underlyingCandlesticksCmd.MarkFlagRequired("underlying")

	tradesCmd := &cobra.Command{
		Use:   "trades",
		Short: "List recent options trades (public, no authentication required)",
		RunE:  runOptionsTrades,
	}
	tradesCmd.Flags().String("contract", "", "Filter by contract name")
	tradesCmd.Flags().Int32("limit", 0, "Number of records")

	marketCmd.AddCommand(
		underlyingsCmd, expirationsCmd, contractsCmd, contractCmd,
		settlementsCmd, settlementCmd, orderBookCmd, tickersCmd,
		underlyingTickersCmd, candlesticksCmd, underlyingCandlesticksCmd, tradesCmd,
	)
	Cmd.AddCommand(marketCmd)
}

func runOptionsUnderlyings(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.OptionsAPI.ListOptionsUnderlyings(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/options/underlyings", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, u := range result {
		rows[i] = []string{u.Name, u.IndexPrice}
	}
	return p.Table([]string{"Name", "Index Price"}, rows)
}

func runOptionsExpirations(cmd *cobra.Command, args []string) error {
	underlying, _ := cmd.Flags().GetString("underlying")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.OptionsAPI.ListOptionsExpirations(c.Context(), underlying)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/options/expirations", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, ts := range result {
		rows[i] = []string{fmt.Sprintf("%d", ts)}
	}
	return p.Table([]string{"Expiration Timestamp"}, rows)
}

func runOptionsContracts(cmd *cobra.Command, args []string) error {
	underlying, _ := cmd.Flags().GetString("underlying")
	expiration, _ := cmd.Flags().GetInt64("expiration")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.ListOptionsContractsOpts{}
	if expiration != 0 {
		opts.Expiration = optional.NewInt64(expiration)
	}

	result, httpResp, err := c.OptionsAPI.ListOptionsContracts(c.Context(), underlying, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/options/contracts", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, ct := range result {
		callPut := "put"
		if ct.IsCall {
			callPut = "call"
		}
		rows[i] = []string{ct.Name, callPut, strconv.FormatFloat(ct.ExpirationTime, 'f', 0, 64), ct.MarkPrice, ct.LastPrice}
	}
	return p.Table([]string{"Name", "Type", "Expiration", "Mark Price", "Last Price"}, rows)
}

func runOptionsContract(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.OptionsAPI.GetOptionsContract(c.Context(), contract)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/options/contracts/"+contract, ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	callPut := "put"
	if result.IsCall {
		callPut = "call"
	}
	return p.Table(
		[]string{"Name", "Type", "Underlying", "Expiration", "Mark Price", "Last Price"},
		[][]string{{result.Name, callPut, result.Underlying, strconv.FormatFloat(result.ExpirationTime, 'f', 0, 64), result.MarkPrice, result.LastPrice}},
	)
}

func runOptionsSettlements(cmd *cobra.Command, args []string) error {
	underlying, _ := cmd.Flags().GetString("underlying")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.OptionsAPI.ListOptionsSettlements(c.Context(), underlying, nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/options/settlements", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, s := range result {
		rows[i] = []string{s.Contract, strconv.FormatFloat(s.Time, 'f', 0, 64), s.StrikePrice, s.SettlePrice, s.Profit}
	}
	return p.Table([]string{"Contract", "Time", "Strike Price", "Settle Price", "Profit"}, rows)
}

func runOptionsSettlement(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	underlying, _ := cmd.Flags().GetString("underlying")
	at, _ := cmd.Flags().GetInt64("at")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.OptionsAPI.GetOptionsSettlement(c.Context(), contract, underlying, at)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/options/settlements/"+contract, ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"Contract", "Time", "Strike Price", "Settle Price", "Profit"},
		[][]string{{result.Contract, strconv.FormatFloat(result.Time, 'f', 0, 64), result.StrikePrice, result.SettlePrice, result.Profit}},
	)
}

func runOptionsOrderBook(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	interval, _ := cmd.Flags().GetString("interval")
	limit, _ := cmd.Flags().GetInt32("limit")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.ListOptionsOrderBookOpts{}
	if interval != "" {
		opts.Interval = optional.NewString(interval)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.OptionsAPI.ListOptionsOrderBook(c.Context(), contract, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/options/order_book", ""))
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

func runOptionsTickers(cmd *cobra.Command, args []string) error {
	underlying, _ := cmd.Flags().GetString("underlying")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.OptionsAPI.ListOptionsTickers(c.Context(), underlying)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/options/tickers", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, t := range result {
		rows[i] = []string{t.Name, t.LastPrice, t.MarkPrice, fmt.Sprintf("%d", t.PositionSize), t.Bid1Price, t.Ask1Price}
	}
	return p.Table([]string{"Contract", "Last", "Mark", "Position Size", "Bid", "Ask"}, rows)
}

func runOptionsUnderlyingTickers(cmd *cobra.Command, args []string) error {
	underlying, _ := cmd.Flags().GetString("underlying")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.OptionsAPI.ListOptionsUnderlyingTickers(c.Context(), underlying)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/options/underlyings/"+underlying+"/tickers", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"Trade Put", "Trade Call", "Index Price"},
		[][]string{{fmt.Sprintf("%d", result.TradePut), fmt.Sprintf("%d", result.TradeCall), result.IndexPrice}},
	)
}

func runOptionsCandlesticks(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	interval, _ := cmd.Flags().GetString("interval")
	limit, _ := cmd.Flags().GetInt32("limit")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.ListOptionsCandlesticksOpts{}
	if interval != "" {
		opts.Interval = optional.NewString(interval)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.OptionsAPI.ListOptionsCandlesticks(c.Context(), contract, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/options/candlesticks", ""))
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

func runOptionsUnderlyingCandlesticks(cmd *cobra.Command, args []string) error {
	underlying, _ := cmd.Flags().GetString("underlying")
	interval, _ := cmd.Flags().GetString("interval")
	limit, _ := cmd.Flags().GetInt32("limit")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.ListOptionsUnderlyingCandlesticksOpts{}
	if interval != "" {
		opts.Interval = optional.NewString(interval)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.OptionsAPI.ListOptionsUnderlyingCandlesticks(c.Context(), underlying, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/options/underlyings/"+underlying+"/candlesticks", ""))
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

func runOptionsTrades(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	limit, _ := cmd.Flags().GetInt32("limit")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.ListOptionsTradesOpts{}
	if contract != "" {
		opts.Contract = optional.NewString(contract)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.OptionsAPI.ListOptionsTrades(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/options/trades", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, t := range result {
		rows[i] = []string{fmt.Sprintf("%d", t.Id), t.Contract, fmt.Sprintf("%d", t.Size), t.Price, fmt.Sprintf("%d", t.CreateTime)}
	}
	return p.Table([]string{"ID", "Contract", "Size", "Price", "Time"}, rows)
}
