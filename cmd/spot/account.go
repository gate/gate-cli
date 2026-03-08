package spot

import (
	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Spot account commands",
}

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all spot account balances (non-zero by default)",
		RunE:  runSpotAccountList,
	}
	listCmd.Flags().Bool("all", false, "Show all currencies including zero balances")

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get balance for a specific currency",
		RunE:  runSpotAccountGet,
	}
	getCmd.Flags().String("currency", "", "Currency symbol, e.g. BTC (required)")
	getCmd.MarkFlagRequired("currency")

	accountCmd.AddCommand(listCmd, getCmd)
	Cmd.AddCommand(accountCmd)
}

func runSpotAccountList(cmd *cobra.Command, args []string) error {
	showAll, _ := cmd.Flags().GetBool("all")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	accounts, httpResp, err := c.SpotAPI.ListSpotAccounts(c.Context(), nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/spot/accounts", ""))
		return nil
	}

	if p.IsJSON() {
		if showAll {
			return p.Print(accounts)
		}
		filtered := make([]gateapi.SpotAccount, 0)
		for _, a := range accounts {
			if a.Available != "0" || a.Locked != "0" {
				filtered = append(filtered, a)
			}
		}
		return p.Print(filtered)
	}

	rows := make([][]string, 0, len(accounts))
	for _, a := range accounts {
		if showAll || a.Available != "0" || a.Locked != "0" {
			rows = append(rows, []string{a.Currency, a.Available, a.Locked})
		}
	}
	return p.Table([]string{"Currency", "Available", "Locked"}, rows)
}

func runSpotAccountGet(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	accounts, httpResp, err := c.SpotAPI.ListSpotAccounts(c.Context(), &gateapi.ListSpotAccountsOpts{
		Currency: optional.NewString(currency),
	})
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/spot/accounts", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(accounts)
	}
	rows := make([][]string, len(accounts))
	for i, a := range accounts {
		rows[i] = []string{a.Currency, a.Available, a.Locked}
	}
	return p.Table([]string{"Currency", "Available", "Locked"}, rows)
}
