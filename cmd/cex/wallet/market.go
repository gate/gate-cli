package wallet

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
	Short: "Wallet market info and fee commands",
}

func init() {
	chainsCmd := &cobra.Command{
		Use:   "chains",
		Short: "List supported chains for a currency (public, no authentication required)",
		RunE:  runWalletChains,
	}
	chainsCmd.Flags().String("currency", "", "Currency name (required)")
	chainsCmd.MarkFlagRequired("currency")

	withdrawStatusCmd := &cobra.Command{
		Use:   "withdraw-status",
		Short: "List withdrawal status for currencies (public, no authentication required)",
		RunE:  runWalletWithdrawStatus,
	}
	withdrawStatusCmd.Flags().String("currency", "", "Filter by currency name")

	tradeFeeCmd := &cobra.Command{
		Use:   "trade-fee",
		Short: "Get personal trading fee rates",
		RunE:  runWalletTradeFee,
	}
	tradeFeeCmd.Flags().String("currency-pair", "", "Specify currency pair for more accurate fee settings")
	tradeFeeCmd.Flags().String("settle", "", "Specify settlement currency for contract fees")

	lowCapCmd := &cobra.Command{
		Use:   "low-cap",
		Short: "List currencies available for low-cap exchange (public, no authentication required)",
		RunE:  runWalletLowCap,
	}

	marketCmd.AddCommand(chainsCmd, withdrawStatusCmd, tradeFeeCmd, lowCapCmd)
	Cmd.AddCommand(marketCmd)
}

func runWalletChains(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.WalletAPI.ListCurrencyChains(c.Context(), currency)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/wallet/currency_chains", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, ch := range result {
		enabled := "false"
		if ch.IsDisabled == 0 {
			enabled = "true"
		}
		rows[i] = []string{ch.Chain, ch.NameCn, fmt.Sprintf("%d", ch.IsDisabled), enabled}
	}
	return p.Table([]string{"Chain", "Name", "Disabled", "Enabled"}, rows)
}

func runWalletWithdrawStatus(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.ListWithdrawStatusOpts{}
	if currency != "" {
		opts.Currency = optional.NewString(currency)
	}

	result, httpResp, err := c.WalletAPI.ListWithdrawStatus(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/wallet/withdraw_status", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, s := range result {
		rows[i] = []string{s.Currency, s.Name, s.WithdrawFix, s.WithdrawPercent, s.WithdrawDayLimit}
	}
	return p.Table([]string{"Currency", "Name", "Fixed Fee", "% Fee", "Day Limit"}, rows)
}

func runWalletTradeFee(cmd *cobra.Command, args []string) error {
	currencyPair, _ := cmd.Flags().GetString("currency-pair")
	settle, _ := cmd.Flags().GetString("settle")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.GetTradeFeeOpts{}
	if currencyPair != "" {
		opts.CurrencyPair = optional.NewString(currencyPair)
	}
	if settle != "" {
		opts.Settle = optional.NewString(settle)
	}

	result, httpResp, err := c.WalletAPI.GetTradeFee(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/wallet/fee", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"User ID", "Taker Fee", "Maker Fee", "Futures Taker", "Futures Maker"},
		[][]string{{
			fmt.Sprintf("%d", result.UserId),
			result.TakerFee,
			result.MakerFee,
			result.FuturesTakerFee,
			result.FuturesMakerFee,
		}},
	)
}

func runWalletLowCap(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.WalletAPI.GetLowCapExchangeList(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/wallet/low_cap_exchange_list", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, s := range result {
		rows[i] = []string{s}
	}
	return p.Table([]string{"Currency"}, rows)
}
