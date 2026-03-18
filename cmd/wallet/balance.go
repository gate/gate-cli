package wallet

import (
	"fmt"
	"strings"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
)

func init() {
	totalBalanceCmd := &cobra.Command{
		Use:   "total-balance",
		Short: "Get total estimated balance across all accounts",
		RunE:  runWalletTotalBalance,
	}
	totalBalanceCmd.Flags().String("currency", "", "Target currency for conversion (BTC, CNY, USD, USDT)")

	subBalancesCmd := &cobra.Command{
		Use:   "sub-balances",
		Short: "List sub-account spot balances",
		RunE:  runWalletSubBalances,
	}
	subBalancesCmd.Flags().String("sub-uid", "", "Filter by sub-account user IDs (comma-separated)")

	subMarginBalancesCmd := &cobra.Command{
		Use:   "sub-margin-balances",
		Short: "List sub-account margin account balances",
		RunE:  runWalletSubMarginBalances,
	}
	subMarginBalancesCmd.Flags().String("sub-uid", "", "Filter by sub-account user IDs (comma-separated)")

	subFuturesBalancesCmd := &cobra.Command{
		Use:   "sub-futures-balances",
		Short: "List sub-account perpetual futures account balances",
		RunE:  runWalletSubFuturesBalances,
	}
	subFuturesBalancesCmd.Flags().String("sub-uid", "", "Filter by sub-account user IDs (comma-separated)")
	subFuturesBalancesCmd.Flags().String("settle", "", "Filter by settlement currency")

	subCrossMarginBalancesCmd := &cobra.Command{
		Use:   "sub-cross-margin-balances",
		Short: "List sub-account cross-margin account balances",
		RunE:  runWalletSubCrossMarginBalances,
	}
	subCrossMarginBalancesCmd.Flags().String("sub-uid", "", "Filter by sub-account user IDs (comma-separated)")

	smallBalanceCmd := &cobra.Command{
		Use:   "small-balance",
		Short: "List small balances convertible to GT",
		RunE:  runWalletSmallBalance,
	}

	smallBalanceHistoryCmd := &cobra.Command{
		Use:   "small-balance-history",
		Short: "List small balance conversion history",
		RunE:  runWalletSmallBalanceHistory,
	}
	smallBalanceHistoryCmd.Flags().String("currency", "", "Filter by currency")
	smallBalanceHistoryCmd.Flags().Int32("page", 0, "Page number")
	smallBalanceHistoryCmd.Flags().Int32("limit", 0, "Number of records to return")

	convertSmallBalanceCmd := &cobra.Command{
		Use:   "convert-small-balance",
		Short: "Convert small balances to GT",
		RunE:  runWalletConvertSmallBalance,
	}
	convertSmallBalanceCmd.Flags().StringSlice("currencies", nil, "Currencies to convert (omit for all)")
	convertSmallBalanceCmd.Flags().Bool("all", false, "Convert all small balances")

	Cmd.AddCommand(
		totalBalanceCmd,
		subBalancesCmd,
		subMarginBalancesCmd,
		subFuturesBalancesCmd,
		subCrossMarginBalancesCmd,
		smallBalanceCmd,
		smallBalanceHistoryCmd,
		convertSmallBalanceCmd,
	)
}

func runWalletTotalBalance(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.GetTotalBalanceOpts{}
	if currency != "" {
		opts.Currency = optional.NewString(currency)
	}

	result, httpResp, err := c.WalletAPI.GetTotalBalance(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/wallet/total_balance", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"Total Amount", "Total Currency", "Account Breakdown"},
		[][]string{{result.Total.Amount, result.Total.Currency, fmt.Sprintf("%d accounts", len(result.Details))}},
	)
}

func runWalletSubBalances(cmd *cobra.Command, args []string) error {
	subUID, _ := cmd.Flags().GetString("sub-uid")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListSubAccountBalancesOpts{}
	if subUID != "" {
		opts.SubUid = optional.NewString(subUID)
	}

	result, httpResp, err := c.WalletAPI.ListSubAccountBalances(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/wallet/sub_account_balances", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, b := range result {
		currencies := make([]string, 0, len(b.Available))
		for cur, bal := range b.Available {
			currencies = append(currencies, cur+":"+bal)
		}
		rows[i] = []string{b.Uid, strings.Join(currencies, " ")}
	}
	return p.Table([]string{"Sub UID", "Available Balances"}, rows)
}

func runWalletSubMarginBalances(cmd *cobra.Command, args []string) error {
	subUID, _ := cmd.Flags().GetString("sub-uid")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListSubAccountMarginBalancesOpts{}
	if subUID != "" {
		opts.SubUid = optional.NewString(subUID)
	}

	result, httpResp, err := c.WalletAPI.ListSubAccountMarginBalances(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/wallet/sub_account_margin_balances", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, b := range result {
		rows[i] = []string{b.Uid, fmt.Sprintf("%d margin accounts", len(b.Available))}
	}
	return p.Table([]string{"Sub UID", "Margin Accounts"}, rows)
}

func runWalletSubFuturesBalances(cmd *cobra.Command, args []string) error {
	subUID, _ := cmd.Flags().GetString("sub-uid")
	settle, _ := cmd.Flags().GetString("settle")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListSubAccountFuturesBalancesOpts{}
	if subUID != "" {
		opts.SubUid = optional.NewString(subUID)
	}
	if settle != "" {
		opts.Settle = optional.NewString(settle)
	}

	result, httpResp, err := c.WalletAPI.ListSubAccountFuturesBalances(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/wallet/sub_account_futures_balances", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, b := range result {
		rows[i] = []string{b.Uid, fmt.Sprintf("%d settle currencies", len(b.Available))}
	}
	return p.Table([]string{"Sub UID", "Futures Balances"}, rows)
}

func runWalletSubCrossMarginBalances(cmd *cobra.Command, args []string) error {
	subUID, _ := cmd.Flags().GetString("sub-uid")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListSubAccountCrossMarginBalancesOpts{}
	if subUID != "" {
		opts.SubUid = optional.NewString(subUID)
	}

	result, httpResp, err := c.WalletAPI.ListSubAccountCrossMarginBalances(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/wallet/sub_account_cross_margin_balances", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, b := range result {
		rows[i] = []string{b.Uid, b.Available.Total, b.Available.Net}
	}
	return p.Table([]string{"Sub UID", "Total", "Net"}, rows)
}

func runWalletSmallBalance(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.WalletAPI.ListSmallBalance(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/wallet/small_balance", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, b := range result {
		rows[i] = []string{b.Currency, b.AvailableBalance, b.EstimatedAsBtc, b.ConvertibleToGt}
	}
	return p.Table([]string{"Currency", "Balance", "Est. BTC", "Convertible GT"}, rows)
}

func runWalletSmallBalanceHistory(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	page, _ := cmd.Flags().GetInt32("page")
	limit, _ := cmd.Flags().GetInt32("limit")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListSmallBalanceHistoryOpts{}
	if currency != "" {
		opts.Currency = optional.NewString(currency)
	}
	if page != 0 {
		opts.Page = optional.NewInt32(page)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.WalletAPI.ListSmallBalanceHistory(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/wallet/small_balance_history", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, h := range result {
		rows[i] = []string{h.Id, h.Currency, h.Amount, h.GtAmount, fmt.Sprintf("%d", h.CreateTime)}
	}
	return p.Table([]string{"ID", "Currency", "Amount", "GT Amount", "Created At"}, rows)
}

func runWalletConvertSmallBalance(cmd *cobra.Command, args []string) error {
	currencies, _ := cmd.Flags().GetStringSlice("currencies")
	isAll, _ := cmd.Flags().GetBool("all")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	req := gateapi.ConvertSmallBalance{
		Currency: currencies,
		IsAll:    isAll,
	}
	httpResp, err := c.WalletAPI.ConvertSmallBalance(c.Context(), req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/wallet/small_balance", ""))
		return nil
	}
	return p.Table([]string{"Status"}, [][]string{{"success"}})
}
