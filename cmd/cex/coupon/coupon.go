package coupon

import (
	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

// Cmd is the root command for the coupon module.
var Cmd = &cobra.Command{
	Use:   "coupon",
	Short: "Coupon commands",
}

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List user coupons",
		RunE:  runList,
	}
	listCmd.Flags().Int32("expired", -1, "0=valid, 1=expired/used")
	listCmd.Flags().Int32("limit", 0, "Items per page (1-20)")
	listCmd.Flags().Int32("last-id", 0, "Cursor: last record ID from previous page")
	listCmd.Flags().String("order-by", "", "Sort: latest or expired")
	listCmd.Flags().String("type", "", "Coupon type filter")
	listCmd.Flags().Int32("is-task-coupon", -1, "0=regular, 1=task coupons")

	detailCmd := &cobra.Command{
		Use:   "detail",
		Short: "Get coupon detail",
		RunE:  runDetail,
	}
	detailCmd.Flags().String("coupon-type", "", "Coupon type (required)")
	detailCmd.Flags().Int32("detail-id", 0, "Detail ID (required)")
	detailCmd.Flags().Int32("is-task-coupon", -1, "0=regular, 1=task coupon")
	detailCmd.MarkFlagRequired("coupon-type")
	detailCmd.MarkFlagRequired("detail-id")

	Cmd.AddCommand(listCmd, detailCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListUserCouponsOpts{}
	if v, _ := cmd.Flags().GetInt32("expired"); v >= 0 {
		opts.Expired = optional.NewInt32(v)
	}
	if v, _ := cmd.Flags().GetInt32("limit"); v > 0 {
		opts.Limit = optional.NewInt32(v)
	}
	if v, _ := cmd.Flags().GetInt32("last-id"); v > 0 {
		opts.LastId = optional.NewInt32(v)
	}
	if v, _ := cmd.Flags().GetString("order-by"); v != "" {
		opts.OrderBy = optional.NewString(v)
	}
	if v, _ := cmd.Flags().GetString("type"); v != "" {
		opts.Type_ = optional.NewString(v)
	}
	if v, _ := cmd.Flags().GetInt32("is-task-coupon"); v >= 0 {
		opts.IsTaskCoupon = optional.NewInt32(v)
	}

	result, httpResp, err := c.CouponAPI.ListUserCoupons(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/coupon/user-coupon-list", ""))
		return nil
	}
	return p.Print(result)
}

func runDetail(cmd *cobra.Command, args []string) error {
	couponType, _ := cmd.Flags().GetString("coupon-type")
	detailID, _ := cmd.Flags().GetInt32("detail-id")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.GetUserCouponDetailOpts{}
	if v, _ := cmd.Flags().GetInt32("is-task-coupon"); v >= 0 {
		opts.IsTaskCoupon = optional.NewInt32(v)
	}

	result, httpResp, err := c.CouponAPI.GetUserCouponDetail(c.Context(), couponType, detailID, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/coupon/user-coupon-detail", ""))
		return nil
	}
	return p.Print(result)
}
