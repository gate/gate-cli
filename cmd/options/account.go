package options

import (
	"fmt"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Options account commands",
}

func init() {
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get options account information",
		RunE:  runOptionsAccount,
	}

	bookCmd := &cobra.Command{
		Use:   "book",
		Short: "List options account change history",
		RunE:  runOptionsAccountBook,
	}
	bookCmd.Flags().Int32("limit", 0, "Number of records to return")
	bookCmd.Flags().Int32("offset", 0, "Number of records to skip")
	bookCmd.Flags().Int64("from", 0, "Start Unix timestamp")
	bookCmd.Flags().Int64("to", 0, "End Unix timestamp")
	bookCmd.Flags().String("type", "", "Change type filter")

	settlementsCmd := &cobra.Command{
		Use:   "settlements",
		Short: "List personal settlement history",
		RunE:  runOptionsMySettlements,
	}
	settlementsCmd.Flags().String("underlying", "", "Underlying name (required)")
	settlementsCmd.MarkFlagRequired("underlying")

	accountCmd.AddCommand(getCmd, bookCmd, settlementsCmd)
	Cmd.AddCommand(accountCmd)
}

func runOptionsAccount(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.OptionsAPI.ListOptionsAccount(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/options/accounts", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"User", "Total", "Equity", "Position Value", "Init Margin", "Unrealised PnL"},
		[][]string{{
			fmt.Sprintf("%d", result.User),
			result.Total,
			result.Equity,
			result.PositionValue,
			result.InitMargin,
			result.UnrealisedPnl,
		}},
	)
}

func runOptionsAccountBook(cmd *cobra.Command, args []string) error {
	limit, _ := cmd.Flags().GetInt32("limit")
	offset, _ := cmd.Flags().GetInt32("offset")
	from, _ := cmd.Flags().GetInt64("from")
	to, _ := cmd.Flags().GetInt64("to")
	typeFl, _ := cmd.Flags().GetString("type")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListOptionsAccountBookOpts{}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if offset != 0 {
		opts.Offset = optional.NewInt32(offset)
	}
	if from != 0 {
		opts.From = optional.NewInt64(from)
	}
	if to != 0 {
		opts.To = optional.NewInt64(to)
	}
	if typeFl != "" {
		opts.Type_ = optional.NewString(typeFl)
	}

	result, httpResp, err := c.OptionsAPI.ListOptionsAccountBook(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/options/account_book", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, b := range result {
		rows[i] = []string{b.Type, b.Change, b.Balance, b.Text}
	}
	return p.Table([]string{"Type", "Change", "Balance", "Text"}, rows)
}

func runOptionsMySettlements(cmd *cobra.Command, args []string) error {
	underlying, _ := cmd.Flags().GetString("underlying")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.OptionsAPI.ListMyOptionsSettlements(c.Context(), underlying, nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/options/my_settlements", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, s := range result {
		rows[i] = []string{s.Contract, s.StrikePrice, s.SettlePrice, s.SettleProfit, s.Fee}
	}
	return p.Table([]string{"Contract", "Strike Price", "Settle Price", "Settle Profit", "Fee"}, rows)
}
