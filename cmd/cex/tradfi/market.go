package tradfi

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
	Short: "TradFi market data (public, no authentication required)",
}

func init() {
	categoriesCmd := &cobra.Command{
		Use:   "categories",
		Short: "List all TradFi symbol categories",
		RunE:  runTradfiCategories,
	}

	symbolsCmd := &cobra.Command{
		Use:   "symbols",
		Short: "List all TradFi symbols",
		RunE:  runTradfiSymbols,
	}

	symbolCmd := &cobra.Command{
		Use:   "symbol",
		Short: "Get details for a TradFi symbol",
		RunE:  runTradfiSymbol,
	}
	symbolCmd.Flags().String("symbol", "", "Symbol name, e.g. XAUUSD (required)")
	symbolCmd.MarkFlagRequired("symbol")

	tickerCmd := &cobra.Command{
		Use:   "ticker",
		Short: "Get ticker for a TradFi symbol",
		RunE:  runTradfiTicker,
	}
	tickerCmd.Flags().String("symbol", "", "Symbol name, e.g. XAUUSD (required)")
	tickerCmd.MarkFlagRequired("symbol")

	klineCmd := &cobra.Command{
		Use:   "kline",
		Short: "Get candlestick data for a TradFi symbol",
		RunE:  runTradfiKline,
	}
	klineCmd.Flags().String("symbol", "", "Symbol name, e.g. XAUUSD (required)")
	klineCmd.Flags().String("type", "M1", "Kline type: M1, M5, M15, M30, H1, H4, D1")
	klineCmd.Flags().Int32("limit", 100, "Number of candlesticks to return")
	klineCmd.MarkFlagRequired("symbol")

	marketCmd.AddCommand(categoriesCmd, symbolsCmd, symbolCmd, tickerCmd, klineCmd)
	Cmd.AddCommand(marketCmd)
}

func runTradfiCategories(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.TradFiAPI.QueryCategories(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/tradfi/symbols/categories", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	if result.Data.List == nil {
		return p.Table([]string{"ID", "Name"}, nil)
	}
	rows := make([][]string, len(result.Data.List))
	for i, cat := range result.Data.List {
		rows[i] = []string{fmt.Sprintf("%d", cat.CategoryId), cat.CategoryName}
	}
	return p.Table([]string{"ID", "Name"}, rows)
}

func runTradfiSymbols(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.TradFiAPI.QuerySymbols(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/tradfi/symbols", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	if result.Data.List == nil {
		return p.Table([]string{"Symbol", "Description", "Category", "Status", "Currency"}, nil)
	}
	rows := make([][]string, len(result.Data.List))
	for i, s := range result.Data.List {
		rows[i] = []string{s.Symbol, s.SymbolDesc, fmt.Sprintf("%d", s.CategoryId), s.Status, s.SettlementCurrency}
	}
	return p.Table([]string{"Symbol", "Description", "Category", "Status", "Currency"}, rows)
}

func runTradfiSymbol(cmd *cobra.Command, args []string) error {
	symbol, _ := cmd.Flags().GetString("symbol")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.TradFiAPI.QuerySymbolDetail(c.Context(), symbol)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/tradfi/symbols/"+symbol, ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	if result.Data.List == nil {
		return p.Table([]string{"Symbol", "Description", "Category", "Trade Mode", "Currency", "Price Precision"}, nil)
	}
	rows := make([][]string, len(result.Data.List))
	for i, s := range result.Data.List {
		rows[i] = []string{s.Symbol, s.SymbolDesc, s.CategoryName, s.TradeMode, s.SettlementCurrency, fmt.Sprintf("%d", s.PricePrecision)}
	}
	return p.Table([]string{"Symbol", "Description", "Category", "Trade Mode", "Currency", "Price Precision"}, rows)
}

func runTradfiTicker(cmd *cobra.Command, args []string) error {
	symbol, _ := cmd.Flags().GetString("symbol")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.TradFiAPI.QuerySymbolTicker(c.Context(), symbol)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/tradfi/symbols/"+symbol+"/tickers", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	d := result.Data
	return p.Table(
		[]string{"Last", "Bid", "Ask", "High", "Low", "Change", "Change %"},
		[][]string{{d.LastPrice, d.BidPrice, d.AskPrice, d.HighestPrice, d.LowestPrice, d.PriceChangeAmount, d.PriceChange}},
	)
}

func runTradfiKline(cmd *cobra.Command, args []string) error {
	symbol, _ := cmd.Flags().GetString("symbol")
	klineType, _ := cmd.Flags().GetString("type")
	limit, _ := cmd.Flags().GetInt32("limit")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.TradFiAPI.QuerySymbolKline(c.Context(), symbol, klineType, &gateapi.QuerySymbolKlineOpts{
		Limit: optional.NewInt32(limit),
	})
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/tradfi/symbols/"+symbol+"/klines", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	if result.Data.List == nil {
		return p.Table([]string{"Timestamp", "Open", "Close", "High", "Low"}, nil)
	}
	rows := make([][]string, len(result.Data.List))
	for i, k := range result.Data.List {
		rows[i] = []string{fmt.Sprintf("%d", k.T), k.O, k.C, k.H, k.L}
	}
	return p.Table([]string{"Timestamp", "Open", "Close", "High", "Low"}, rows)
}
