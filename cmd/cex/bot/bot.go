// Package bot wraps Gate AI Hub (BotAPI) endpoints under `gate-cli cex bot`.
//
// Coverage of the BotApiService surface in gateapi-go v7.2.78:
//
//	bot recommend                       GetAIHubStrategyRecommend
//	bot running                         GetAIHubPortfolioRunning
//	bot detail                          GetAIHubPortfolioDetail
//	bot stop                            PostAIHubPortfolioStop
//	bot grid spot      / margin / infinite / futures  (Post*GridCreate)
//	bot martingale spot / contract                    (Post*MartingaleCreate)
package bot

import (
	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

// Cmd is the root command for the AI Hub (bot) module.
var Cmd = &cobra.Command{
	Use:   "bot",
	Short: "Gate AI Hub (quant strategies) commands",
	Long:  "Discover, create, query, and stop AI Hub strategies (spot/margin/infinite/futures grid + spot/contract martingale).",
}

func init() {
	recommendCmd := &cobra.Command{
		Use:   "recommend",
		Short: "List AI Hub recommended strategies",
		RunE:  runBotRecommend,
	}
	recommendCmd.Flags().String("market", "", "Trading pair, e.g. BTC_USDT")
	recommendCmd.Flags().String("strategy-type", "", "Strategy type filter (e.g. spot_grid, futures_grid, spot_martingale)")
	recommendCmd.Flags().String("direction", "", "Market direction")
	recommendCmd.Flags().String("invest-amount", "", "Investment amount")
	recommendCmd.Flags().String("scene", "", "Recommend scene: top1 / bundle / filter / refresh")
	recommendCmd.Flags().String("refresh-recommendation-id", "", "Recommendation ID for scene=refresh; format strategy_type|market[|backtest_id]")
	recommendCmd.Flags().Int32("limit", 0, "Max results returned (filter scene capped at 10)")
	recommendCmd.Flags().String("max-drawdown-lte", "", "Max drawdown upper bound")
	recommendCmd.Flags().String("backtest-apr-gte", "", "Backtest annualized return lower bound")

	runningCmd := &cobra.Command{
		Use:   "running",
		Short: "List currently-running strategies",
		RunE:  runBotRunning,
	}
	runningCmd.Flags().String("strategy-type", "", "Filter by strategy type")
	runningCmd.Flags().String("market", "", "Filter by trading pair")
	runningCmd.Flags().Int32("page", 0, "Page number")
	runningCmd.Flags().Int32("page-size", 0, "Page size")

	detailCmd := &cobra.Command{
		Use:   "detail",
		Short: "Get strategy details",
		RunE:  runBotDetail,
	}
	detailCmd.Flags().String("strategy-id", "", "Strategy ID (required)")
	detailCmd.MarkFlagRequired("strategy-id")
	detailCmd.Flags().String("strategy-type", "", "Strategy type (required); used to dispatch to the underlying detail implementation")
	detailCmd.MarkFlagRequired("strategy-type")

	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop a running strategy",
		RunE:  runBotStop,
	}
	stopCmd.Flags().String("strategy-id", "", "Strategy ID (required)")
	stopCmd.MarkFlagRequired("strategy-id")
	stopCmd.Flags().String("strategy-type", "", "Strategy type (required)")
	stopCmd.MarkFlagRequired("strategy-type")

	Cmd.AddCommand(recommendCmd, runningCmd, detailCmd, stopCmd)
}

func runBotRecommend(cmd *cobra.Command, args []string) error {
	market, _ := cmd.Flags().GetString("market")
	strategyType, _ := cmd.Flags().GetString("strategy-type")
	direction, _ := cmd.Flags().GetString("direction")
	investAmount, _ := cmd.Flags().GetString("invest-amount")
	scene, _ := cmd.Flags().GetString("scene")
	refreshID, _ := cmd.Flags().GetString("refresh-recommendation-id")
	limit, _ := cmd.Flags().GetInt32("limit")
	maxDD, _ := cmd.Flags().GetString("max-drawdown-lte")
	aprGte, _ := cmd.Flags().GetString("backtest-apr-gte")

	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.GetAIHubStrategyRecommendOpts{}
	if market != "" {
		opts.Market = optional.NewString(market)
	}
	if strategyType != "" {
		opts.StrategyType = optional.NewString(strategyType)
	}
	if direction != "" {
		opts.Direction = optional.NewString(direction)
	}
	if investAmount != "" {
		opts.InvestAmount = optional.NewString(investAmount)
	}
	if scene != "" {
		opts.Scene = optional.NewString(scene)
	}
	if refreshID != "" {
		opts.RefreshRecommendationId = optional.NewString(refreshID)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if maxDD != "" {
		opts.MaxDrawdownLte = optional.NewString(maxDD)
	}
	if aprGte != "" {
		opts.BacktestAprGte = optional.NewString(aprGte)
	}

	result, httpResp, err := c.BotAPI.GetAIHubStrategyRecommend(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/bot/strategy/recommend", ""))
		return nil
	}
	return p.Print(result)
}

func runBotRunning(cmd *cobra.Command, args []string) error {
	strategyType, _ := cmd.Flags().GetString("strategy-type")
	market, _ := cmd.Flags().GetString("market")
	page, _ := cmd.Flags().GetInt32("page")
	pageSize, _ := cmd.Flags().GetInt32("page-size")

	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.GetAIHubPortfolioRunningOpts{}
	if strategyType != "" {
		opts.StrategyType = optional.NewString(strategyType)
	}
	if market != "" {
		opts.Market = optional.NewString(market)
	}
	if page != 0 {
		opts.Page = optional.NewInt32(page)
	}
	if pageSize != 0 {
		opts.PageSize = optional.NewInt32(pageSize)
	}

	result, httpResp, err := c.BotAPI.GetAIHubPortfolioRunning(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/bot/portfolio/running", ""))
		return nil
	}
	return p.Print(result)
}

func runBotDetail(cmd *cobra.Command, args []string) error {
	strategyID, _ := cmd.Flags().GetString("strategy-id")
	strategyType, _ := cmd.Flags().GetString("strategy-type")

	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.BotAPI.GetAIHubPortfolioDetail(c.Context(), strategyID, strategyType, nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/bot/portfolio/detail", ""))
		return nil
	}
	return p.Print(result)
}

func runBotStop(cmd *cobra.Command, args []string) error {
	strategyID, _ := cmd.Flags().GetString("strategy-id")
	strategyType, _ := cmd.Flags().GetString("strategy-type")

	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body := gateapi.AiHubPortfolioStopRequest{
		StrategyId:   strategyID,
		StrategyType: gateapi.StrategyType(strategyType),
	}

	result, httpResp, err := c.BotAPI.PostAIHubPortfolioStop(c.Context(), body, nil)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/bot/portfolio/stop", ""))
		return nil
	}
	return p.Print(result)
}
