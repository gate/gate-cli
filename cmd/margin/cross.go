package margin

import (
	"fmt"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

var crossCmd = &cobra.Command{
	Use:   "cross",
	Short: "Cross margin commands",
}

func init() {
	loansCmd := &cobra.Command{
		Use:   "loans",
		Short: "List cross margin loans",
		RunE:  runCrossLoans,
	}
	loansCmd.Flags().Int32("status", 2, "Loan status (default 2: borrowed)")
	loansCmd.Flags().String("currency", "", "Filter by currency")
	loansCmd.Flags().Int32("limit", 0, "Number of records to return")
	loansCmd.Flags().Int32("offset", 0, "Record offset for pagination")

	repaymentsCmd := &cobra.Command{
		Use:   "repayments",
		Short: "List cross margin repayments",
		RunE:  runCrossRepayments,
	}
	repaymentsCmd.Flags().String("currency", "", "Filter by currency")
	repaymentsCmd.Flags().Int32("limit", 0, "Number of records to return")
	repaymentsCmd.Flags().Int32("offset", 0, "Record offset for pagination")

	crossCmd.AddCommand(loansCmd, repaymentsCmd)
	Cmd.AddCommand(crossCmd)
}

func runCrossLoans(cmd *cobra.Command, args []string) error {
	status, _ := cmd.Flags().GetInt32("status")
	currency, _ := cmd.Flags().GetString("currency")
	limit, _ := cmd.Flags().GetInt32("limit")
	offset, _ := cmd.Flags().GetInt32("offset")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListCrossMarginLoansOpts{}
	if currency != "" {
		opts.Currency = optional.NewString(currency)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if offset != 0 {
		opts.Offset = optional.NewInt32(offset)
	}

	result, httpResp, err := c.MarginAPI.ListCrossMarginLoans(c.Context(), status, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/margin/cross/loans", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, l := range result {
		rows[i] = []string{l.Id, l.Currency, l.Amount, fmt.Sprintf("%d", l.Status), fmt.Sprintf("%d", l.CreateTime)}
	}
	return p.Table([]string{"ID", "Currency", "Amount", "Status", "Create Time"}, rows)
}

func runCrossRepayments(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	limit, _ := cmd.Flags().GetInt32("limit")
	offset, _ := cmd.Flags().GetInt32("offset")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListCrossMarginRepaymentsOpts{}
	if currency != "" {
		opts.Currency = optional.NewString(currency)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if offset != 0 {
		opts.Offset = optional.NewInt32(offset)
	}

	result, httpResp, err := c.MarginAPI.ListCrossMarginRepayments(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/margin/cross/repayments", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{r.Id, r.LoanId, r.Currency, r.Principal, r.Interest, fmt.Sprintf("%d", r.CreateTime)}
	}
	return p.Table([]string{"ID", "Loan ID", "Currency", "Principal", "Interest", "Create Time"}, rows)
}
