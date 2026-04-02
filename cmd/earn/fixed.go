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

var fixedCmd = &cobra.Command{
	Use:   "fixed",
	Short: "Fixed-term earn commands",
}

func init() {
	productsCmd := &cobra.Command{
		Use:   "products",
		Short: "List fixed-term products (public, no auth required)",
		RunE:  runFixedProducts,
	}
	productsCmd.Flags().Int32("page", 1, "Page number")
	productsCmd.Flags().Int32("limit", 10, "Page size")
	productsCmd.Flags().String("asset", "", "Filter by currency")
	productsCmd.Flags().Int32("type", 0, "Product type: 1=regular, 2=VIP")

	productsAssetCmd := &cobra.Command{
		Use:   "products-asset",
		Short: "List fixed-term products by single currency (public, no auth required)",
		RunE:  runFixedProductsByAsset,
	}
	productsAssetCmd.Flags().String("currency", "", "Currency name, e.g. USDT, BTC (required)")
	productsAssetCmd.Flags().String("type", "", "Product type: 1=regular, 2=VIP, 0=all")
	productsAssetCmd.MarkFlagRequired("currency")

	lendsCmd := &cobra.Command{
		Use:   "lends",
		Short: "List fixed-term subscription orders",
		RunE:  runFixedLends,
	}
	lendsCmd.Flags().String("order-type", "1", "Order type: 1=current, 2=historical")
	lendsCmd.Flags().Int32("page", 1, "Page number")
	lendsCmd.Flags().Int32("limit", 10, "Page size")
	lendsCmd.Flags().Int32("product-id", 0, "Product ID")
	lendsCmd.Flags().Int64("order-id", 0, "Order ID")
	lendsCmd.Flags().String("asset", "", "Currency")

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Subscribe to a fixed-term product",
		RunE:  runFixedCreate,
	}
	createCmd.Flags().String("json", "", "JSON body for subscription request (required)")
	createCmd.MarkFlagRequired("json")

	preRedeemCmd := &cobra.Command{
		Use:   "pre-redeem",
		Short: "Early redeem a fixed-term order",
		RunE:  runFixedPreRedeem,
	}
	preRedeemCmd.Flags().String("json", "", "JSON body for pre-redeem request (required)")
	preRedeemCmd.MarkFlagRequired("json")

	historyCmd := &cobra.Command{
		Use:   "history",
		Short: "List fixed-term subscription history",
		RunE:  runFixedHistory,
	}
	historyCmd.Flags().String("type", "1", "Type: 1=subscription, 2=redemption, 3=interest, 4=bonus")
	historyCmd.Flags().Int32("page", 1, "Page number")
	historyCmd.Flags().Int32("limit", 10, "Page size")
	historyCmd.Flags().Int32("product-id", 0, "Product ID")
	historyCmd.Flags().String("order-id", "", "Order ID")
	historyCmd.Flags().String("asset", "", "Currency")
	historyCmd.Flags().Int32("start-at", 0, "Start timestamp")
	historyCmd.Flags().Int32("end-at", 0, "End timestamp")

	fixedCmd.AddCommand(productsCmd, productsAssetCmd, lendsCmd, createCmd, preRedeemCmd, historyCmd)
}

func runFixedProducts(cmd *cobra.Command, args []string) error {
	page, _ := cmd.Flags().GetInt32("page")
	limit, _ := cmd.Flags().GetInt32("limit")
	asset, _ := cmd.Flags().GetString("asset")
	typ, _ := cmd.Flags().GetInt32("type")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	var opts *gateapi.ListEarnFixedTermProductsOpts
	if asset != "" || cmd.Flags().Changed("type") {
		opts = &gateapi.ListEarnFixedTermProductsOpts{}
		if asset != "" {
			opts.Asset = optional.NewString(asset)
		}
		if cmd.Flags().Changed("type") {
			opts.Type_ = optional.NewInt32(typ)
		}
	}

	result, httpResp, err := c.EarnAPI.ListEarnFixedTermProducts(c.Context(), page, limit, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/earn/fixed-term/product", ""))
		return nil
	}
	return p.Print(result)
}

func runFixedProductsByAsset(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	typ, _ := cmd.Flags().GetString("type")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	var opts *gateapi.ListEarnFixedTermProductsByAssetOpts
	if typ != "" {
		opts = &gateapi.ListEarnFixedTermProductsByAssetOpts{
			Type_: optional.NewString(typ),
		}
	}

	result, httpResp, err := c.EarnAPI.ListEarnFixedTermProductsByAsset(c.Context(), currency, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/earn/fixed-term/product/"+currency+"/list", ""))
		return nil
	}
	return p.Print(result)
}

func runFixedLends(cmd *cobra.Command, args []string) error {
	orderType, _ := cmd.Flags().GetString("order-type")
	page, _ := cmd.Flags().GetInt32("page")
	limit, _ := cmd.Flags().GetInt32("limit")
	productID, _ := cmd.Flags().GetInt32("product-id")
	orderID, _ := cmd.Flags().GetInt64("order-id")
	asset, _ := cmd.Flags().GetString("asset")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListEarnFixedTermLendsOpts{}
	if productID != 0 {
		opts.ProductId = optional.NewInt32(productID)
	}
	if orderID != 0 {
		opts.OrderId = optional.NewInt64(orderID)
	}
	if asset != "" {
		opts.Asset = optional.NewString(asset)
	}

	result, httpResp, err := c.EarnAPI.ListEarnFixedTermLends(c.Context(), orderType, page, limit, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/earn/fixed-term/user/lend", ""))
		return nil
	}
	return p.Print(result)
}

func runFixedCreate(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.FixedTermLendRequest
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}

	opts := &gateapi.CreateEarnFixedTermLendOpts{
		FixedTermLendRequest: optional.NewInterface(body),
	}
	result, httpResp, err := c.EarnAPI.CreateEarnFixedTermLend(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/earn/fixed-term/user/lend", jsonStr))
		return nil
	}
	return p.Print(result)
}

func runFixedPreRedeem(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.EarnFixedTermPreRedeemRequest
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}

	opts := &gateapi.CreateEarnFixedTermPreRedeemOpts{
		EarnFixedTermPreRedeemRequest: optional.NewInterface(body),
	}
	result, httpResp, err := c.EarnAPI.CreateEarnFixedTermPreRedeem(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/earn/fixed-term/user/pre-redeem", jsonStr))
		return nil
	}
	return p.Print(result)
}

func runFixedHistory(cmd *cobra.Command, args []string) error {
	typ, _ := cmd.Flags().GetString("type")
	page, _ := cmd.Flags().GetInt32("page")
	limit, _ := cmd.Flags().GetInt32("limit")
	productID, _ := cmd.Flags().GetInt32("product-id")
	orderID, _ := cmd.Flags().GetString("order-id")
	asset, _ := cmd.Flags().GetString("asset")
	startAt, _ := cmd.Flags().GetInt32("start-at")
	endAt, _ := cmd.Flags().GetInt32("end-at")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListEarnFixedTermHistoryOpts{}
	if productID != 0 {
		opts.ProductId = optional.NewInt32(productID)
	}
	if orderID != "" {
		opts.OrderId = optional.NewString(orderID)
	}
	if asset != "" {
		opts.Asset = optional.NewString(asset)
	}
	if startAt != 0 {
		opts.StartAt = optional.NewInt32(startAt)
	}
	if endAt != 0 {
		opts.EndAt = optional.NewInt32(endAt)
	}

	result, httpResp, err := c.EarnAPI.ListEarnFixedTermHistory(c.Context(), typ, page, limit, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/earn/fixed-term/history", ""))
		return nil
	}
	return p.Print(result)
}
