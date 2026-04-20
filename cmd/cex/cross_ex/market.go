package crossex

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
	Short: "Cross-exchange market data commands",
}

func init() {
	symbolsCmd := &cobra.Command{
		Use:   "symbols",
		Short: "Query trading pair information",
		RunE:  runSymbols,
	}
	symbolsCmd.Flags().String("symbols", "", "Symbol list, comma-separated (e.g. BINANCE_FUTURE_ADA_USDT,OKX_FUTURE_ADA_USDT)")

	riskLimitsCmd := &cobra.Command{
		Use:   "risk-limits",
		Short: "Query risk limit information",
		RunE:  runRiskLimits,
	}
	riskLimitsCmd.Flags().String("symbols", "", "Symbol list, comma-separated (required)")
	riskLimitsCmd.MarkFlagRequired("symbols")

	transferCoinsCmd := &cobra.Command{
		Use:   "transfer-coins",
		Short: "Query supported transfer currencies",
		RunE:  runTransferCoins,
	}
	transferCoinsCmd.Flags().String("coin", "", "Filter by currency")

	feeCmd := &cobra.Command{
		Use:   "fee",
		Short: "Query cross-exchange fee rates",
		RunE:  runFee,
	}

	interestRateCmd := &cobra.Command{
		Use:   "interest-rate",
		Short: "Query margin asset interest rates",
		RunE:  runInterestRate,
	}
	interestRateCmd.Flags().String("coin", "", "Filter by currency")
	interestRateCmd.Flags().String("exchange-type", "", "Exchange (BINANCE/OKX/GATE/BYBIT)")

	discountRateCmd := &cobra.Command{
		Use:   "discount-rate",
		Short: "Query currency discount rate",
		RunE:  runDiscountRate,
	}
	discountRateCmd.Flags().String("coin", "", "Filter by currency")
	discountRateCmd.Flags().String("exchange-type", "", "Exchange (OKX/GATE/BINANCE/BYBIT)")

	marketCmd.AddCommand(symbolsCmd, riskLimitsCmd, transferCoinsCmd, feeCmd, interestRateCmd, discountRateCmd)
	Cmd.AddCommand(marketCmd)
}

func runSymbols(cmd *cobra.Command, args []string) error {
	symbols, _ := cmd.Flags().GetString("symbols")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	var opts *gateapi.ListCrossexRuleSymbolsOpts
	if symbols != "" {
		opts = &gateapi.ListCrossexRuleSymbolsOpts{
			Symbols: optional.NewString(symbols),
		}
	}

	result, httpResp, err := c.CrossExAPI.ListCrossexRuleSymbols(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/crossex/rule/symbols", ""))
		return nil
	}
	return p.Print(result)
}

func runRiskLimits(cmd *cobra.Command, args []string) error {
	symbols, _ := cmd.Flags().GetString("symbols")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.CrossExAPI.ListCrossexRuleRiskLimits(c.Context(), symbols)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/crossex/rule/risk_limits", ""))
		return nil
	}
	return p.Print(result)
}

func runTransferCoins(cmd *cobra.Command, args []string) error {
	coin, _ := cmd.Flags().GetString("coin")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	var opts *gateapi.ListCrossexTransferCoinsOpts
	if coin != "" {
		opts = &gateapi.ListCrossexTransferCoinsOpts{
			Coin: optional.NewString(coin),
		}
	}

	result, httpResp, err := c.CrossExAPI.ListCrossexTransferCoins(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/crossex/rule/transfer_coins", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{r.Coin, fmt.Sprintf("%.8f", r.MinTransAmount), fmt.Sprintf("%.8f", r.EstFee), fmt.Sprintf("%d", r.Precision), fmt.Sprintf("%d", r.IsDisabled)}
	}
	return p.Table([]string{"Coin", "Min Amount", "Est Fee", "Precision", "Disabled"}, rows)
}

func runFee(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.CrossExAPI.GetCrossexFee(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/crossex/fee", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{r.ExchangeType, r.SpotMakerFee, r.SpotTakerFee, r.FutureMakerFee, r.FutureTakerFee}
	}
	return p.Table([]string{"Exchange", "Spot Maker", "Spot Taker", "Future Maker", "Future Taker"}, rows)
}

func runInterestRate(cmd *cobra.Command, args []string) error {
	coin, _ := cmd.Flags().GetString("coin")
	exchangeType, _ := cmd.Flags().GetString("exchange-type")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.GetCrossexInterestRateOpts{}
	if coin != "" {
		opts.Coin = optional.NewString(coin)
	}
	if exchangeType != "" {
		opts.ExchangeType = optional.NewString(exchangeType)
	}

	result, httpResp, err := c.CrossExAPI.GetCrossexInterestRate(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/crossex/interest_rate", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{r.Coin, r.ExchangeType, r.HourInterestRate, r.Time}
	}
	return p.Table([]string{"Coin", "Exchange", "Hourly Rate", "Time"}, rows)
}

func runDiscountRate(cmd *cobra.Command, args []string) error {
	coin, _ := cmd.Flags().GetString("coin")
	exchangeType, _ := cmd.Flags().GetString("exchange-type")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.ListCrossexCoinDiscountRateOpts{}
	if coin != "" {
		opts.Coin = optional.NewString(coin)
	}
	if exchangeType != "" {
		opts.ExchangeType = optional.NewString(exchangeType)
	}

	result, httpResp, err := c.CrossExAPI.ListCrossexCoinDiscountRate(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/crossex/coin_discount_rate", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{r.Coin, r.ExchangeType, r.Tier, r.MinValue, r.MaxValue, r.DiscountRate}
	}
	return p.Table([]string{"Coin", "Exchange", "Tier", "Min Value", "Max Value", "Discount Rate"}, rows)
}
