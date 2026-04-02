package margin

import (
	"fmt"
	"strings"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

var uniCmd = &cobra.Command{
	Use:   "uni",
	Short: "Unified margin (Portfolio Margin) commands",
}

func init() {
	pairsCmd := &cobra.Command{
		Use:   "pairs",
		Short: "List all uni currency pairs (public)",
		RunE:  runUniPairs,
	}

	pairCmd := &cobra.Command{
		Use:   "pair",
		Short: "Get uni currency pair detail (public)",
		RunE:  runUniPair,
	}
	pairCmd.Flags().String("pair", "", "Currency pair, e.g. BTC_USDT (required)")
	pairCmd.MarkFlagRequired("pair")

	estimateRateCmd := &cobra.Command{
		Use:   "estimate-rate",
		Short: "Get estimated interest rates",
		RunE:  runUniEstimateRate,
	}
	estimateRateCmd.Flags().String("currencies", "", "Comma-separated currency list, e.g. BTC,ETH (required)")
	estimateRateCmd.MarkFlagRequired("currencies")

	loansCmd := &cobra.Command{
		Use:   "loans",
		Short: "List uni loans",
		RunE:  runUniLoans,
	}
	loansCmd.Flags().String("pair", "", "Filter by currency pair")
	loansCmd.Flags().String("currency", "", "Filter by currency")
	loansCmd.Flags().Int32("page", 0, "Page number")
	loansCmd.Flags().Int32("limit", 0, "Number of records to return")

	lendCmd := &cobra.Command{
		Use:   "lend",
		Short: "Create a uni loan (borrow)",
		RunE:  runUniLend,
	}
	lendCmd.Flags().String("currency", "", "Currency to borrow (required)")
	lendCmd.MarkFlagRequired("currency")
	lendCmd.Flags().String("amount", "", "Borrow amount (required)")
	lendCmd.MarkFlagRequired("amount")
	lendCmd.Flags().String("pair", "", "Currency pair (required)")
	lendCmd.MarkFlagRequired("pair")
	lendCmd.Flags().String("type", "margin", "Loan type (default: margin)")

	loanRecordsCmd := &cobra.Command{
		Use:   "loan-records",
		Short: "List uni loan records",
		RunE:  runUniLoanRecords,
	}
	loanRecordsCmd.Flags().String("type", "", "Filter by type: borrow or repay")
	loanRecordsCmd.Flags().String("currency", "", "Filter by currency")
	loanRecordsCmd.Flags().String("pair", "", "Filter by currency pair")
	loanRecordsCmd.Flags().Int32("page", 0, "Page number")
	loanRecordsCmd.Flags().Int32("limit", 0, "Number of records to return")

	interestRecordsCmd := &cobra.Command{
		Use:   "interest-records",
		Short: "List uni loan interest records",
		RunE:  runUniInterestRecords,
	}
	interestRecordsCmd.Flags().String("pair", "", "Filter by currency pair")
	interestRecordsCmd.Flags().String("currency", "", "Filter by currency")
	interestRecordsCmd.Flags().Int32("page", 0, "Page number")
	interestRecordsCmd.Flags().Int32("limit", 0, "Number of records to return")
	interestRecordsCmd.Flags().Int64("from", 0, "Start Unix timestamp")
	interestRecordsCmd.Flags().Int64("to", 0, "End Unix timestamp")

	borrowableCmd := &cobra.Command{
		Use:   "borrowable",
		Short: "Get maximum borrowable amount",
		RunE:  runUniBorrowable,
	}
	borrowableCmd.Flags().String("currency", "", "Currency name (required)")
	borrowableCmd.MarkFlagRequired("currency")
	borrowableCmd.Flags().String("pair", "", "Currency pair (required)")
	borrowableCmd.MarkFlagRequired("pair")

	uniCmd.AddCommand(pairsCmd, pairCmd, estimateRateCmd, loansCmd, lendCmd,
		loanRecordsCmd, interestRecordsCmd, borrowableCmd)
	Cmd.AddCommand(uniCmd)
}

func runUniPairs(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.MarginUniAPI.ListUniCurrencyPairs(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/margin/uni/currency_pairs", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, cp := range result {
		rows[i] = []string{cp.CurrencyPair, cp.Leverage, cp.BaseMinBorrowAmount, cp.QuoteMinBorrowAmount}
	}
	return p.Table([]string{"Pair", "Leverage", "Base Min Borrow", "Quote Min Borrow"}, rows)
}

func runUniPair(cmd *cobra.Command, args []string) error {
	pair, _ := cmd.Flags().GetString("pair")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.MarginUniAPI.GetUniCurrencyPair(c.Context(), pair)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/margin/uni/currency_pairs/"+pair, ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"Pair", "Leverage", "Base Min Borrow", "Quote Min Borrow"},
		[][]string{{result.CurrencyPair, result.Leverage, result.BaseMinBorrowAmount, result.QuoteMinBorrowAmount}},
	)
}

func runUniEstimateRate(cmd *cobra.Command, args []string) error {
	currenciesStr, _ := cmd.Flags().GetString("currencies")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	currencies := strings.Split(currenciesStr, ",")

	result, httpResp, err := c.MarginUniAPI.GetMarginUniEstimateRate(c.Context(), currencies)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/margin/uni/estimate_rate", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, 0, len(result))
	for cur, rate := range result {
		rows = append(rows, []string{cur, rate})
	}
	return p.Table([]string{"Currency", "Rate"}, rows)
}

func runUniLoans(cmd *cobra.Command, args []string) error {
	pair, _ := cmd.Flags().GetString("pair")
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

	opts := &gateapi.ListUniLoansOpts{}
	if pair != "" {
		opts.CurrencyPair = optional.NewString(pair)
	}
	if currency != "" {
		opts.Currency = optional.NewString(currency)
	}
	if page != 0 {
		opts.Page = optional.NewInt32(page)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.MarginUniAPI.ListUniLoans(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/margin/uni/loans", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, l := range result {
		rows[i] = []string{l.Currency, l.CurrencyPair, l.Amount, l.Type, fmt.Sprintf("%d", l.CreateTime)}
	}
	return p.Table([]string{"Currency", "Pair", "Amount", "Type", "Create Time"}, rows)
}

func runUniLend(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	amount, _ := cmd.Flags().GetString("amount")
	pair, _ := cmd.Flags().GetString("pair")
	typ, _ := cmd.Flags().GetString("type")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body := gateapi.CreateUniLoan{
		Currency:     currency,
		Amount:       amount,
		CurrencyPair: pair,
		Type:         typ,
	}

	httpResp, err := c.MarginUniAPI.CreateUniLoan(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/margin/uni/loans", ""))
		return nil
	}
	return p.Print(map[string]string{
		"message":       "loan created successfully",
		"currency":      currency,
		"amount":        amount,
		"currency_pair": pair,
	})
}

func runUniLoanRecords(cmd *cobra.Command, args []string) error {
	typ, _ := cmd.Flags().GetString("type")
	currency, _ := cmd.Flags().GetString("currency")
	pair, _ := cmd.Flags().GetString("pair")
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

	opts := &gateapi.ListUniLoanRecordsOpts{}
	if typ != "" {
		opts.Type_ = optional.NewString(typ)
	}
	if currency != "" {
		opts.Currency = optional.NewString(currency)
	}
	if pair != "" {
		opts.CurrencyPair = optional.NewString(pair)
	}
	if page != 0 {
		opts.Page = optional.NewInt32(page)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.MarginUniAPI.ListUniLoanRecords(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/margin/uni/loan_records", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{r.Type, r.Currency, r.CurrencyPair, r.Amount, fmt.Sprintf("%d", r.CreateTime)}
	}
	return p.Table([]string{"Type", "Currency", "Pair", "Amount", "Create Time"}, rows)
}

func runUniInterestRecords(cmd *cobra.Command, args []string) error {
	pair, _ := cmd.Flags().GetString("pair")
	currency, _ := cmd.Flags().GetString("currency")
	page, _ := cmd.Flags().GetInt32("page")
	limit, _ := cmd.Flags().GetInt32("limit")
	from, _ := cmd.Flags().GetInt64("from")
	to, _ := cmd.Flags().GetInt64("to")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListUniLoanInterestRecordsOpts{}
	if pair != "" {
		opts.CurrencyPair = optional.NewString(pair)
	}
	if currency != "" {
		opts.Currency = optional.NewString(currency)
	}
	if page != 0 {
		opts.Page = optional.NewInt32(page)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if from != 0 {
		opts.From = optional.NewInt64(from)
	}
	if to != 0 {
		opts.To = optional.NewInt64(to)
	}

	result, httpResp, err := c.MarginUniAPI.ListUniLoanInterestRecords(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/margin/uni/interest_records", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{r.Currency, r.CurrencyPair, r.ActualRate, r.Interest, fmt.Sprintf("%d", r.CreateTime)}
	}
	return p.Table([]string{"Currency", "Pair", "Rate", "Interest", "Create Time"}, rows)
}

func runUniBorrowable(cmd *cobra.Command, args []string) error {
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

	result, httpResp, err := c.MarginUniAPI.GetUniBorrowable(c.Context(), currency, pair)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/margin/uni/borrowable", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"Currency", "Pair", "Borrowable"},
		[][]string{{result.Currency, result.CurrencyPair, result.Borrowable}},
	)
}
