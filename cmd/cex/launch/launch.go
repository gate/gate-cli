package launch

import (
	"encoding/json"
	"fmt"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

// Cmd is the root command for the launch module.
var Cmd = &cobra.Command{
	Use:   "launch",
	Short: "Launch pool commands",
}

func init() {
	projectsCmd := &cobra.Command{
		Use:   "projects",
		Short: "List LaunchPool projects (public)",
		RunE:  runProjects,
	}
	projectsCmd.Flags().Int32("status", 0, "Filter: 0=all, 1=ongoing, 2=warming up, 3=ended, 4=ongoing+warming up")
	projectsCmd.Flags().String("mortgage-coin", "", "Exact match by staking currency")
	projectsCmd.Flags().String("search-coin", "", "Fuzzy match by reward currency/name")
	projectsCmd.Flags().Int32("limit-rule", -1, "0=regular pool, 1=beginner pool")
	projectsCmd.Flags().Int32("sort-type", 0, "1=max APR desc, 2=max APR asc")
	projectsCmd.Flags().Int32("page", 0, "Page number")
	projectsCmd.Flags().Int32("page-size", 0, "Items per page (max 30)")

	pledgeCmd := &cobra.Command{
		Use:   "pledge",
		Short: "Create a LaunchPool staking order",
		RunE:  runPledge,
	}
	pledgeCmd.Flags().String("json", "", "JSON body for CreateOrderV4 (required)")
	pledgeCmd.MarkFlagRequired("json")

	redeemCmd := &cobra.Command{
		Use:   "redeem",
		Short: "Redeem LaunchPool staked assets",
		RunE:  runRedeem,
	}
	redeemCmd.Flags().String("json", "", "JSON body for RedeemV4 (required)")
	redeemCmd.MarkFlagRequired("json")

	pledgeRecordsCmd := &cobra.Command{
		Use:   "pledge-records",
		Short: "List user pledge records",
		RunE:  runPledgeRecords,
	}
	pledgeRecordsCmd.Flags().Int64("page", 0, "Page number")
	pledgeRecordsCmd.Flags().Int64("page-size", 0, "Items per page (max 30)")
	pledgeRecordsCmd.Flags().Int32("type", 0, "1=pledge, 2=redemption")
	pledgeRecordsCmd.Flags().String("start-time", "", "Start time (YYYY-MM-DD HH:MM:SS)")
	pledgeRecordsCmd.Flags().String("end-time", "", "End time (YYYY-MM-DD HH:MM:SS)")
	pledgeRecordsCmd.Flags().String("coin", "", "Collateral currency")

	rewardRecordsCmd := &cobra.Command{
		Use:   "reward-records",
		Short: "List user reward records",
		RunE:  runRewardRecords,
	}
	rewardRecordsCmd.Flags().Int64("page", 0, "Page number")
	rewardRecordsCmd.Flags().Int64("page-size", 0, "Items per page (max 30)")
	rewardRecordsCmd.Flags().Int64("start-time", 0, "Start timestamp")
	rewardRecordsCmd.Flags().Int64("end-time", 0, "End timestamp")
	rewardRecordsCmd.Flags().String("coin", "", "Reward currency")

	Cmd.AddCommand(projectsCmd, pledgeCmd, redeemCmd, pledgeRecordsCmd, rewardRecordsCmd)
	registerCandyDropCommands(Cmd)
	registerHodlerAirdropCommands(Cmd)
}

func runProjects(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.ListLaunchPoolProjectsOpts{}
	if v, _ := cmd.Flags().GetInt32("status"); v > 0 {
		opts.Status = optional.NewInt32(v)
	}
	if v, _ := cmd.Flags().GetString("mortgage-coin"); v != "" {
		opts.MortgageCoin = optional.NewString(v)
	}
	if v, _ := cmd.Flags().GetString("search-coin"); v != "" {
		opts.SearchCoin = optional.NewString(v)
	}
	if v, _ := cmd.Flags().GetInt32("limit-rule"); v >= 0 {
		opts.LimitRule = optional.NewInt32(v)
	}
	if v, _ := cmd.Flags().GetInt32("sort-type"); v > 0 {
		opts.SortType = optional.NewInt32(v)
	}
	if v, _ := cmd.Flags().GetInt32("page"); v > 0 {
		opts.Page = optional.NewInt32(v)
	}
	if v, _ := cmd.Flags().GetInt32("page-size"); v > 0 {
		opts.PageSize = optional.NewInt32(v)
	}

	result, httpResp, err := c.LaunchAPI.ListLaunchPoolProjects(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/launch/project-list", ""))
		return nil
	}
	return p.Print(result)
}

func runPledge(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.CreateOrderV4
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}
	result, httpResp, err := c.LaunchAPI.CreateLaunchPoolOrder(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/launch/create-order", jsonStr))
		return nil
	}
	return p.Print(result)
}

func runRedeem(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.RedeemV4
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}
	result, httpResp, err := c.LaunchAPI.RedeemLaunchPool(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/launch/redeem", jsonStr))
		return nil
	}
	return p.Print(result)
}

func runPledgeRecords(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListLaunchPoolPledgeRecordsOpts{}
	if v, _ := cmd.Flags().GetInt64("page"); v > 0 {
		opts.Page = optional.NewInt64(v)
	}
	if v, _ := cmd.Flags().GetInt64("page-size"); v > 0 {
		opts.PageSize = optional.NewInt64(v)
	}
	if v, _ := cmd.Flags().GetInt32("type"); v > 0 {
		opts.Type_ = optional.NewInt32(v)
	}
	if v, _ := cmd.Flags().GetString("start-time"); v != "" {
		opts.StartTime = optional.NewString(v)
	}
	if v, _ := cmd.Flags().GetString("end-time"); v != "" {
		opts.EndTime = optional.NewString(v)
	}
	if v, _ := cmd.Flags().GetString("coin"); v != "" {
		opts.Coin = optional.NewString(v)
	}

	result, httpResp, err := c.LaunchAPI.ListLaunchPoolPledgeRecords(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/launch/user-pledge-records", ""))
		return nil
	}
	return p.Print(result)
}

func runRewardRecords(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListLaunchPoolRewardRecordsOpts{}
	if v, _ := cmd.Flags().GetInt64("page"); v > 0 {
		opts.Page = optional.NewInt64(v)
	}
	if v, _ := cmd.Flags().GetInt64("page-size"); v > 0 {
		opts.PageSize = optional.NewInt64(v)
	}
	if v, _ := cmd.Flags().GetInt64("start-time"); v > 0 {
		opts.StartTime = optional.NewInt64(v)
	}
	if v, _ := cmd.Flags().GetInt64("end-time"); v > 0 {
		opts.EndTime = optional.NewInt64(v)
	}
	if v, _ := cmd.Flags().GetString("coin"); v != "" {
		opts.Coin = optional.NewString(v)
	}

	result, httpResp, err := c.LaunchAPI.ListLaunchPoolRewardRecords(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/launch/get-user-reward-records", ""))
		return nil
	}
	return p.Print(result)
}
