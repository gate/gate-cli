package bot

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

var martingaleCmd = &cobra.Command{
	Use:   "martingale",
	Short: "Create AI Hub martingale strategies (spot/contract)",
}

func init() {
	spotCmd := &cobra.Command{
		Use:   "spot",
		Short: "Create a spot martingale strategy",
		Long: `Wraps PostAIHubSpotMartingaleCreate.

v7.2.78 contract: stop-loss is expressed via create_params.stop_loss_per_cycle
(per-round ratio); the legacy create_params.stop_loss_price is no longer
mapped on this path. Optional create_params.trigger_price is also accepted.`,
		RunE: runBotMartingaleSpotCreate,
	}
	spotCmd.Flags().String("json", "", "JSON body for SpotMartingaleCreateRequest (required)")
	mustRequireBotFlag(spotCmd, "json")

	contractCmd := &cobra.Command{
		Use:   "contract",
		Short: "Create a contract martingale strategy",
		Long: `Wraps PostAIHubContractMartingaleCreate.

Required JSON shape: {strategy_type, market, create_params:{invest_amount,
price_deviation, max_orders, take_profit_ratio, direction(buy/sell), leverage,
...}}.

Fixed stop-loss requires stop_loss_price and stop_loss_price_type together;
price type is last, index or mark. invest_amount is the total budget.
max_orders counts additional entries,
excluding the first. Monetary values must be JSON strings.`,
		RunE: runBotMartingaleContractCreate,
	}
	contractCmd.Flags().String("json", "", "JSON body for ContractMartingaleCreateRequest (required)")
	mustRequireBotFlag(contractCmd, "json")

	martingaleCmd.AddCommand(spotCmd, contractCmd)
	Cmd.AddCommand(martingaleCmd)
}

func runBotMartingaleSpotCreate(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.SpotMartingaleCreateRequest
	if err := decodeBotJSON(jsonStr, &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}

	result, httpResp, err := c.BotAPI.PostAIHubSpotMartingaleCreate(c.Context(), body, nil)
	if err := client.BotResultError(result, err, httpResp, "POST", "/api/v4/bot/spot-martingale/create", jsonStr); err != nil {
		return err
	}
	return p.Print(result)
}

func runBotMartingaleContractCreate(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.ContractMartingaleCreateRequest
	if err := decodeBotJSON(jsonStr, &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}

	if err := validateBotContractStop(body.CreateParams); err != nil {
		return err
	}
	result, httpResp, err := c.BotAPI.PostAIHubContractMartingaleCreate(c.Context(), body, nil)
	if err := client.BotResultError(result, err, httpResp, "POST", "/api/v4/bot/contract-martingale/create", jsonStr); err != nil {
		return err
	}
	return p.Print(result)
}
