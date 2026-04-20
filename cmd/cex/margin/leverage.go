package margin

import (
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

var leverageCmd = &cobra.Command{
	Use:   "leverage",
	Short: "Margin leverage commands",
}

func init() {
	setCmd := &cobra.Command{
		Use:   "set",
		Short: "Set user market leverage",
		RunE:  runLeverageSet,
	}
	setCmd.Flags().String("pair", "", "Currency pair (required)")
	setCmd.MarkFlagRequired("pair")
	setCmd.Flags().String("leverage", "", "Leverage value (required)")
	setCmd.MarkFlagRequired("leverage")

	leverageCmd.AddCommand(setCmd)
	Cmd.AddCommand(leverageCmd)
}

func runLeverageSet(cmd *cobra.Command, args []string) error {
	pair, _ := cmd.Flags().GetString("pair")
	leverage, _ := cmd.Flags().GetString("leverage")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body := gateapi.MarginMarketLeverage{
		CurrencyPair: pair,
		Leverage:     leverage,
	}

	httpResp, err := c.MarginAPI.SetUserMarketLeverage(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/margin/leverage/user_market_setting", ""))
		return nil
	}
	return p.Print(map[string]string{
		"message":       "leverage set successfully",
		"currency_pair": pair,
		"leverage":      leverage,
	})
}
