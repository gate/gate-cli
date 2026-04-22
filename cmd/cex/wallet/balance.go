package wallet

import (
	"fmt"
	"strings"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

var balanceCmd = &cobra.Command{
	Use:   "balance",
	Short: "Wallet balance commands",
}

func init() {
	totalCmd := &cobra.Command{
		Use:   "total",
		Short: "Get total estimated balance across all accounts",
		RunE:  runWalletTotalBalance,
	}
	totalCmd.Flags().String("currency", "", "Target currency for conversion (BTC, CNY, USD, USDT)")

	subCmd := &cobra.Command{
		Use:   "sub",
		Short: "List sub-account spot balances",
		RunE:  runWalletSubBalances,
	}
	subCmd.Flags().String("sub-uid", "", "Filter by sub-account user IDs (comma-separated)")
	subCmd.Flags().Int32("page", 0, "Page number (default 1)")
	subCmd.Flags().Int32("limit", 0, "Page size, max 100 (default 100)")

	subMarginCmd := &cobra.Command{
		Use:   "sub-margin",
		Short: "List sub-account margin account balances",
		RunE:  runWalletSubMarginBalances,
	}
	subMarginCmd.Flags().String("sub-uid", "", "Filter by sub-account user IDs (comma-separated)")

	subFuturesCmd := &cobra.Command{
		Use:   "sub-futures",
		Short: "List sub-account perpetual futures account balances",
		RunE:  runWalletSubFuturesBalances,
	}
	subFuturesCmd.Flags().String("sub-uid", "", "Filter by sub-account user IDs (comma-separated)")
	subFuturesCmd.Flags().String("settle", "", "Filter by settlement currency")

	subCrossMarginCmd := &cobra.Command{
		Use:   "sub-cross-margin",
		Short: "List sub-account cross-margin account balances",
		RunE:  runWalletSubCrossMarginBalances,
	}
	subCrossMarginCmd.Flags().String("sub-uid", "", "Filter by sub-account user IDs (comma-separated)")

	smallCmd := &cobra.Command{
		Use:   "small",
		Short: "List small balances convertible to GT",
		RunE:  runWalletSmallBalance,
	}

	smallHistoryCmd := &cobra.Command{
		Use:   "small-history",
		Short: "List small balance conversion history",
		RunE:  runWalletSmallBalanceHistory,
	}
	smallHistoryCmd.Flags().String("currency", "", "Filter by currency")
	smallHistoryCmd.Flags().Int32("page", 0, "Page number")
	smallHistoryCmd.Flags().Int32("limit", 0, "Number of records to return")

	convertSmallCmd := &cobra.Command{
		Use:   "convert-small",
		Short: "Convert small balances to GT",
		RunE:  runWalletConvertSmallBalance,
	}
	convertSmallCmd.Flags().StringSlice("currencies", nil, "Currencies to convert (omit for all)")
	convertSmallCmd.Flags().Bool("all", false, "Convert all small balances")

	balanceCmd.AddCommand(
		totalCmd,
		subCmd,
		subMarginCmd,
		subFuturesCmd,
		subCrossMarginCmd,
		smallCmd,
		smallHistoryCmd,
		convertSmallCmd,
	)
	Cmd.AddCommand(balanceCmd)
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

	opts := &gateapi.ListSubAccountBalancesOpts{}
	if subUID != "" {
		opts.SubUid = optional.NewString(subUID)
	}
	if page != 0 {
		opts.Page = optional.NewInt32(page)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
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
