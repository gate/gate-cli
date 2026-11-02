package unified

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
	Short: "Unified account balance and currency commands",
}

func init() {
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get unified account balances",
		RunE:  runUnifiedAccountGet,
	}
	getCmd.Flags().String("currency", "", "Filter by currency")

	currenciesCmd := &cobra.Command{
		Use:   "currencies",
		Short: "List loan currencies supported by unified account",
		RunE:  runUnifiedAccountCurrencies,
	}

	accountCmd.AddCommand(getCmd, currenciesCmd)
	Cmd.AddCommand(accountCmd)
}

func runUnifiedAccountGet(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var opts *gateapi.ListUnifiedAccountsOpts
	if currency != "" {
		opts = &gateapi.ListUnifiedAccountsOpts{
			Currency: optional.NewString(currency),
		}
	}

	result, httpResp, err := c.UnifiedAPI.ListUnifiedAccounts(c.Context(), opts)
	if err != nil {
		// 服务端拒绝必须传播为命令失败，避免Agentic鉴权错误被自动化当作执行成功。
		return client.CommandError(err, httpResp, "GET", "/api/v4/unified/accounts", "")
	}

	if p.IsJSON() {
		return p.Print(result)
	}

	rows := make([][]string, 0, len(result.Balances))
	for cur, b := range result.Balances {
		rows = append(rows, []string{cur, b.Available, b.Freeze, b.Borrowed, b.Equity})
	}
	header := []string{"Currency", "Available", "Freeze", "Borrowed", "Equity"}
	if len(rows) == 0 {
		return p.Table(header, [][]string{{"(none)", "", "", "", ""}})
	}

	// Print summary row first
	summary := [][]string{{
		fmt.Sprintf("Mode=%s", result.Mode),
		fmt.Sprintf("Total=%s", result.UnifiedAccountTotal),
		fmt.Sprintf("Margin=%s", result.TotalMarginBalance),
		fmt.Sprintf("AvailMargin=%s", result.TotalAvailableMargin),
		"",
	}}
	_ = p.Table([]string{"Mode", "Total(USD)", "Margin Balance", "Available Margin", ""}, summary)
	return p.Table(header, rows)
}

func runUnifiedAccountCurrencies(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.UnifiedAPI.ListUnifiedCurrencies(c.Context(), nil)
	if err != nil {
		return client.CommandError(err, httpResp, "GET", "/api/v4/unified/currencies", "")
	}
	if p.IsJSON() {
		return p.Print(result)
	}

	rows := make([][]string, len(result))
	for i, cur := range result {
		rows[i] = []string{cur.Name, cur.Prec, cur.MinBorrowAmount, cur.LoanStatus}
	}
	return p.Table([]string{"Name", "Precision", "Min Borrow", "Loan Status"}, rows)
}
