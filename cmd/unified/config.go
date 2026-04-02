package unified

import (
	"fmt"
	"strings"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Unified account configuration commands",
}

func init() {
	collateralCmd := &cobra.Command{
		Use:   "collateral",
		Short: "Set collateral mode for unified account",
		RunE:  runConfigCollateral,
	}
	collateralCmd.Flags().Int32("type", 0, "Collateral type: 0=all, 1=custom (required)")
	collateralCmd.Flags().String("enable", "", "Comma-separated currencies to enable as collateral")
	collateralCmd.Flags().String("disable", "", "Comma-separated currencies to disable as collateral")
	collateralCmd.MarkFlagRequired("type")

	discountTiersCmd := &cobra.Command{
		Use:   "discount-tiers",
		Short: "List currency discount tiers (public, no auth)",
		RunE:  runConfigDiscountTiers,
	}

	loanTiersCmd := &cobra.Command{
		Use:   "loan-tiers",
		Short: "List loan margin tiers (public, no auth)",
		RunE:  runConfigLoanTiers,
	}

	leverageConfigCmd := &cobra.Command{
		Use:   "leverage-config",
		Short: "Get leverage config for a currency",
		RunE:  runConfigLeverageConfig,
	}
	leverageConfigCmd.Flags().String("currency", "", "Currency name (required)")
	leverageConfigCmd.MarkFlagRequired("currency")

	leverageGetCmd := &cobra.Command{
		Use:   "leverage-get",
		Short: "Get leverage setting for currencies",
		RunE:  runConfigLeverageGet,
	}
	leverageGetCmd.Flags().String("currency", "", "Filter by currency")

	leverageSetCmd := &cobra.Command{
		Use:   "leverage-set",
		Short: "Set leverage for a currency",
		RunE:  runConfigLeverageSet,
	}
	leverageSetCmd.Flags().String("currency", "", "Currency name (required)")
	leverageSetCmd.Flags().String("leverage", "", "Leverage multiplier (required)")
	leverageSetCmd.MarkFlagRequired("currency")
	leverageSetCmd.MarkFlagRequired("leverage")

	configCmd.AddCommand(collateralCmd, discountTiersCmd, loanTiersCmd,
		leverageConfigCmd, leverageGetCmd, leverageSetCmd)
	Cmd.AddCommand(configCmd)
}

func runConfigCollateral(cmd *cobra.Command, args []string) error {
	colType, _ := cmd.Flags().GetInt32("type")
	enable, _ := cmd.Flags().GetString("enable")
	disable, _ := cmd.Flags().GetString("disable")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body := gateapi.UnifiedCollateralReq{
		CollateralType: colType,
	}
	if enable != "" {
		body.EnableList = strings.Split(enable, ",")
	}
	if disable != "" {
		body.DisableList = strings.Split(disable, ",")
	}

	result, httpResp, err := c.UnifiedAPI.SetUnifiedCollateral(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/unified/collateral", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"Success"},
		[][]string{{fmt.Sprintf("%v", result.IsSuccess)}},
	)
}

func runConfigDiscountTiers(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.UnifiedAPI.ListCurrencyDiscountTiers(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/unified/currency_discount_tiers", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	// Flatten nested tiers into rows
	rows := make([][]string, 0)
	for _, d := range result {
		for _, t := range d.DiscountTiers {
			rows = append(rows, []string{d.Currency, t.Tier, t.Discount, t.LowerLimit, t.UpperLimit, t.Leverage})
		}
	}
	return p.Table([]string{"Currency", "Tier", "Discount", "Lower", "Upper", "Leverage"}, rows)
}

func runConfigLoanTiers(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.UnifiedAPI.ListLoanMarginTiers(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/unified/loan_margin_tiers", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, 0)
	for _, m := range result {
		for _, t := range m.MarginTiers {
			rows = append(rows, []string{m.Currency, t.Tier, t.MarginRate, t.LowerLimit, t.UpperLimit, t.Leverage})
		}
	}
	return p.Table([]string{"Currency", "Tier", "Margin Rate", "Lower", "Upper", "Leverage"}, rows)
}

func runConfigLeverageConfig(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.UnifiedAPI.GetUserLeverageCurrencyConfig(c.Context(), currency)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/unified/leverage/user_currency_config/"+currency, ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"Current", "Min", "Max", "Debit", "Available Margin", "Borrowable"},
		[][]string{{
			result.CurrentLeverage, result.MinLeverage, result.MaxLeverage,
			result.Debit, result.AvailableMargin, result.Borrowable,
		}},
	)
}

func runConfigLeverageGet(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var opts *gateapi.GetUserLeverageCurrencySettingOpts
	if currency != "" {
		opts = &gateapi.GetUserLeverageCurrencySettingOpts{
			Currency: optional.NewString(currency),
		}
	}

	result, httpResp, err := c.UnifiedAPI.GetUserLeverageCurrencySetting(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/unified/leverage/user_currency_setting", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, s := range result {
		rows[i] = []string{s.Currency, s.Leverage}
	}
	return p.Table([]string{"Currency", "Leverage"}, rows)
}

func runConfigLeverageSet(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	leverage, _ := cmd.Flags().GetString("leverage")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body := gateapi.UnifiedLeverageSetting{
		Currency: currency,
		Leverage: leverage,
	}

	httpResp, err := c.UnifiedAPI.SetUserLeverageCurrencySetting(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/unified/leverage/user_currency_setting", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(map[string]string{"status": "ok", "currency": currency, "leverage": leverage})
	}
	return p.Table(
		[]string{"Status", "Currency", "Leverage"},
		[][]string{{"ok", currency, leverage}},
	)
}
