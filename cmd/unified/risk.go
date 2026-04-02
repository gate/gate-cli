package unified

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

var riskCmd = &cobra.Command{
	Use:   "risk",
	Short: "Risk units and portfolio margin calculator",
}

func init() {
	unitsCmd := &cobra.Command{
		Use:   "units",
		Short: "Get risk units for unified account",
		RunE:  runRiskUnits,
	}

	calculateCmd := &cobra.Command{
		Use:   "calculate",
		Short: "Calculate portfolio margin from JSON input (via stdin or --input)",
		Long: `Calculate portfolio margin. Provide JSON input via --input flag.
Example: gate-cli unified risk calculate --input '{"spot_balances":[...]}'`,
		RunE: runRiskCalculate,
	}
	calculateCmd.Flags().String("input", "", "Portfolio input as JSON string (required)")
	calculateCmd.MarkFlagRequired("input")

	riskCmd.AddCommand(unitsCmd, calculateCmd)
	Cmd.AddCommand(riskCmd)
}

func runRiskUnits(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.UnifiedAPI.GetUnifiedRiskUnits(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/unified/risk_units", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result.RiskUnits))
	for i, ru := range result.RiskUnits {
		rows[i] = []string{
			ru.Symbol, ru.MaintainMargin, ru.InitialMargin,
			ru.Delta, ru.Gamma, ru.Theta, ru.Vega,
		}
	}
	_ = p.Table(
		[]string{"User ID", "Spot Hedge"},
		[][]string{{fmt.Sprintf("%d", result.UserId), fmt.Sprintf("%v", result.SpotHedge)}},
	)
	return p.Table([]string{"Symbol", "Maint Margin", "Init Margin", "Delta", "Gamma", "Theta", "Vega"}, rows)
}

func runRiskCalculate(cmd *cobra.Command, args []string) error {
	inputStr, _ := cmd.Flags().GetString("input")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.UnifiedPortfolioInput
	if err := json.Unmarshal([]byte(inputStr), &body); err != nil {
		return fmt.Errorf("invalid JSON input: %w", err)
	}

	result, httpResp, err := c.UnifiedAPI.CalculatePortfolioMargin(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/unified/portfolio_calculator", ""))
		return nil
	}
	// Complex nested output, prefer JSON-like print for both formats
	return p.Print(result)
}
