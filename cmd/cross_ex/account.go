package crossex

import (
	"encoding/json"
	"fmt"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Cross-exchange account commands",
}

func init() {
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Query account assets",
		RunE:  runAccountGet,
	}
	getCmd.Flags().String("exchange-type", "", "Exchange (BINANCE/OKX/GATE/BYBIT); required in isolated mode")

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Modify account position mode or account mode",
		RunE:  runAccountUpdate,
	}
	updateCmd.Flags().String("json", "", "JSON body for account update request (required)")
	updateCmd.MarkFlagRequired("json")

	bookCmd := &cobra.Command{
		Use:   "book",
		Short: "Query account asset change history",
		RunE:  runAccountBook,
	}
	bookCmd.Flags().Int32("page", 0, "Page number")
	bookCmd.Flags().Int32("limit", 0, "Max records to return")
	bookCmd.Flags().String("coin", "", "Filter by currency")
	bookCmd.Flags().Int32("from", 0, "Start millisecond timestamp")
	bookCmd.Flags().Int32("to", 0, "End millisecond timestamp")

	accountCmd.AddCommand(getCmd, updateCmd, bookCmd)
	Cmd.AddCommand(accountCmd)
}

func runAccountGet(cmd *cobra.Command, args []string) error {
	exchangeType, _ := cmd.Flags().GetString("exchange-type")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var opts *gateapi.GetCrossexAccountOpts
	if exchangeType != "" {
		opts = &gateapi.GetCrossexAccountOpts{
			ExchangeType: optional.NewString(exchangeType),
		}
	}

	result, httpResp, err := c.CrossExAPI.GetCrossexAccount(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/crossex/account", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}

	summary := [][]string{{
		result.AccountMode,
		result.PositionMode,
		result.AvailableMargin,
		result.MarginBalance,
		result.InitialMarginRate,
		result.MaintenanceMarginRate,
	}}
	return p.Table([]string{"Account Mode", "Position Mode", "Avail Margin", "Margin Balance", "Init Margin Rate", "Maint Margin Rate"}, summary)
}

func runAccountUpdate(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.CrossexAccountUpdateRequest
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}

	opts := &gateapi.UpdateCrossexAccountOpts{
		CrossexAccountUpdateRequest: optional.NewInterface(body),
	}
	result, httpResp, err := c.CrossExAPI.UpdateCrossexAccount(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "PUT", "/api/v4/crossex/account", jsonStr))
		return nil
	}
	return p.Print(result)
}

func runAccountBook(cmd *cobra.Command, args []string) error {
	page, _ := cmd.Flags().GetInt32("page")
	limit, _ := cmd.Flags().GetInt32("limit")
	coin, _ := cmd.Flags().GetString("coin")
	from, _ := cmd.Flags().GetInt32("from")
	to, _ := cmd.Flags().GetInt32("to")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListCrossexAccountBookOpts{}
	if page != 0 {
		opts.Page = optional.NewInt32(page)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if coin != "" {
		opts.Coin = optional.NewString(coin)
	}
	if from != 0 {
		opts.From = optional.NewInt32(from)
	}
	if to != 0 {
		opts.To = optional.NewInt32(to)
	}

	result, httpResp, err := c.CrossExAPI.ListCrossexAccountBook(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/crossex/account_book", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{r.Id, r.Type, r.Coin, r.ExchangeType, r.Change, r.Balance, r.CreateTime}
	}
	return p.Table([]string{"ID", "Type", "Coin", "Exchange", "Change", "Balance", "Created"}, rows)
}
