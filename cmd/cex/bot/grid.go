package bot

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

var gridCmd = &cobra.Command{
	Use:   "grid",
	Short: "Create AI Hub grid strategies (spot/margin/infinite/futures)",
}

func init() {
	spotCmd := &cobra.Command{
		Use:   "spot",
		Short: "Create a spot grid strategy",
		Long:  "Wraps PostAIHubSpotGridCreate. Required JSON shape: {strategy_type, market, create_params:{money, low_price, high_price, grid_num, price_type, ...}}",
		RunE:  runBotGridSpotCreate,
	}
	spotCmd.Flags().String("json", "", "JSON body for SpotGridCreateRequest (required)")
	spotCmd.MarkFlagRequired("json")

	marginCmd := &cobra.Command{
		Use:   "margin",
		Short: "Create a margin (leverage) grid strategy",
		Long:  "Wraps PostAIHubMarginGridCreate. Required JSON shape: {strategy_type, market, create_params:{money, low_price, high_price, grid_num, price_type, leverage, direction?, ...}}",
		RunE:  runBotGridMarginCreate,
	}
	marginCmd.Flags().String("json", "", "JSON body for MarginGridCreateRequest (required)")
	marginCmd.MarkFlagRequired("json")

	infiniteCmd := &cobra.Command{
		Use:   "infinite",
		Short: "Create an infinite grid strategy",
		Long:  "Wraps PostAIHubInfiniteGridCreate. v7.2.78 makes grid_num/price_type optional; required JSON keys are money, price_floor, profit_per_grid.",
		RunE:  runBotGridInfiniteCreate,
	}
	infiniteCmd.Flags().String("json", "", "JSON body for InfiniteGridCreateRequest (required)")
	infiniteCmd.MarkFlagRequired("json")

	futuresCmd := &cobra.Command{
		Use:   "futures",
		Short: "Create a futures (contract) grid strategy",
		Long:  "Wraps PostAIHubFuturesGridCreate. Required JSON shape: {strategy_type, market, create_params:{money, low_price, high_price, grid_num, price_type, leverage, direction?, ...}}",
		RunE:  runBotGridFuturesCreate,
	}
	futuresCmd.Flags().String("json", "", "JSON body for FuturesGridCreateRequest (required)")
	futuresCmd.MarkFlagRequired("json")

	gridCmd.AddCommand(spotCmd, marginCmd, infiniteCmd, futuresCmd)
	Cmd.AddCommand(gridCmd)
}

func runBotGridSpotCreate(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.SpotGridCreateRequest
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}

	result, httpResp, err := c.BotAPI.PostAIHubSpotGridCreate(c.Context(), body, nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/bot/spot/grid/create", jsonStr))
		return nil
	}
	return p.Print(result)
}

func runBotGridMarginCreate(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.MarginGridCreateRequest
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}

	result, httpResp, err := c.BotAPI.PostAIHubMarginGridCreate(c.Context(), body, nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/bot/margin/grid/create", jsonStr))
		return nil
	}
	return p.Print(result)
}

func runBotGridInfiniteCreate(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.InfiniteGridCreateRequest
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}

	result, httpResp, err := c.BotAPI.PostAIHubInfiniteGridCreate(c.Context(), body, nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/bot/infinite/grid/create", jsonStr))
		return nil
	}
	return p.Print(result)
}

func runBotGridFuturesCreate(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.FuturesGridCreateRequest
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}

	result, httpResp, err := c.BotAPI.PostAIHubFuturesGridCreate(c.Context(), body, nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/bot/futures/grid/create", jsonStr))
		return nil
	}
	return p.Print(result)
}
