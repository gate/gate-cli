package mcl

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

// Cmd is the root command for the multi-collateral loan module.
var Cmd = &cobra.Command{
	Use:   "mcl",
	Short: "Multi-collateral loan commands",
}

func init() {
	currenciesCmd := &cobra.Command{
		Use:   "currencies",
		Short: "List supported multi-collateral currencies (public)",
		RunE:  runCurrencies,
	}

	quotaCmd := &cobra.Command{
		Use:   "quota",
		Short: "Query user currency quota",
		RunE:  runQuota,
	}
	quotaCmd.Flags().String("type", "", "Currency type: collateral or borrow (required)")
	quotaCmd.Flags().String("currency", "", "Currency name(s), comma-separated for collateral (required)")
	quotaCmd.MarkFlagRequired("type")
	quotaCmd.MarkFlagRequired("currency")

	ltvCmd := &cobra.Command{
		Use:   "ltv",
		Short: "Get multi-collateral LTV (public)",
		RunE:  runLtv,
	}

	currentRateCmd := &cobra.Command{
		Use:   "current-rate",
		Short: "Query currency current interest rate (public)",
		RunE:  runCurrentRate,
	}
	currentRateCmd.Flags().String("currencies", "", "Currency names, comma-separated (required)")
	currentRateCmd.Flags().String("vip-level", "", "VIP level (optional)")
	currentRateCmd.MarkFlagRequired("currencies")

	fixRateCmd := &cobra.Command{
		Use:   "fix-rate",
		Short: "Get multi-collateral fixed interest rate (public)",
		RunE:  runFixRate,
	}

	ordersCmd := &cobra.Command{
		Use:   "orders",
		Short: "List multi-collateral loan orders",
		RunE:  runOrders,
	}
	ordersCmd.Flags().Int32("page", 0, "Page number")
	ordersCmd.Flags().Int32("limit", 0, "Maximum number of records")
	ordersCmd.Flags().String("sort", "", "Sort type: time_desc, ltv_asc, ltv_desc")
	ordersCmd.Flags().String("order-type", "", "Order type: current or fixed")

	orderCmd := &cobra.Command{
		Use:   "order",
		Short: "Get multi-collateral order detail",
		RunE:  runOrder,
	}
	orderCmd.Flags().String("id", "", "Order ID (required)")
	orderCmd.MarkFlagRequired("id")

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a multi-collateral loan order",
		RunE:  runCreate,
	}
	createCmd.Flags().String("json", "", "JSON body for CreateMultiCollateralOrder (required)")
	createCmd.MarkFlagRequired("json")

	repayCmd := &cobra.Command{
		Use:   "repay",
		Short: "Repay a multi-collateral loan",
		RunE:  runRepay,
	}
	repayCmd.Flags().String("json", "", "JSON body for RepayMultiLoan (required)")
	repayCmd.MarkFlagRequired("json")

	repayRecordsCmd := &cobra.Command{
		Use:   "repay-records",
		Short: "List multi-collateral repayment records",
		RunE:  runRepayRecords,
	}
	repayRecordsCmd.Flags().String("type", "", "Operation type: repay or liquidate (required)")
	repayRecordsCmd.Flags().String("borrow-currency", "", "Filter by borrowed currency")
	repayRecordsCmd.Flags().Int32("page", 0, "Page number")
	repayRecordsCmd.Flags().Int32("limit", 0, "Maximum number of records")
	repayRecordsCmd.Flags().Int64("from", 0, "Start timestamp")
	repayRecordsCmd.Flags().Int64("to", 0, "End timestamp")
	repayRecordsCmd.MarkFlagRequired("type")

	recordsCmd := &cobra.Command{
		Use:   "records",
		Short: "List collateral adjustment records",
		RunE:  runRecords,
	}
	recordsCmd.Flags().Int32("page", 0, "Page number")
	recordsCmd.Flags().Int32("limit", 0, "Maximum number of records")
	recordsCmd.Flags().Int64("from", 0, "Start timestamp")
	recordsCmd.Flags().Int64("to", 0, "End timestamp")
	recordsCmd.Flags().String("collateral-currency", "", "Filter by collateral currency")

	collateralCmd := &cobra.Command{
		Use:   "collateral",
		Short: "Adjust collateral (append or redeem)",
		RunE:  runCollateral,
	}
	collateralCmd.Flags().String("json", "", "JSON body for CollateralAdjust (required)")
	collateralCmd.MarkFlagRequired("json")

	Cmd.AddCommand(currenciesCmd, quotaCmd, ltvCmd, currentRateCmd, fixRateCmd,
		ordersCmd, orderCmd, createCmd, repayCmd, repayRecordsCmd, recordsCmd, collateralCmd)
}

func runCurrencies(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.MultiCollateralLoanAPI.ListMultiCollateralCurrencies(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/loan/multi_collateral/currencies", ""))
		return nil
	}
	return p.Print(result)
}

func runQuota(cmd *cobra.Command, args []string) error {
	typ, _ := cmd.Flags().GetString("type")
	currency, _ := cmd.Flags().GetString("currency")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.MultiCollateralLoanAPI.ListUserCurrencyQuota(c.Context(), typ, currency)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/loan/multi_collateral/currency_quota", ""))
		return nil
	}
	return p.Print(result)
}

func runLtv(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.MultiCollateralLoanAPI.GetMultiCollateralLtv(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/loan/multi_collateral/ltv", ""))
		return nil
	}
	return p.Print(result)
}

func runCurrentRate(cmd *cobra.Command, args []string) error {
	currenciesStr, _ := cmd.Flags().GetString("currencies")
	vipLevel, _ := cmd.Flags().GetString("vip-level")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	currencies := strings.Split(currenciesStr, ",")

	opts := &gateapi.GetMultiCollateralCurrentRateOpts{}
	if vipLevel != "" {
		opts.VipLevel = optional.NewString(vipLevel)
	}

	result, httpResp, err := c.MultiCollateralLoanAPI.GetMultiCollateralCurrentRate(c.Context(), currencies, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/loan/multi_collateral/current_rate", ""))
		return nil
	}
	return p.Print(result)
}

func runFixRate(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.MultiCollateralLoanAPI.GetMultiCollateralFixRate(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/loan/multi_collateral/fixed_rate", ""))
		return nil
	}
	return p.Print(result)
}

func runOrders(cmd *cobra.Command, args []string) error {
	page, _ := cmd.Flags().GetInt32("page")
	limit, _ := cmd.Flags().GetInt32("limit")
	sort, _ := cmd.Flags().GetString("sort")
	orderType, _ := cmd.Flags().GetString("order-type")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListMultiCollateralOrdersOpts{}
	if page > 0 {
		opts.Page = optional.NewInt32(page)
	}
	if limit > 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if sort != "" {
		opts.Sort = optional.NewString(sort)
	}
	if orderType != "" {
		opts.OrderType = optional.NewString(orderType)
	}

	result, httpResp, err := c.MultiCollateralLoanAPI.ListMultiCollateralOrders(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/loan/multi_collateral/orders", ""))
		return nil
	}
	return p.Print(result)
}

func runOrder(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetString("id")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.MultiCollateralLoanAPI.GetMultiCollateralOrderDetail(c.Context(), id)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", fmt.Sprintf("/api/v4/loan/multi_collateral/orders/%s", id), ""))
		return nil
	}
	return p.Print(result)
}

func runCreate(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.CreateMultiCollateralOrder
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}
	result, httpResp, err := c.MultiCollateralLoanAPI.CreateMultiCollateral(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/loan/multi_collateral/orders", jsonStr))
		return nil
	}
	return p.Print(result)
}

func runRepay(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.RepayMultiLoan
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}
	result, httpResp, err := c.MultiCollateralLoanAPI.RepayMultiCollateralLoan(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/loan/multi_collateral/repay", jsonStr))
		return nil
	}
	return p.Print(result)
}

func runRepayRecords(cmd *cobra.Command, args []string) error {
	typ, _ := cmd.Flags().GetString("type")
	borrowCurrency, _ := cmd.Flags().GetString("borrow-currency")
	page, _ := cmd.Flags().GetInt32("page")
	limit, _ := cmd.Flags().GetInt32("limit")
	from, _ := cmd.Flags().GetInt64("from")
	to, _ := cmd.Flags().GetInt64("to")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListMultiRepayRecordsOpts{}
	if borrowCurrency != "" {
		opts.BorrowCurrency = optional.NewString(borrowCurrency)
	}
	if page > 0 {
		opts.Page = optional.NewInt32(page)
	}
	if limit > 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if from != 0 {
		opts.From = optional.NewInt64(from)
	}
	if to != 0 {
		opts.To = optional.NewInt64(to)
	}

	result, httpResp, err := c.MultiCollateralLoanAPI.ListMultiRepayRecords(c.Context(), typ, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/loan/multi_collateral/repay", ""))
		return nil
	}
	return p.Print(result)
}

func runRecords(cmd *cobra.Command, args []string) error {
	page, _ := cmd.Flags().GetInt32("page")
	limit, _ := cmd.Flags().GetInt32("limit")
	from, _ := cmd.Flags().GetInt64("from")
	to, _ := cmd.Flags().GetInt64("to")
	collateralCurrency, _ := cmd.Flags().GetString("collateral-currency")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListMultiCollateralRecordsOpts{}
	if page > 0 {
		opts.Page = optional.NewInt32(page)
	}
	if limit > 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if from != 0 {
		opts.From = optional.NewInt64(from)
	}
	if to != 0 {
		opts.To = optional.NewInt64(to)
	}
	if collateralCurrency != "" {
		opts.CollateralCurrency = optional.NewString(collateralCurrency)
	}

	result, httpResp, err := c.MultiCollateralLoanAPI.ListMultiCollateralRecords(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/loan/multi_collateral/mortgage", ""))
		return nil
	}
	return p.Print(result)
}

func runCollateral(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.CollateralAdjust
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}
	result, httpResp, err := c.MultiCollateralLoanAPI.OperateMultiCollateral(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/loan/multi_collateral/mortgage", jsonStr))
		return nil
	}
	return p.Print(result)
}
