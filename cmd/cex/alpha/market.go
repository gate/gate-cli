package alpha

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
	Short: "Alpha market data (public, no authentication required)",
}

func init() {
	currenciesCmd := &cobra.Command{
		Use:   "currencies",
		Short: "List Alpha currencies",
		RunE:  runAlphaCurrencies,
	}
	currenciesCmd.Flags().String("currency", "", "Filter by currency symbol")
	currenciesCmd.Flags().Int32("limit", 100, "Maximum number of records")
	currenciesCmd.Flags().Int32("page", 1, "Page number")

	tickersCmd := &cobra.Command{
		Use:   "tickers",
		Short: "List Alpha tickers",
		RunE:  runAlphaTickers,
	}
	tickersCmd.Flags().String("currency", "", "Filter by currency symbol")
	tickersCmd.Flags().Int32("limit", 100, "Maximum number of records")
	tickersCmd.Flags().Int32("page", 1, "Page number")

	tokensCmd := &cobra.Command{
		Use:   "tokens",
		Short: "List Alpha tokens",
		RunE:  runAlphaTokens,
	}
	tokensCmd.Flags().String("chain", "", "Filter by chain (e.g. solana, eth, bsc)")
	tokensCmd.Flags().String("platform", "", "Filter by launch platform (e.g. pump, gatefun)")
	tokensCmd.Flags().String("address", "", "Filter by contract address")
	tokensCmd.Flags().Int32("page", 1, "Page number")

	marketCmd.AddCommand(currenciesCmd, tickersCmd, tokensCmd)
	Cmd.AddCommand(marketCmd)
}

func runAlphaCurrencies(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	limit, _ := cmd.Flags().GetInt32("limit")
	page, _ := cmd.Flags().GetInt32("page")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.ListAlphaCurrenciesOpts{
		Limit: optional.NewInt32(limit),
		Page:  optional.NewInt32(page),
	}
	if currency != "" {
		opts.Currency = optional.NewString(currency)
	}

	result, httpResp, err := c.AlphaAPI.ListAlphaCurrencies(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/alpha/currencies", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, cur := range result {
		rows[i] = []string{cur.Currency, cur.Name, cur.Chain, cur.Address, fmt.Sprintf("%d", cur.Status)}
	}
	return p.Table([]string{"Currency", "Name", "Chain", "Address", "Status"}, rows)
}

func runAlphaTickers(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	limit, _ := cmd.Flags().GetInt32("limit")
	page, _ := cmd.Flags().GetInt32("page")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.ListAlphaTickersOpts{
		Limit: optional.NewInt32(limit),
		Page:  optional.NewInt32(page),
	}
	if currency != "" {
		opts.Currency = optional.NewString(currency)
	}

	result, httpResp, err := c.AlphaAPI.ListAlphaTickers(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/alpha/tickers", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, t := range result {
		rows[i] = []string{t.Currency, t.Last, t.Change, t.Volume, t.MarketCap}
	}
	return p.Table([]string{"Currency", "Last", "Change %", "Volume (USDT)", "Market Cap"}, rows)
}

func runAlphaTokens(cmd *cobra.Command, args []string) error {
	chain, _ := cmd.Flags().GetString("chain")
	platform, _ := cmd.Flags().GetString("platform")
	address, _ := cmd.Flags().GetString("address")
	page, _ := cmd.Flags().GetInt32("page")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.ListAlphaTokensOpts{
		Page: optional.NewInt32(page),
	}
	if chain != "" {
		opts.Chain = optional.NewString(chain)
	}
	if platform != "" {
		opts.LaunchPlatform = optional.NewString(platform)
	}
	if address != "" {
		opts.Address = optional.NewString(address)
	}

	result, httpResp, err := c.AlphaAPI.ListAlphaTokens(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/alpha/tokens", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, t := range result {
		rows[i] = []string{t.Currency, t.Name, t.Chain, t.Address, fmt.Sprintf("%d", t.Status)}
	}
	return p.Table([]string{"Currency", "Name", "Chain", "Address", "Status"}, rows)
}
