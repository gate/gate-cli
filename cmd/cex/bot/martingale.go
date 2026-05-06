package bot

import (
	"encoding/json"
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
	spotCmd.MarkFlagRequired("json")

	contractCmd := &cobra.Command{
		Use:   "contract",
		Short: "Create a contract martingale strategy",
		Long: `Wraps PostAIHubContractMartingaleCreate.

Required JSON shape: {strategy_type, market, create_params:{invest_amount,
price_deviation, max_orders, take_profit_ratio, direction(buy/sell), leverage,
...}}.

v7.2.78 note: the SDK still defines create_params.stop_loss_price for backward
compatibility, but per upstream SDK docs the AIHub contract_martingale path
does not map this field. Do not include stop_loss_price in --json; follow the
contract martingale rules of the underlying API instead.`,
		RunE: runBotMartingaleContractCreate,
	}
	contractCmd.Flags().String("json", "", "JSON body for ContractMartingaleCreateRequest (required)")
	contractCmd.MarkFlagRequired("json")

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
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}

	result, httpResp, err := c.BotAPI.PostAIHubSpotMartingaleCreate(c.Context(), body, nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/bot/spot/martingale/create", jsonStr))
		return nil
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
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}

	result, httpResp, err := c.BotAPI.PostAIHubContractMartingaleCreate(c.Context(), body, nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/bot/contract/martingale/create", jsonStr))
		return nil
	}
	return p.Print(result)
}
