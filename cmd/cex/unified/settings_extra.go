package unified

import (
	"fmt"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/spf13/cobra"
)

// SDK-backed commands share the existing transport and authentication boundary.
func init() {
	unifiedDeltaNeutralGetCmd := &cobra.Command{Use: "delta-neutral-get", Short: "Get Delta-neutral strategy mode", Args: cobra.NoArgs, RunE: runUnifiedDeltaNeutralGet}
	configCmd.AddCommand(unifiedDeltaNeutralGetCmd)
	unifiedDeltaNeutralSetCmd := &cobra.Command{Use: "delta-neutral-set", Short: "Enable or disable Delta-neutral strategy mode", Args: cobra.NoArgs, RunE: runUnifiedDeltaNeutralSet}
	unifiedDeltaNeutralSetCmd.Flags().Bool("enabled", false, "Explicit true or false")
	cmdutil.MustRequireFlags(unifiedDeltaNeutralSetCmd, "enabled")
	configCmd.AddCommand(unifiedDeltaNeutralSetCmd)
	unifiedQuickEstimateCmd := &cobra.Command{Use: "quick-estimate", Short: "Estimate quick repayment", Args: cobra.NoArgs, RunE: runUnifiedQuickEstimate}
	loanCmd.AddCommand(unifiedQuickEstimateCmd)
	unifiedQuickRepayCmd := &cobra.Command{Use: "quick-repay", Short: "Repay liabilities using selected available currencies", Args: cobra.NoArgs, RunE: runUnifiedQuickRepay}
	unifiedQuickRepayCmd.Flags().StringSlice("debt-currencies", nil, "Comma-separated liability currencies")
	cmdutil.MustRequireFlags(unifiedQuickRepayCmd, "debt-currencies")
	unifiedQuickRepayCmd.Flags().StringSlice("available-currencies", nil, "Comma-separated repayment currencies")
	cmdutil.MustRequireFlags(unifiedQuickRepayCmd, "available-currencies")
	loanCmd.AddCommand(unifiedQuickRepayCmd)
	unifiedLeverageSetAllCmd := &cobra.Command{Use: "leverage-set-all", Short: "Set the account leverage; currency limits may cap the requested value", Args: cobra.NoArgs, RunE: runUnifiedLeverageSetAll}
	unifiedLeverageSetAllCmd.Flags().String("leverage", "", "Positive leverage multiplier as a decimal string")
	cmdutil.MustRequireFlags(unifiedLeverageSetAllCmd, "leverage")
	configCmd.AddCommand(unifiedLeverageSetAllCmd)
}

func runUnifiedDeltaNeutralGet(cmd *cobra.Command, args []string) error {

	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}
	result, resp, err := c.UnifiedAPI.GetUnifiedDeltaNeutral(c.Context())
	if err != nil {
		return client.CommandError(err, resp, "GET", "/api/v4/unified/delta_neutral", "")
	}
	return cmdutil.GetPrinter(cmd).Print(result)
}

func runUnifiedDeltaNeutralSet(cmd *cobra.Command, args []string) error {
	enabled, _ := cmd.Flags().GetBool("enabled")

	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}
	result, resp, err := c.UnifiedAPI.SetUnifiedDeltaNeutral(c.Context(), gateapi.DeltaNeutralEnabled{Enabled: enabled})
	if err != nil {
		return client.CommandError(err, resp, "POST", "/api/v4/unified/delta_neutral", "")
	}
	return cmdutil.GetPrinter(cmd).Print(result)
}

func runUnifiedQuickEstimate(cmd *cobra.Command, args []string) error {

	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}
	result, resp, err := c.UnifiedAPI.GetEstimatedQuickRepayment(c.Context())
	if err != nil {
		return client.CommandError(err, resp, "GET", "/api/v4/unified/estimated_quick_repayment", "")
	}
	return cmdutil.GetPrinter(cmd).Print(result)
}

func runUnifiedQuickRepay(cmd *cobra.Command, args []string) error {
	debtCurrencies, _ := cmd.Flags().GetStringSlice("debt-currencies")
	availableCurrencies, _ := cmd.Flags().GetStringSlice("available-currencies")
	for _, currencies := range [][]string{debtCurrencies, availableCurrencies} {
		if len(currencies) == 0 || len(currencies) > 100 {
			return fmt.Errorf("currency lists must contain 1 to 100 entries")
		}
		seen := map[string]bool{}
		for _, currency := range currencies {
			if err := cmdutil.TradingToken("currency", currency); err != nil {
				return err
			}
			if seen[currency] {
				return fmt.Errorf("duplicate currency: %s", currency)
			}
			seen[currency] = true
		}
	}

	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}
	result, resp, err := c.UnifiedAPI.CreateQuickRepayment(c.Context(), gateapi.QuickRepaymentRequest{DebtCurrencies: debtCurrencies, AvailableCurrencies: availableCurrencies})
	if err != nil {
		return client.CommandError(err, resp, "POST", "/api/v4/unified/quick_repayment", "")
	}
	return cmdutil.GetPrinter(cmd).Print(result)
}

func runUnifiedLeverageSetAll(cmd *cobra.Command, args []string) error {
	leverage, _ := cmd.Flags().GetString("leverage")
	if _, err := cmdutil.TradingDecimal("--leverage", leverage, true); err != nil {
		return err
	}

	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}
	result, resp, err := c.UnifiedAPI.SetUserLeverage(c.Context(), gateapi.UserLeverageSetting{Leverage: leverage})
	if err != nil {
		return client.CommandError(err, resp, "POST", "/api/v4/unified/leverage/user_setting", "")
	}
	// This endpoint reports per-currency failures with HTTP 200 and no rollback.
	// A zero exit status must not imply that all leverage settings were updated.
	if result == nil {
		return fmt.Errorf("missing leverage update failure list; inspect account settings before retrying")
	}
	if len(result) > 0 {
		return fmt.Errorf("leverage update failed for %d currencies: %+v; other currencies may already be updated; inspect settings before retrying", len(result), result)
	}
	return cmdutil.GetPrinter(cmd).Print(result)
}
