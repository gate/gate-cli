package earn

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	"github.com/gate/gate-cli/internal/output"
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
	productsCmd.Flags().Int32("status", 0, "Optional product status filter; 2=subscribable")
	productsCmd.Flags().String("show-status", "", "Optional display status filter; 2=visible")

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
	addFixedFilterFlags(lendsCmd)

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
	addFixedFilterFlags(historyCmd)

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
	if asset != "" || cmd.Flags().Changed("type") || cmd.Flags().Changed("status") || cmd.Flags().Changed("show-status") {
		opts = &gateapi.ListEarnFixedTermProductsOpts{}
		if asset != "" {
			opts.Asset = optional.NewString(asset)
		}
		if cmd.Flags().Changed("type") {
			opts.Type_ = optional.NewInt32(typ)
		}
		// Presence matters: omission preserves upstream defaults, while explicit
		// zero or an empty string must still reach the SDK as supplied.
		if cmd.Flags().Changed("status") {
			status, _ := cmd.Flags().GetInt32("status")
			opts.Status = optional.NewInt32(status)
		}
		if cmd.Flags().Changed("show-status") {
			showStatus, _ := cmd.Flags().GetString("show-status")
			opts.ShowStatus = optional.NewString(showStatus)
		}
	}

	result, httpResp, err := c.EarnAPI.ListEarnFixedTermProducts(c.Context(), page, limit, opts)
	if err != nil {
		// 服务端拒绝必须传播为命令失败，避免Agentic鉴权错误被自动化当作执行成功。
		return client.CommandError(err, httpResp, "GET", "/api/v4/earn/fixed-term/product", "")
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
		return client.CommandError(err, httpResp, "GET", "/api/v4/earn/fixed-term/product/"+currency+"/list", "")
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
	subBusiness, businessFilter, err := readFixedFilterFlags(cmd)
	if err != nil {
		return err
	}
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListEarnFixedTermLendsOpts{SubBusiness: subBusiness, BusinessFilter: businessFilter}
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
		return client.CommandError(err, httpResp, "GET", "/api/v4/earn/fixed-term/user/lend", "")
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
		return client.CommandError(err, httpResp, "POST", "/api/v4/earn/fixed-term/user/lend", jsonStr)
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
		return client.CommandError(err, httpResp, "POST", "/api/v4/earn/fixed-term/user/pre-redeem", jsonStr)
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
	subBusiness, businessFilter, err := readFixedFilterFlags(cmd)
	if err != nil {
		return err
	}
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListEarnFixedTermHistoryOpts{SubBusiness: subBusiness, BusinessFilter: businessFilter}
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
		return client.CommandError(err, httpResp, "GET", "/api/v4/earn/fixed-term/history", "")
	}
	return p.Print(result)
}

// These SDK query fields belong to both fixed-term read operations; keep their
// CLI spelling and omission semantics in one place instead of divergent handlers.
func addFixedFilterFlags(cmd *cobra.Command) {
	cmd.Flags().Int32("sub-business", 0, "Sub-business filter (omitted unless explicitly set; 0 is supported)")
	cmd.Flags().String("business-filter", "", `Business filter JSON array of objects, e.g. [{"business":1,"sub_business":0}] (maximum 65536 bytes)`)
}

// Match the existing CLI JSON input bound without adding a runtime configuration.
const maxFixedBusinessFilterBytes = 65536

// readFixedFilterFlags validates before credential/client access. Explicit zero
// must remain set in the SDK optional value, while omitted flags retain old queries.
func readFixedFilterFlags(cmd *cobra.Command) (optional.Int32, optional.String, error) {
	var subBusiness optional.Int32
	var businessFilter optional.String
	if cmd.Flags().Changed("sub-business") {
		value, err := cmd.Flags().GetInt32("sub-business")
		if err != nil {
			return subBusiness, businessFilter, err
		}
		subBusiness = optional.NewInt32(value)
	}
	if cmd.Flags().Changed("business-filter") {
		raw, err := cmd.Flags().GetString("business-filter")
		if err != nil {
			return subBusiness, businessFilter, err
		}
		if len(raw) > maxFixedBusinessFilterBytes {
			return subBusiness, businessFilter, output.InvalidArgsError("--business-filter exceeds 65536 bytes")
		}
		var items []json.RawMessage
		if err := json.Unmarshal([]byte(raw), &items); err != nil || items == nil {
			return subBusiness, businessFilter, output.InvalidArgsError("--business-filter must be a JSON array of objects")
		}
		for _, item := range items {
			value := bytes.TrimSpace(item)
			if len(value) == 0 || value[0] != '{' {
				return subBusiness, businessFilter, output.InvalidArgsError("--business-filter must contain JSON objects")
			}
		}
		// Preserve caller bytes and SDK-defined business semantics; the API owns the
		// business/sub_business combinations, not a second CLI enum or JSON encoder.
		businessFilter = optional.NewString(raw)
	}
	return subBusiness, businessFilter, nil
}
