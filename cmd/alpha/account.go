package alpha

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
	Short: "Alpha account commands",
}

func init() {
	balancesCmd := &cobra.Command{
		Use:   "balances",
		Short: "List Alpha account balances",
		RunE:  runAlphaBalances,
	}

	bookCmd := &cobra.Command{
		Use:   "book",
		Short: "List Alpha account transaction history",
		RunE:  runAlphaAccountBook,
	}
	bookCmd.Flags().Int64("from", 0, "Start timestamp (Unix seconds, required)")
	bookCmd.Flags().Int64("to", 0, "End timestamp (Unix seconds)")
	bookCmd.Flags().Int32("page", 1, "Page number")
	bookCmd.Flags().Int32("limit", 100, "Maximum number of records")
	bookCmd.MarkFlagRequired("from")

	accountCmd.AddCommand(balancesCmd, bookCmd)
	Cmd.AddCommand(accountCmd)
}

func runAlphaBalances(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.AlphaAPI.ListAlphaAccounts(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/alpha/accounts", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, acc := range result {
		rows[i] = []string{acc.Currency, acc.Available, acc.Locked, acc.Chain, acc.TokenAddress}
	}
	return p.Table([]string{"Currency", "Available", "Locked", "Chain", "Token Address"}, rows)
}

func runAlphaAccountBook(cmd *cobra.Command, args []string) error {
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

	opts := &gateapi.ListAlphaAccountBookOpts{
		Page:  optional.NewInt32(page),
		Limit: optional.NewInt32(limit),
	}
	if to != 0 {
		opts.To = optional.NewInt64(to)
	}

	result, httpResp, err := c.AlphaAPI.ListAlphaAccountBook(c.Context(), from, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/alpha/account_book", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, entry := range result {
		rows[i] = []string{fmt.Sprintf("%d", entry.Id), fmt.Sprintf("%d", entry.Time), entry.Currency, entry.Change, entry.Balance}
	}
	return p.Table([]string{"ID", "Time", "Currency", "Change", "Balance"}, rows)
}
