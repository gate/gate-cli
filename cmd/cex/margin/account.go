package margin

import (
	"fmt"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Margin account commands",
}

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List margin accounts",
		RunE:  runMarginAccountList,
	}
	listCmd.Flags().String("pair", "", "Filter by currency pair")

	bookCmd := &cobra.Command{
		Use:   "book",
		Short: "List margin account book records",
		RunE:  runMarginAccountBook,
	}
	bookCmd.Flags().String("currency", "", "Filter by currency")
	bookCmd.Flags().String("pair", "", "Filter by currency pair")
	bookCmd.Flags().String("type", "", "Filter by type")
	bookCmd.Flags().Int64("from", 0, "Start Unix timestamp")
	bookCmd.Flags().Int64("to", 0, "End Unix timestamp")
	bookCmd.Flags().Int32("page", 0, "Page number")
	bookCmd.Flags().Int32("limit", 0, "Number of records to return")

	fundingCmd := &cobra.Command{
		Use:   "funding",
		Short: "List funding accounts",
		RunE:  runMarginFunding,
	}
	fundingCmd.Flags().String("currency", "", "Filter by currency")

	transferableCmd := &cobra.Command{
		Use:   "transferable",
		Short: "Get maximum transferable amount for isolated margin",
		RunE:  runMarginTransferable,
	}
	transferableCmd.Flags().String("currency", "", "Currency name (required)")
	transferableCmd.MarkFlagRequired("currency")
	transferableCmd.Flags().String("pair", "", "Currency pair")

	userInfoCmd := &cobra.Command{
		Use:   "user-info",
		Short: "List user's isolated margin accounts",
		RunE:  runMarginUserInfo,
	}
	userInfoCmd.Flags().String("pair", "", "Filter by currency pair")

	accountCmd.AddCommand(listCmd, bookCmd, fundingCmd, transferableCmd, userInfoCmd)
	Cmd.AddCommand(accountCmd)
}

func runMarginAccountList(cmd *cobra.Command, args []string) error {
	pair, _ := cmd.Flags().GetString("pair")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var opts *gateapi.ListMarginAccountsOpts
	if pair != "" {
		opts = &gateapi.ListMarginAccountsOpts{
			CurrencyPair: optional.NewString(pair),
		}
	}

	result, httpResp, err := c.MarginAPI.ListMarginAccounts(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/margin/accounts", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, a := range result {
		rows[i] = []string{a.CurrencyPair, a.Leverage, a.Mmr, a.AccountType}
	}
	return p.Table([]string{"Pair", "Leverage", "MMR", "Account Type"}, rows)
}

func runMarginAccountBook(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	pair, _ := cmd.Flags().GetString("pair")
	typ, _ := cmd.Flags().GetString("type")
	from, _ := cmd.Flags().GetInt64("from")
	to, _ := cmd.Flags().GetInt64("to")
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

	opts := &gateapi.ListMarginAccountBookOpts{}
	if currency != "" {
		opts.Currency = optional.NewString(currency)
	}
	if pair != "" {
		opts.CurrencyPair = optional.NewString(pair)
	}
	if typ != "" {
		opts.Type_ = optional.NewString(typ)
	}
	if from != 0 {
		opts.From = optional.NewInt64(from)
	}
	if to != 0 {
		opts.To = optional.NewInt64(to)
	}
	if page != 0 {
		opts.Page = optional.NewInt32(page)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.MarginAPI.ListMarginAccountBook(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/margin/account_book", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{r.Id, r.Time, r.Currency, r.CurrencyPair, r.Change, r.Balance}
	}
	return p.Table([]string{"ID", "Time", "Currency", "Pair", "Change", "Balance"}, rows)
}

func runMarginFunding(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var opts *gateapi.ListFundingAccountsOpts
	if currency != "" {
		opts = &gateapi.ListFundingAccountsOpts{
			Currency: optional.NewString(currency),
		}
	}

	result, httpResp, err := c.MarginAPI.ListFundingAccounts(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/margin/funding_accounts", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, a := range result {
		rows[i] = []string{a.Currency, a.Available, a.Locked, a.Lent, a.TotalLent}
	}
	return p.Table([]string{"Currency", "Available", "Locked", "Lent", "Total Lent"}, rows)
}

func runMarginTransferable(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	pair, _ := cmd.Flags().GetString("pair")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var opts *gateapi.GetMarginTransferableOpts
	if pair != "" {
		opts = &gateapi.GetMarginTransferableOpts{
			CurrencyPair: optional.NewString(pair),
		}
	}

	result, httpResp, err := c.MarginAPI.GetMarginTransferable(c.Context(), currency, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/margin/transferable", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"Currency", "Pair", "Amount"},
		[][]string{{result.Currency, result.CurrencyPair, result.Amount}},
	)
}

func runMarginUserInfo(cmd *cobra.Command, args []string) error {
	pair, _ := cmd.Flags().GetString("pair")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var opts *gateapi.ListMarginUserAccountOpts
	if pair != "" {
		opts = &gateapi.ListMarginUserAccountOpts{
			CurrencyPair: optional.NewString(pair),
		}
	}

	result, httpResp, err := c.MarginAPI.ListMarginUserAccount(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/margin/user_accounts", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, a := range result {
		rows[i] = []string{a.CurrencyPair, a.Leverage, a.Mmr, fmt.Sprintf("%v", a.Locked)}
	}
	return p.Table([]string{"Pair", "Leverage", "MMR", "Locked"}, rows)
}
