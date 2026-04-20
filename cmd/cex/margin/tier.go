package margin

import (
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
)

var tierCmd = &cobra.Command{
	Use:   "tier",
	Short: "Margin leverage tier commands",
}

func init() {
	userCmd := &cobra.Command{
		Use:   "user",
		Short: "Get user margin leverage tier",
		RunE:  runTierUser,
	}
	userCmd.Flags().String("pair", "", "Currency pair (required)")
	userCmd.MarkFlagRequired("pair")

	marketCmd := &cobra.Command{
		Use:   "market",
		Short: "Get market margin leverage tier (public)",
		RunE:  runTierMarket,
	}
	marketCmd.Flags().String("pair", "", "Currency pair (required)")
	marketCmd.MarkFlagRequired("pair")

	tierCmd.AddCommand(userCmd, marketCmd)
	Cmd.AddCommand(tierCmd)
}

func runTierUser(cmd *cobra.Command, args []string) error {
	pair, _ := cmd.Flags().GetString("pair")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.MarginAPI.GetUserMarginTier(c.Context(), pair)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/margin/user_leverage_tier", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, t := range result {
		rows[i] = []string{t.Leverage, t.UpperLimit, t.Mmr}
	}
	return p.Table([]string{"Leverage", "Upper Limit", "MMR"}, rows)
}

func runTierMarket(cmd *cobra.Command, args []string) error {
	pair, _ := cmd.Flags().GetString("pair")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.MarginAPI.GetMarketMarginTier(c.Context(), pair)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/margin/market_leverage_tier", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, t := range result {
		rows[i] = []string{t.Leverage, t.UpperLimit, t.Mmr}
	}
	return p.Table([]string{"Leverage", "Upper Limit", "MMR"}, rows)
}
