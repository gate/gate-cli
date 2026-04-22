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

	refundPreviewCmd := &cobra.Command{
		Use:   "refund-preview <order-id>",
		Short: "Preview early-redemption of a dual investment order",
		Args:  cobra.ExactArgs(1),
		RunE:  runDualRefundPreview,
	}

	refundCmd := &cobra.Command{
		Use:   "refund",
		Short: "Execute early-redemption of a dual investment order",
		RunE:  runDualRefund,
	}
	refundCmd.Flags().String("order-id", "", "Order ID (required)")
	refundCmd.Flags().String("req-id", "", "Request ID returned by refund-preview (required)")
	refundCmd.MarkFlagRequired("order-id")
	refundCmd.MarkFlagRequired("req-id")

	modifyReinvestCmd := &cobra.Command{
		Use:   "modify-reinvest",
		Short: "Modify reinvest setting of a dual investment order",
		RunE:  runDualModifyReinvest,
	}
	modifyReinvestCmd.Flags().Int64("order-id", 0, "Order ID (required)")
	modifyReinvestCmd.Flags().Int32("status", 0, "0=off, 1=on (required)")
	modifyReinvestCmd.Flags().Int64("duration", 0, "Effective duration in seconds; default 86400 (1 day) if omitted")
	modifyReinvestCmd.MarkFlagRequired("order-id")
	modifyReinvestCmd.MarkFlagRequired("status")

	recommendCmd := &cobra.Command{
		Use:   "recommend",
		Short: "Get recommended dual investment projects (public, no auth required)",
		RunE:  runDualRecommend,
	}
	recommendCmd.Flags().String("mode", "", "Sort mode: normal / senior / apy_up etc.")
	recommendCmd.Flags().String("coin", "", "Filter by invest currency, e.g. BTC, USDT")
	recommendCmd.Flags().String("type", "", "Filter by type: call (sell high) / put (buy low)")
	recommendCmd.Flags().String("history-pids", "", "Comma-separated product IDs already held by the user")

	dualCmd.AddCommand(plansCmd, ordersCmd, placeCmd, balanceCmd,
		refundPreviewCmd, refundCmd, modifyReinvestCmd, recommendCmd)
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

func runDualRefundPreview(cmd *cobra.Command, args []string) error {
	orderID := args[0]
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.EarnAPI.GetDualOrderRefundPreview(c.Context(), orderID)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/earn/dual/order-refund-preview", ""))
		return nil
	}
	return p.Print(result)
}

func runDualRefund(cmd *cobra.Command, args []string) error {
	orderID, _ := cmd.Flags().GetString("order-id")
	reqID, _ := cmd.Flags().GetString("req-id")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body := gateapi.DualOrderRefundParams{
		OrderId: orderID,
		ReqId:   reqID,
	}
	bodyJSON, _ := json.Marshal(body)
	httpResp, err := c.EarnAPI.PlaceDualOrderRefund(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/earn/dual/order-refund", string(bodyJSON)))
		return nil
	}
	return p.Print(map[string]string{"status": "ok", "order_id": orderID, "req_id": reqID})
}

func runDualModifyReinvest(cmd *cobra.Command, args []string) error {
	orderID, _ := cmd.Flags().GetInt64("order-id")
	status, _ := cmd.Flags().GetInt32("status")
	duration, _ := cmd.Flags().GetInt64("duration")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body := gateapi.DualModifyOrderReinvestParams{
		OrderId:               orderID,
		Status:                status,
		EffectiveTimeDuration: duration,
	}
	bodyJSON, _ := json.Marshal(body)
	httpResp, err := c.EarnAPI.ModifyDualOrderReinvest(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/earn/dual/modify-order-reinvest", string(bodyJSON)))
		return nil
	}
	return p.Print(map[string]any{"status": "ok", "order_id": orderID, "reinvest": status})
}

func runDualRecommend(cmd *cobra.Command, args []string) error {
	mode, _ := cmd.Flags().GetString("mode")
	coin, _ := cmd.Flags().GetString("coin")
	typ, _ := cmd.Flags().GetString("type")
	historyPids, _ := cmd.Flags().GetString("history-pids")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.GetDualProjectRecommendOpts{}
	if mode != "" {
		opts.Mode = optional.NewString(mode)
	}
	if coin != "" {
		opts.Coin = optional.NewString(coin)
	}
	if typ != "" {
		opts.Type_ = optional.NewString(typ)
	}
	if historyPids != "" {
		opts.HistoryPids = optional.NewString(historyPids)
	}

	result, httpResp, err := c.EarnAPI.GetDualProjectRecommend(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/earn/dual/project-recommend", ""))
		return nil
	}
	return p.Print(result)
}

// Ensure fmt is used (for potential future table formatting).
var _ = fmt.Sprintf
