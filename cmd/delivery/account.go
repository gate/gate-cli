package delivery

import (
	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Delivery futures account commands",
}

func init() {
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get delivery futures account information",
		RunE:  runDeliveryAccount,
	}
	getCmd.Flags().String("settle", "usdt", "Settlement currency")

	bookCmd := &cobra.Command{
		Use:   "book",
		Short: "List delivery futures account change history",
		RunE:  runDeliveryAccountBook,
	}
	bookCmd.Flags().String("settle", "usdt", "Settlement currency")
	bookCmd.Flags().Int32("limit", 0, "Number of records to return")
	bookCmd.Flags().String("type", "", "Change type filter")

	accountCmd.AddCommand(getCmd, bookCmd)
	Cmd.AddCommand(accountCmd)
}

func runDeliveryAccount(cmd *cobra.Command, args []string) error {
	settle := getSettle(cmd)
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.DeliveryAPI.ListDeliveryAccounts(c.Context(), settle)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/delivery/"+settle+"/accounts", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"Total", "Available", "Order Margin", "Unrealised PnL", "Currency"},
		[][]string{{result.Total, result.Available, result.OrderMargin, result.UnrealisedPnl, result.Currency}},
	)
}

func runDeliveryAccountBook(cmd *cobra.Command, args []string) error {
	settle := getSettle(cmd)
	limit, _ := cmd.Flags().GetInt32("limit")
	typeFl, _ := cmd.Flags().GetString("type")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListDeliveryAccountBookOpts{}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if typeFl != "" {
		opts.Type_ = optional.NewString(typeFl)
	}

	result, httpResp, err := c.DeliveryAPI.ListDeliveryAccountBook(c.Context(), settle, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/delivery/"+settle+"/account_book", ""))
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
