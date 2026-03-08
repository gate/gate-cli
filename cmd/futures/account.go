package futures

import (
	"github.com/spf13/cobra"

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

	accountCmd.AddCommand(getCmd)
	Cmd.AddCommand(accountCmd)
}

func runFuturesAccountGet(cmd *cobra.Command, args []string) error {
	settle := getSettle(cmd)
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
