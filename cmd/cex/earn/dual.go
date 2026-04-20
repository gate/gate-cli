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

var dualCmd = &cobra.Command{
	Use:   "dual",
	Short: "Dual Investment commands",
}

func init() {
	plansCmd := &cobra.Command{
		Use:   "plans",
		Short: "List dual investment plans (public, no auth required)",
		RunE:  runDualPlans,
	}
	plansCmd.Flags().Int64("plan-id", 0, "Filter by plan ID")

	ordersCmd := &cobra.Command{
		Use:   "orders",
		Short: "List dual investment orders",
		RunE:  runDualOrders,
	}
	ordersCmd.Flags().Int64("from", 0, "Start settlement time")
	ordersCmd.Flags().Int64("to", 0, "End settlement time")
	ordersCmd.Flags().Int32("page", 0, "Page number")
	ordersCmd.Flags().Int32("limit", 0, "Maximum number of records")

	placeCmd := &cobra.Command{
		Use:   "place",
		Short: "Place a dual investment order",
		RunE:  runDualPlace,
	}
	placeCmd.Flags().String("plan-id", "", "Product plan ID (required)")
	placeCmd.Flags().String("amount", "", "Subscription amount (required)")
	placeCmd.Flags().String("text", "", "Custom order ID (must start with t-)")
	placeCmd.MarkFlagRequired("plan-id")
	placeCmd.MarkFlagRequired("amount")

	balanceCmd := &cobra.Command{
		Use:   "balance",
		Short: "Get dual investment balance",
		RunE:  runDualBalance,
	}

	dualCmd.AddCommand(plansCmd, ordersCmd, placeCmd, balanceCmd)
}

func runDualPlans(cmd *cobra.Command, args []string) error {
	planID, _ := cmd.Flags().GetInt64("plan-id")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	var opts *gateapi.ListDualInvestmentPlansOpts
	if planID != 0 {
		opts = &gateapi.ListDualInvestmentPlansOpts{
			PlanId: optional.NewInt64(planID),
		}
	}

	result, httpResp, err := c.EarnAPI.ListDualInvestmentPlans(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/earn/dual/investment_plan", ""))
		return nil
	}
	return p.Print(result)
}

func runDualOrders(cmd *cobra.Command, args []string) error {
	from, _ := cmd.Flags().GetInt64("from")
	to, _ := cmd.Flags().GetInt64("to")
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

	opts := &gateapi.ListDualOrdersOpts{}
	if from != 0 {
		opts.From = optional.NewInt64(from)
	}
	if to != 0 {
		opts.To = optional.NewInt64(to)
	}
	if page != 0 {
		opts.Page = optional.NewInt32(page)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.EarnAPI.ListDualOrders(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/earn/dual/orders", ""))
		return nil
	}
	return p.Print(result)
}

func runDualPlace(cmd *cobra.Command, args []string) error {
	planID, _ := cmd.Flags().GetString("plan-id")
	amount, _ := cmd.Flags().GetString("amount")
	text, _ := cmd.Flags().GetString("text")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body := gateapi.PlaceDualInvestmentOrderParams{
		PlanId: planID,
		Amount: amount,
		Text:   text,
	}
	bodyJSON, _ := json.Marshal(body)
	result, httpResp, err := c.EarnAPI.PlaceDualOrder(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/earn/dual/orders", string(bodyJSON)))
		return nil
	}
	return p.Print(result)
}

func runDualBalance(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.EarnAPI.ListDualBalance(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/earn/dual/accounts", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"Asset (USDT)", "Asset (BTC)", "Interest (USDT)", "Interest (BTC)"},
		[][]string{{result.UserAssetUsdt, result.UserAssetBtc, result.UserTotalInterestUsdt, result.UserTotalInterestBtc}},
	)
}

// Ensure fmt is used (for potential future table formatting).
var _ = fmt.Sprintf
