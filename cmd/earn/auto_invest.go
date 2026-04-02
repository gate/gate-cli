package earn

import (
	"encoding/json"
	"fmt"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

var autoInvestCmd = &cobra.Command{
	Use:   "auto-invest",
	Short: "Auto-invest plan commands",
}

func init() {
	plansCmd := &cobra.Command{
		Use:   "plans",
		Short: "List auto-invest plans",
		RunE:  runAutoInvestPlans,
	}
	plansCmd.Flags().String("status", "Active", "Plan status: Active or History")
	plansCmd.Flags().Int64("page", 0, "Page number")
	plansCmd.Flags().Int64("page-size", 0, "Items per page (max 100)")

	planDetailCmd := &cobra.Command{
		Use:   "plan-detail",
		Short: "Get auto-invest plan detail",
		RunE:  runAutoInvestPlanDetail,
	}
	planDetailCmd.Flags().Int64("plan-id", 0, "Plan ID (required)")
	planDetailCmd.MarkFlagRequired("plan-id")

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create an auto-invest plan",
		RunE:  runAutoInvestCreate,
	}
	createCmd.Flags().String("json", "", "JSON body for plan creation (required)")
	createCmd.MarkFlagRequired("json")

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update an auto-invest plan",
		RunE:  runAutoInvestUpdate,
	}
	updateCmd.Flags().String("json", "", "JSON body for plan update (required)")
	updateCmd.MarkFlagRequired("json")

	stopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop an auto-invest plan",
		RunE:  runAutoInvestStop,
	}
	stopCmd.Flags().Int64("plan-id", 0, "Plan ID (required)")
	stopCmd.MarkFlagRequired("plan-id")

	addPositionCmd := &cobra.Command{
		Use:   "add-position",
		Short: "Add position to an auto-invest plan immediately",
		RunE:  runAutoInvestAddPosition,
	}
	addPositionCmd.Flags().Int64("plan-id", 0, "Plan ID (required)")
	addPositionCmd.Flags().String("amount", "", "Amount (required)")
	addPositionCmd.MarkFlagRequired("plan-id")
	addPositionCmd.MarkFlagRequired("amount")

	coinsCmd := &cobra.Command{
		Use:   "coins",
		Short: "List currencies supporting auto-invest (public, no auth required)",
		RunE:  runAutoInvestCoins,
	}
	coinsCmd.Flags().String("plan-money", "", "Pricing currency: USDT or BTC (default USDT)")

	minAmountCmd := &cobra.Command{
		Use:   "min-amount",
		Short: "Query minimum invest amount (public, no auth required)",
		RunE:  runAutoInvestMinAmount,
	}
	minAmountCmd.Flags().String("json", "", "JSON body for min amount query (required)")
	minAmountCmd.MarkFlagRequired("json")

	configCmd := &cobra.Command{
		Use:   "config",
		Short: "List auto-invest config (public, no auth required)",
		RunE:  runAutoInvestConfig,
	}

	recordsCmd := &cobra.Command{
		Use:   "records",
		Short: "List plan execution records",
		RunE:  runAutoInvestRecords,
	}
	recordsCmd.Flags().Int64("plan-id", 0, "Plan ID (required)")
	recordsCmd.Flags().Int64("page", 0, "Page number")
	recordsCmd.Flags().Int64("page-size", 0, "Items per page (max 100)")
	recordsCmd.MarkFlagRequired("plan-id")

	ordersCmd := &cobra.Command{
		Use:   "orders",
		Short: "List auto-invest orders",
		RunE:  runAutoInvestOrders,
	}
	ordersCmd.Flags().Int64("plan-id", 0, "Plan ID (required)")
	ordersCmd.Flags().Int64("record-id", 0, "Record ID (required)")
	ordersCmd.MarkFlagRequired("plan-id")
	ordersCmd.MarkFlagRequired("record-id")

	autoInvestCmd.AddCommand(plansCmd, planDetailCmd, createCmd, updateCmd, stopCmd,
		addPositionCmd, coinsCmd, minAmountCmd, configCmd, recordsCmd, ordersCmd)
}

func runAutoInvestPlans(cmd *cobra.Command, args []string) error {
	status, _ := cmd.Flags().GetString("status")
	page, _ := cmd.Flags().GetInt64("page")
	pageSize, _ := cmd.Flags().GetInt64("page-size")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListAutoInvestPlansOpts{}
	if page != 0 {
		opts.Page = optional.NewInt64(page)
	}
	if pageSize != 0 {
		opts.PageSize = optional.NewInt64(pageSize)
	}

	result, httpResp, err := c.EarnAPI.ListAutoInvestPlans(c.Context(), status, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/earn/autoinvest/plans/list_info", ""))
		return nil
	}
	return p.Print(result)
}

func runAutoInvestPlanDetail(cmd *cobra.Command, args []string) error {
	planID, _ := cmd.Flags().GetInt64("plan-id")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.EarnAPI.GetAutoInvestPlanDetail(c.Context(), planID)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/earn/autoinvest/plans/detail", ""))
		return nil
	}
	return p.Print(result)
}

func runAutoInvestCreate(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.AutoInvestPlanCreate
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}

	result, httpResp, err := c.EarnAPI.CreateAutoInvestPlan(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/earn/autoinvest/plans", jsonStr))
		return nil
	}
	return p.Print(result)
}

func runAutoInvestUpdate(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.AutoInvestPlanUpdate
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}

	httpResp, err := c.EarnAPI.UpdateAutoInvestPlan(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "PUT", "/api/v4/earn/autoinvest/plans", jsonStr))
		return nil
	}
	return p.Print(map[string]string{"status": "ok"})
}

func runAutoInvestStop(cmd *cobra.Command, args []string) error {
	planID, _ := cmd.Flags().GetInt64("plan-id")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body := gateapi.AutoInvestPlanStop{PlanId: planID}
	bodyJSON, _ := json.Marshal(body)
	httpResp, err := c.EarnAPI.StopAutoInvestPlan(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/earn/autoinvest/plans/stop", string(bodyJSON)))
		return nil
	}
	return p.Print(map[string]string{"status": "ok"})
}

func runAutoInvestAddPosition(cmd *cobra.Command, args []string) error {
	planID, _ := cmd.Flags().GetInt64("plan-id")
	amount, _ := cmd.Flags().GetString("amount")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body := gateapi.AutoInvestPlanAddPosition{PlanId: planID, Amount: amount}
	bodyJSON, _ := json.Marshal(body)
	httpResp, err := c.EarnAPI.AddPositionAutoInvestPlan(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/earn/autoinvest/plans/add_position", string(bodyJSON)))
		return nil
	}
	return p.Print(map[string]string{"status": "ok"})
}

func runAutoInvestCoins(cmd *cobra.Command, args []string) error {
	planMoney, _ := cmd.Flags().GetString("plan-money")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	var opts *gateapi.ListAutoInvestCoinsOpts
	if planMoney != "" {
		opts = &gateapi.ListAutoInvestCoinsOpts{
			PlanMoney: optional.NewString(planMoney),
		}
	}

	result, httpResp, err := c.EarnAPI.ListAutoInvestCoins(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/earn/autoinvest/coins", ""))
		return nil
	}
	return p.Print(result)
}

func runAutoInvestMinAmount(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	var body gateapi.AutoInvestMinInvestAmount
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}

	result, httpResp, err := c.EarnAPI.GetAutoInvestMinAmount(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/earn/autoinvest/min_invest_amount", jsonStr))
		return nil
	}
	return p.Print(result)
}

func runAutoInvestConfig(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.EarnAPI.ListAutoInvestConfig(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/earn/autoinvest/config", ""))
		return nil
	}
	return p.Print(result)
}

func runAutoInvestRecords(cmd *cobra.Command, args []string) error {
	planID, _ := cmd.Flags().GetInt64("plan-id")
	page, _ := cmd.Flags().GetInt64("page")
	pageSize, _ := cmd.Flags().GetInt64("page-size")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListAutoInvestPlanRecordsOpts{}
	if page != 0 {
		opts.Page = optional.NewInt64(page)
	}
	if pageSize != 0 {
		opts.PageSize = optional.NewInt64(pageSize)
	}

	result, httpResp, err := c.EarnAPI.ListAutoInvestPlanRecords(c.Context(), planID, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/earn/autoinvest/plans/records", ""))
		return nil
	}
	return p.Print(result)
}

func runAutoInvestOrders(cmd *cobra.Command, args []string) error {
	planID, _ := cmd.Flags().GetInt64("plan-id")
	recordID, _ := cmd.Flags().GetInt64("record-id")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.EarnAPI.ListAutoInvestOrders(c.Context(), planID, recordID)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/earn/autoinvest/orders", ""))
		return nil
	}
	return p.Print(result)
}
