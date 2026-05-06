package spot

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
	Short: "Spot account commands",
}

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all spot account balances (non-zero by default)",
		RunE:  runSpotAccountList,
	}
	listCmd.Flags().Bool("all", false, "Show all currencies including zero balances")

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get balance for a specific currency",
		RunE:  runSpotAccountGet,
	}
	getCmd.Flags().String("currency", "", "Currency symbol, e.g. BTC (required)")
	getCmd.MarkFlagRequired("currency")

	feeCmd := &cobra.Command{
		Use:   "fee",
		Short: "Get personal trading fee rates",
		RunE:  runSpotAccountFee,
	}
	feeCmd.Flags().String("pair", "", "Specify currency pair for more accurate fee settings")

	batchFeeCmd := &cobra.Command{
		Use:   "batch-fee",
		Short: "Get trading fee rates for multiple currency pairs",
		RunE:  runSpotAccountBatchFee,
	}
	batchFeeCmd.Flags().String("pairs", "", "Comma-separated currency pairs, e.g. BTC_USDT,ETH_USDT (required)")
	batchFeeCmd.MarkFlagRequired("pairs")

	bookCmd := &cobra.Command{
		Use:   "book",
		Short: "List spot account transaction history",
		RunE:  runSpotAccountBook,
	}
	bookCmd.Flags().String("currency", "", "Filter by currency name")
	bookCmd.Flags().Int64("from", 0, "Start Unix timestamp (ms)")
	bookCmd.Flags().Int64("to", 0, "End Unix timestamp (ms)")
	bookCmd.Flags().Int32("limit", 0, "Number of records to return")

	accountCmd.AddCommand(listCmd, getCmd, feeCmd, batchFeeCmd, bookCmd)
	Cmd.AddCommand(accountCmd)
}

func runSpotAccountList(cmd *cobra.Command, args []string) error {
	showAll, _ := cmd.Flags().GetBool("all")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	accounts, httpResp, err := c.SpotAPI.ListSpotAccounts(c.Context(), nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/spot/accounts", ""))
		return nil
	}

	if p.IsJSON() {
		if showAll {
			return p.Print(accounts)
		}
		filtered := make([]gateapi.SpotAccount, 0)
		for _, a := range accounts {
			if a.Available != "0" || a.Locked != "0" {
				filtered = append(filtered, a)
			}
		}
		return p.Print(filtered)
	}

	rows := make([][]string, 0, len(accounts))
	for _, a := range accounts {
		if showAll || a.Available != "0" || a.Locked != "0" {
			rows = append(rows, []string{a.Currency, a.Available, a.Locked})
		}
	}
	return p.Table([]string{"Currency", "Available", "Locked"}, rows)
}

func runSpotAccountGet(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	accounts, httpResp, err := c.SpotAPI.ListSpotAccounts(c.Context(), &gateapi.ListSpotAccountsOpts{
		Currency: optional.NewString(currency),
	})
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/spot/accounts", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(accounts)
	}
	rows := make([][]string, len(accounts))
	for i, a := range accounts {
		rows[i] = []string{a.Currency, a.Available, a.Locked}
	}
	return p.Table([]string{"Currency", "Available", "Locked"}, rows)
}

func runSpotAccountFee(cmd *cobra.Command, args []string) error {
	pair, _ := cmd.Flags().GetString("pair")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.GetFeeOpts{}
	if pair != "" {
		opts.CurrencyPair = optional.NewString(pair)
	}

	result, httpResp, err := c.SpotAPI.GetFee(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/spot/fee", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"User ID", "Taker Fee", "Maker Fee", "GT Taker", "GT Maker"},
		[][]string{{fmt.Sprintf("%d", result.UserId), result.TakerFee, result.MakerFee, result.GtTakerFee, result.GtMakerFee}},
	)
}

func runSpotAccountBatchFee(cmd *cobra.Command, args []string) error {
	pairs, _ := cmd.Flags().GetString("pairs")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.SpotAPI.GetBatchSpotFee(c.Context(), pairs)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/spot/batch_fee", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, 0, len(result))
	for pair, fee := range result {
		rows = append(rows, []string{pair, fee.TakerFee, fee.MakerFee})
	}
	return p.Table([]string{"Pair", "Taker Fee", "Maker Fee"}, rows)
}

func runSpotAccountBook(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	from, _ := cmd.Flags().GetInt64("from")
	to, _ := cmd.Flags().GetInt64("to")
	limit, _ := cmd.Flags().GetInt32("limit")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListSpotAccountBookOpts{}
	if currency != "" {
		opts.Currency = optional.NewString(currency)
	}
	if from != 0 {
		opts.From = optional.NewInt64(from)
	}
	if to != 0 {
		opts.To = optional.NewInt64(to)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.SpotAPI.ListSpotAccountBook(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/spot/account_book", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{r.Id, fmt.Sprintf("%d", r.Time), r.Currency, r.Change, r.Balance, r.Type, r.Code}
	}
	// SDK v7.2.78 marks Type as deprecated; Code is the authoritative
	// account-change identifier. Keep Type for backward visibility while
	// surfacing Code so downstream tooling can migrate.
	return p.Table([]string{"ID", "Time(ms)", "Currency", "Change", "Balance", "Type", "Code"}, rows)
}
