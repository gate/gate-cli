package unified

import (
	"fmt"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

var loanCmd = &cobra.Command{
	Use:   "loan",
	Short: "Unified account loan commands",
}

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List current unified loans",
		RunE:  runLoanList,
	}
	listCmd.Flags().String("currency", "", "Filter by currency")
	listCmd.Flags().String("type", "", "Filter by type: platform, margin")
	listCmd.Flags().Int32("page", 0, "Page number")
	listCmd.Flags().Int32("limit", 0, "Max records to return")

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Borrow or repay in unified account",
		RunE:  runLoanCreate,
	}
	createCmd.Flags().String("currency", "", "Currency to borrow/repay (required)")
	createCmd.Flags().String("type", "borrow", "Operation type: borrow, repay")
	createCmd.Flags().String("amount", "", "Amount (required)")
	createCmd.MarkFlagRequired("currency")
	createCmd.MarkFlagRequired("amount")

	recordsCmd := &cobra.Command{
		Use:   "records",
		Short: "List loan records",
		RunE:  runLoanRecords,
	}
	recordsCmd.Flags().String("currency", "", "Filter by currency")
	recordsCmd.Flags().String("type", "", "Filter by type: borrow, repay")
	recordsCmd.Flags().Int32("page", 0, "Page number")
	recordsCmd.Flags().Int32("limit", 0, "Max records to return")

	interestCmd := &cobra.Command{
		Use:   "interest",
		Short: "List loan interest records",
		RunE:  runLoanInterest,
	}
	interestCmd.Flags().String("currency", "", "Filter by currency")
	interestCmd.Flags().String("type", "", "Filter by type")
	interestCmd.Flags().Int64("from", 0, "Start Unix timestamp")
	interestCmd.Flags().Int64("to", 0, "End Unix timestamp")
	interestCmd.Flags().Int32("page", 0, "Page number")
	interestCmd.Flags().Int32("limit", 0, "Max records to return")

	historyRateCmd := &cobra.Command{
		Use:   "history-rate",
		Short: "Get historical lending rates for a currency",
		RunE:  runLoanHistoryRate,
	}
	historyRateCmd.Flags().String("currency", "", "Currency name (required)")
	historyRateCmd.Flags().String("tier", "", "VIP tier for floating rate")
	historyRateCmd.Flags().Int32("page", 0, "Page number")
	historyRateCmd.Flags().Int32("limit", 0, "Max records to return")
	historyRateCmd.MarkFlagRequired("currency")

	loanCmd.AddCommand(listCmd, createCmd, recordsCmd, interestCmd, historyRateCmd)
	Cmd.AddCommand(loanCmd)
}

func runLoanList(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	loanType, _ := cmd.Flags().GetString("type")
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

	opts := &gateapi.ListUnifiedLoansOpts{}
	if currency != "" {
		opts.Currency = optional.NewString(currency)
	}
	if loanType != "" {
		opts.Type_ = optional.NewString(loanType)
	}
	if page != 0 {
		opts.Page = optional.NewInt32(page)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.UnifiedAPI.ListUnifiedLoans(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/unified/loans", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, l := range result {
		rows[i] = []string{l.Currency, l.CurrencyPair, l.Amount, l.Type, fmt.Sprintf("%d", l.CreateTime)}
	}
	return p.Table([]string{"Currency", "Pair", "Amount", "Type", "Created"}, rows)
}

func runLoanCreate(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	loanType, _ := cmd.Flags().GetString("type")
	amount, _ := cmd.Flags().GetString("amount")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body := gateapi.UnifiedLoan{
		Currency: currency,
		Type:     loanType,
		Amount:   amount,
	}

	result, httpResp, err := c.UnifiedAPI.CreateUnifiedLoan(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/unified/loans", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"Transaction ID"},
		[][]string{{fmt.Sprintf("%d", result.TranId)}},
	)
}

func runLoanRecords(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	loanType, _ := cmd.Flags().GetString("type")
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

	opts := &gateapi.ListUnifiedLoanRecordsOpts{}
	if currency != "" {
		opts.Currency = optional.NewString(currency)
	}
	if loanType != "" {
		opts.Type_ = optional.NewString(loanType)
	}
	if page != 0 {
		opts.Page = optional.NewInt32(page)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.UnifiedAPI.ListUnifiedLoanRecords(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/unified/loan_records", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{
			fmt.Sprintf("%d", r.Id), r.Type, r.Currency, r.CurrencyPair,
			r.Amount, fmt.Sprintf("%d", r.CreateTime),
		}
	}
	return p.Table([]string{"ID", "Type", "Currency", "Pair", "Amount", "Created"}, rows)
}

func runLoanInterest(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	loanType, _ := cmd.Flags().GetString("type")
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

	opts := &gateapi.ListUnifiedLoanInterestRecordsOpts{}
	if currency != "" {
		opts.Currency = optional.NewString(currency)
	}
	if loanType != "" {
		opts.Type_ = optional.NewString(loanType)
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

	result, httpResp, err := c.UnifiedAPI.ListUnifiedLoanInterestRecords(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/unified/interest_records", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{
			r.Currency, r.CurrencyPair, r.ActualRate, r.Interest,
			r.Type, fmt.Sprintf("%d", r.CreateTime),
		}
	}
	return p.Table([]string{"Currency", "Pair", "Rate", "Interest", "Type", "Created"}, rows)
}

func runLoanHistoryRate(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	tier, _ := cmd.Flags().GetString("tier")
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

	opts := &gateapi.GetHistoryLoanRateOpts{}
	if tier != "" {
		opts.Tier = optional.NewString(tier)
	}
	if page != 0 {
		opts.Page = optional.NewInt32(page)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.UnifiedAPI.GetHistoryLoanRate(c.Context(), currency, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/unified/history_loan_rate", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result.Rates))
	for i, r := range result.Rates {
		rows[i] = []string{fmt.Sprintf("%d", r.Time), r.Rate}
	}
	header := fmt.Sprintf("Currency: %s | Tier: %s | TierUpRate: %s", result.Currency, result.Tier, result.TierUpRate)
	_ = p.Table([]string{header}, [][]string{})
	return p.Table([]string{"Time", "Rate"}, rows)
}
