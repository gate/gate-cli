package launch

import (
	"encoding/json"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

// registerCandyDropCommands attaches the Candy Drop V4 sub-tree under
// `cex launch candy-drop ...`. Introduced alongside SDK v7.2.71 adoption to
// close the CLI gap against the launch-module MCP tools.
func registerCandyDropCommands(parent *cobra.Command) {
	candyDropCmd := &cobra.Command{
		Use:   "candy-drop",
		Short: "Candy Drop V4 commands",
	}

	activitiesCmd := &cobra.Command{
		Use:   "activities",
		Short: "List Candy Drop V4 activities (public)",
		RunE:  runCandyDropActivities,
	}
	activitiesCmd.Flags().String("status", "", "Filter by activity status")
	activitiesCmd.Flags().String("rule-name", "", "Filter by rule name")
	activitiesCmd.Flags().String("register-status", "", "Filter by register status")
	activitiesCmd.Flags().String("currency", "", "Filter by reward currency")
	activitiesCmd.Flags().Int32("limit", 0, "Page size")
	activitiesCmd.Flags().Int32("offset", 0, "Pagination offset")

	rulesCmd := &cobra.Command{
		Use:   "rules",
		Short: "Query Candy Drop V4 activity rules (public)",
		RunE:  runCandyDropRules,
	}
	rulesCmd.Flags().Int64("activity-id", 0, "Activity ID")
	rulesCmd.Flags().String("currency", "", "Currency")

	registerCmd := &cobra.Command{
		Use:   "register",
		Short: "Register for a Candy Drop V4 activity (auth required)",
		RunE:  runCandyDropRegister,
	}
	registerCmd.Flags().String("currency", "", "Project/currency name (required)")
	registerCmd.Flags().Int64("activity-id", 0, "Activity ID (optional, used with currency)")
	registerCmd.MarkFlagRequired("currency")

	progressCmd := &cobra.Command{
		Use:   "progress",
		Short: "Query Candy Drop V4 task completion progress (auth required)",
		RunE:  runCandyDropProgress,
	}
	progressCmd.Flags().Int64("activity-id", 0, "Activity ID")
	progressCmd.Flags().String("currency", "", "Currency")

	participationsCmd := &cobra.Command{
		Use:   "participations",
		Short: "List Candy Drop V4 participation records (auth required)",
		RunE:  runCandyDropParticipations,
	}
	participationsCmd.Flags().String("currency", "", "Currency")
	participationsCmd.Flags().String("status", "", "Participation status")
	participationsCmd.Flags().Int64("start-time", 0, "Start timestamp")
	participationsCmd.Flags().Int64("end-time", 0, "End timestamp")
	participationsCmd.Flags().Int32("page", 0, "Page number")
	participationsCmd.Flags().Int32("limit", 0, "Page size")

	airdropsCmd := &cobra.Command{
		Use:   "airdrops",
		Short: "List Candy Drop V4 airdrop records (auth required)",
		RunE:  runCandyDropAirdrops,
	}
	airdropsCmd.Flags().String("currency", "", "Currency")
	airdropsCmd.Flags().Int64("start-time", 0, "Start timestamp")
	airdropsCmd.Flags().Int64("end-time", 0, "End timestamp")
	airdropsCmd.Flags().Int32("page", 0, "Page number")
	airdropsCmd.Flags().Int32("limit", 0, "Page size")

	candyDropCmd.AddCommand(activitiesCmd, rulesCmd, registerCmd, progressCmd, participationsCmd, airdropsCmd)
	parent.AddCommand(candyDropCmd)
}

func runCandyDropActivities(cmd *cobra.Command, args []string) error {
	status, _ := cmd.Flags().GetString("status")
	ruleName, _ := cmd.Flags().GetString("rule-name")
	registerStatus, _ := cmd.Flags().GetString("register-status")
	currency, _ := cmd.Flags().GetString("currency")
	limit, _ := cmd.Flags().GetInt32("limit")
	offset, _ := cmd.Flags().GetInt32("offset")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.GetCandyDropActivityListV4Opts{}
	if status != "" {
		opts.Status = optional.NewString(status)
	}
	if ruleName != "" {
		opts.RuleName = optional.NewString(ruleName)
	}
	if registerStatus != "" {
		opts.RegisterStatus = optional.NewString(registerStatus)
	}
	if currency != "" {
		opts.Currency = optional.NewString(currency)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if offset != 0 {
		opts.Offset = optional.NewInt32(offset)
	}

	result, httpResp, err := c.LaunchAPI.GetCandyDropActivityListV4(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/candy_drop/v4/activities", ""))
		return nil
	}
	return p.Print(result)
}

func runCandyDropRules(cmd *cobra.Command, args []string) error {
	activityID, _ := cmd.Flags().GetInt64("activity-id")
	currency, _ := cmd.Flags().GetString("currency")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.GetCandyDropActivityRulesV4Opts{}
	if activityID != 0 {
		opts.ActivityId = optional.NewInt64(activityID)
	}
	if currency != "" {
		opts.Currency = optional.NewString(currency)
	}

	result, httpResp, err := c.LaunchAPI.GetCandyDropActivityRulesV4(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/candy_drop/v4/rules", ""))
		return nil
	}
	return p.Print(result)
}

func runCandyDropRegister(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	activityID, _ := cmd.Flags().GetInt64("activity-id")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	req := gateapi.CandyDropV4RegisterReqCd02{
		Currency:   currency,
		ActivityId: activityID,
	}
	body, _ := json.Marshal(req)
	result, httpResp, err := c.LaunchAPI.RegisterCandyDropV4(c.Context(), req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/candy_drop/v4/register", string(body)))
		return nil
	}
	return p.Print(result)
}

func runCandyDropProgress(cmd *cobra.Command, args []string) error {
	activityID, _ := cmd.Flags().GetInt64("activity-id")
	currency, _ := cmd.Flags().GetString("currency")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.GetCandyDropTaskProgressV4Opts{}
	if activityID != 0 {
		opts.ActivityId = optional.NewInt64(activityID)
	}
	if currency != "" {
		opts.Currency = optional.NewString(currency)
	}

	result, httpResp, err := c.LaunchAPI.GetCandyDropTaskProgressV4(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/candy_drop/v4/progress", ""))
		return nil
	}
	return p.Print(result)
}

func runCandyDropParticipations(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	status, _ := cmd.Flags().GetString("status")
	startTime, _ := cmd.Flags().GetInt64("start-time")
	endTime, _ := cmd.Flags().GetInt64("end-time")
	page, _ := cmd.Flags().GetInt32("page")
	limit, _ := cmd.Flags().GetInt32("limit")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.GetCandyDropParticipationRecordsV4Opts{}
	if currency != "" {
		opts.Currency = optional.NewString(currency)
	}
	if status != "" {
		opts.Status = optional.NewString(status)
	}
	if startTime != 0 {
		opts.StartTime = optional.NewInt64(startTime)
	}
	if endTime != 0 {
		opts.EndTime = optional.NewInt64(endTime)
	}
	if page != 0 {
		opts.Page = optional.NewInt32(page)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.LaunchAPI.GetCandyDropParticipationRecordsV4(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/candy_drop/v4/participations", ""))
		return nil
	}
	return p.Print(result)
}

func runCandyDropAirdrops(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	startTime, _ := cmd.Flags().GetInt64("start-time")
	endTime, _ := cmd.Flags().GetInt64("end-time")
	page, _ := cmd.Flags().GetInt32("page")
	limit, _ := cmd.Flags().GetInt32("limit")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.GetCandyDropAirdropRecordsV4Opts{}
	if currency != "" {
		opts.Currency = optional.NewString(currency)
	}
	if startTime != 0 {
		opts.StartTime = optional.NewInt64(startTime)
	}
	if endTime != 0 {
		opts.EndTime = optional.NewInt64(endTime)
	}
	if page != 0 {
		opts.Page = optional.NewInt32(page)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.LaunchAPI.GetCandyDropAirdropRecordsV4(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/candy_drop/v4/airdrops", ""))
		return nil
	}
	return p.Print(result)
}
