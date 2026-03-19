package futures

import (
	"encoding/json"
	"fmt"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Futures account commands",
}

func init() {
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get futures account details",
		RunE:  runFuturesAccountGet,
	}
	addSettleFlag(getCmd)

	bookCmd := &cobra.Command{
		Use:   "book",
		Short: "List futures account change records",
		RunE:  runFuturesAccountBook,
	}
	bookCmd.Flags().String("contract", "", "Filter by contract name")
	bookCmd.Flags().Int64("from", 0, "Start Unix timestamp")
	bookCmd.Flags().Int64("to", 0, "End Unix timestamp")
	bookCmd.Flags().Int32("limit", 0, "Number of records to return")
	addSettleFlag(bookCmd)

	feeCmd := &cobra.Command{
		Use:   "fee",
		Short: "Get futures trading fee rates",
		RunE:  runFuturesAccountFee,
	}
	feeCmd.Flags().String("contract", "", "Filter by contract name")
	addSettleFlag(feeCmd)

	dualModeCmd := &cobra.Command{
		Use:   "dual-mode",
		Short: "Enable or disable dual position mode",
		RunE:  runFuturesDualMode,
	}
	dualModeCmd.Flags().Bool("enable", false, "Enable dual mode (omit to disable)")
	addSettleFlag(dualModeCmd)

	positionModeCmd := &cobra.Command{
		Use:   "position-mode",
		Short: "Set position mode (classic or cross_margin)",
		RunE:  runFuturesPositionMode,
	}
	positionModeCmd.Flags().String("mode", "", "Position mode (required)")
	positionModeCmd.MarkFlagRequired("mode")
	addSettleFlag(positionModeCmd)

	accountCmd.AddCommand(getCmd, bookCmd, feeCmd, dualModeCmd, positionModeCmd)
	Cmd.AddCommand(accountCmd)
}

func runFuturesAccountGet(cmd *cobra.Command, args []string) error {
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	account, httpResp, err := c.FuturesAPI.ListFuturesAccounts(c.Context(), settle)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/accounts", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(account)
	}
	return p.Table(
		[]string{"Currency", "Total", "Available", "Unrealised PNL", "Order Margin"},
		[][]string{{account.Currency, account.Total, account.Available, account.UnrealisedPnl, account.OrderMargin}},
	)
}

func runFuturesAccountBook(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	from, _ := cmd.Flags().GetInt64("from")
	to, _ := cmd.Flags().GetInt64("to")
	limit, _ := cmd.Flags().GetInt32("limit")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListFuturesAccountBookOpts{}
	if contract != "" {
		opts.Contract = optional.NewString(contract)
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

	result, httpResp, err := c.FuturesAPI.ListFuturesAccountBook(c.Context(), settle, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/account_book", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{fmt.Sprintf("%g", r.Time), r.Change, r.Balance, r.Type, r.Text}
	}
	return p.Table([]string{"Time", "Change", "Balance", "Type", "Text"}, rows)
}

func runFuturesAccountFee(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.GetFuturesFeeOpts{}
	if contract != "" {
		opts.Contract = optional.NewString(contract)
	}

	result, httpResp, err := c.FuturesAPI.GetFuturesFee(c.Context(), settle, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/futures/"+settle+"/fee", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, 0, len(result))
	for ct, fee := range result {
		rows = append(rows, []string{ct, fee.TakerFee, fee.MakerFee})
	}
	return p.Table([]string{"Contract", "Taker Fee", "Maker Fee"}, rows)
}

func runFuturesDualMode(cmd *cobra.Command, args []string) error {
	enable, _ := cmd.Flags().GetBool("enable")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body, _ := json.Marshal(map[string]bool{"dual_mode": enable})
	result, httpResp, err := c.FuturesAPI.SetDualMode(c.Context(), settle, enable)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/futures/"+settle+"/dual_mode", string(body)))
		return nil
	}
	return p.Print(result)
}

func runFuturesPositionMode(cmd *cobra.Command, args []string) error {
	mode, _ := cmd.Flags().GetString("mode")
	settle := cmdutil.GetSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body, _ := json.Marshal(map[string]string{"position_mode": mode})
	result, httpResp, err := c.FuturesAPI.SetPositionMode(c.Context(), settle, mode)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/futures/"+settle+"/position_mode", string(body)))
		return nil
	}
	return p.Print(result)
}
