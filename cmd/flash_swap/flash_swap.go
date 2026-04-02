package flashswap

import (
	"encoding/json"
	"fmt"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

// Cmd is the root command for the flash-swap module.
var Cmd = &cobra.Command{
	Use:   "flash-swap",
	Short: "Flash swap commands",
}

func init() {
	pairsCmd := &cobra.Command{
		Use:   "pairs",
		Short: "List all supported currency pairs in flash swap (public)",
		RunE:  runPairs,
	}
	pairsCmd.Flags().String("currency", "", "Filter by currency name")
	pairsCmd.Flags().Int32("page", 1, "Page number")
	pairsCmd.Flags().Int32("limit", 0, "Maximum number of items returned")

	ordersCmd := &cobra.Command{
		Use:   "orders",
		Short: "List flash swap orders",
		RunE:  runOrders,
	}
	ordersCmd.Flags().Int32("status", 0, "Order status: 1=success, 2=failed")
	ordersCmd.Flags().String("sell-currency", "", "Currency sold")
	ordersCmd.Flags().String("buy-currency", "", "Currency bought")
	ordersCmd.Flags().Int32("limit", 0, "Maximum number of records")
	ordersCmd.Flags().Int32("page", 1, "Page number")

	orderCmd := &cobra.Command{
		Use:   "order",
		Short: "Get a flash swap order by ID",
		RunE:  runOrder,
	}
	orderCmd.Flags().Int32("id", 0, "Order ID (required)")
	orderCmd.MarkFlagRequired("id")

	previewCmd := &cobra.Command{
		Use:   "preview",
		Short: "Preview a flash swap order",
		RunE:  runPreview,
	}
	previewCmd.Flags().String("sell-currency", "", "Currency to sell (required)")
	previewCmd.Flags().String("buy-currency", "", "Currency to buy (required)")
	previewCmd.Flags().String("sell-amount", "", "Amount to sell (either sell-amount or buy-amount)")
	previewCmd.Flags().String("buy-amount", "", "Amount to buy (either sell-amount or buy-amount)")
	previewCmd.MarkFlagRequired("sell-currency")
	previewCmd.MarkFlagRequired("buy-currency")

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a flash swap order",
		RunE:  runCreate,
	}
	createCmd.Flags().String("preview-id", "", "Preview result ID (required)")
	createCmd.Flags().String("sell-currency", "", "Currency to sell (required)")
	createCmd.Flags().String("sell-amount", "", "Amount to sell (required)")
	createCmd.Flags().String("buy-currency", "", "Currency to buy (required)")
	createCmd.Flags().String("buy-amount", "", "Amount to buy (required)")
	createCmd.MarkFlagRequired("preview-id")
	createCmd.MarkFlagRequired("sell-currency")
	createCmd.MarkFlagRequired("sell-amount")
	createCmd.MarkFlagRequired("buy-currency")
	createCmd.MarkFlagRequired("buy-amount")

	previewV1Cmd := &cobra.Command{
		Use:   "preview-v1",
		Short: "Preview a flash swap order (v1, one-to-one)",
		RunE:  runPreviewV1,
	}
	previewV1Cmd.Flags().String("sell-asset", "", "Currency to sell (required)")
	previewV1Cmd.Flags().String("buy-asset", "", "Currency to buy (required)")
	previewV1Cmd.Flags().String("sell-amount", "", "Sell amount (either sell-amount or buy-amount)")
	previewV1Cmd.Flags().String("buy-amount", "", "Buy amount (either sell-amount or buy-amount)")
	previewV1Cmd.MarkFlagRequired("sell-asset")
	previewV1Cmd.MarkFlagRequired("buy-asset")

	createV1Cmd := &cobra.Command{
		Use:   "create-v1",
		Short: "Create a flash swap order (v1)",
		RunE:  runCreateV1,
	}
	createV1Cmd.Flags().String("json", "", "JSON body for FlashSwapOrderCreateReq (required)")
	createV1Cmd.MarkFlagRequired("json")

	previewManyToOneCmd := &cobra.Command{
		Use:   "preview-many-to-one",
		Short: "Preview a multi-currency many-to-one flash swap order",
		RunE:  runPreviewManyToOne,
	}
	previewManyToOneCmd.Flags().String("json", "", "JSON body for preview params (required)")
	previewManyToOneCmd.MarkFlagRequired("json")

	createManyToOneCmd := &cobra.Command{
		Use:   "create-many-to-one",
		Short: "Create a multi-currency many-to-one flash swap order",
		RunE:  runCreateManyToOne,
	}
	createManyToOneCmd.Flags().String("json", "", "JSON body for create params (required)")
	createManyToOneCmd.MarkFlagRequired("json")

	previewOneToManyCmd := &cobra.Command{
		Use:   "preview-one-to-many",
		Short: "Preview a multi-currency one-to-many flash swap order",
		RunE:  runPreviewOneToMany,
	}
	previewOneToManyCmd.Flags().String("json", "", "JSON body for preview params (required)")
	previewOneToManyCmd.MarkFlagRequired("json")

	createOneToManyCmd := &cobra.Command{
		Use:   "create-one-to-many",
		Short: "Create a multi-currency one-to-many flash swap order",
		RunE:  runCreateOneToMany,
	}
	createOneToManyCmd.Flags().String("json", "", "JSON body for create params (required)")
	createOneToManyCmd.MarkFlagRequired("json")

	Cmd.AddCommand(pairsCmd, ordersCmd, orderCmd, previewCmd, createCmd,
		previewV1Cmd, createV1Cmd, previewManyToOneCmd, createManyToOneCmd,
		previewOneToManyCmd, createOneToManyCmd)
}

func runPairs(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	page, _ := cmd.Flags().GetInt32("page")
	limit, _ := cmd.Flags().GetInt32("limit")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.ListFlashSwapCurrencyPairOpts{}
	if currency != "" {
		opts.Currency = optional.NewString(currency)
	}
	if page > 0 {
		opts.Page = optional.NewInt32(page)
	}
	if limit > 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.FlashSwapAPI.ListFlashSwapCurrencyPair(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/flash_swap/currency_pairs", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, cp := range result {
		rows[i] = []string{cp.CurrencyPair, cp.SellCurrency, cp.BuyCurrency,
			cp.SellMinAmount, cp.SellMaxAmount, cp.BuyMinAmount, cp.BuyMaxAmount}
	}
	return p.Table([]string{"Pair", "Sell", "Buy", "Sell Min", "Sell Max", "Buy Min", "Buy Max"}, rows)
}

func runOrders(cmd *cobra.Command, args []string) error {
	status, _ := cmd.Flags().GetInt32("status")
	sellCurrency, _ := cmd.Flags().GetString("sell-currency")
	buyCurrency, _ := cmd.Flags().GetString("buy-currency")
	limit, _ := cmd.Flags().GetInt32("limit")
	page, _ := cmd.Flags().GetInt32("page")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListFlashSwapOrdersOpts{}
	if status != 0 {
		opts.Status = optional.NewInt32(status)
	}
	if sellCurrency != "" {
		opts.SellCurrency = optional.NewString(sellCurrency)
	}
	if buyCurrency != "" {
		opts.BuyCurrency = optional.NewString(buyCurrency)
	}
	if limit > 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if page > 0 {
		opts.Page = optional.NewInt32(page)
	}

	result, httpResp, err := c.FlashSwapAPI.ListFlashSwapOrders(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/flash_swap/orders", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, o := range result {
		rows[i] = []string{fmt.Sprintf("%d", o.Id), o.SellCurrency, o.SellAmount,
			o.BuyCurrency, o.BuyAmount, o.Price, fmt.Sprintf("%d", o.Status)}
	}
	return p.Table([]string{"ID", "Sell", "Sell Amt", "Buy", "Buy Amt", "Price", "Status"}, rows)
}

func runOrder(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetInt32("id")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.FlashSwapAPI.GetFlashSwapOrder(c.Context(), id)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", fmt.Sprintf("/api/v4/flash_swap/orders/%d", id), ""))
		return nil
	}
	return p.Print(result)
}

func runPreview(cmd *cobra.Command, args []string) error {
	sellCurrency, _ := cmd.Flags().GetString("sell-currency")
	buyCurrency, _ := cmd.Flags().GetString("buy-currency")
	sellAmount, _ := cmd.Flags().GetString("sell-amount")
	buyAmount, _ := cmd.Flags().GetString("buy-amount")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body := gateapi.FlashSwapPreviewRequest{
		SellCurrency: sellCurrency,
		BuyCurrency:  buyCurrency,
		SellAmount:   sellAmount,
		BuyAmount:    buyAmount,
	}
	reqBody, _ := json.Marshal(body)
	result, httpResp, err := c.FlashSwapAPI.PreviewFlashSwapOrder(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/flash_swap/orders/preview", string(reqBody)))
		return nil
	}
	return p.Print(result)
}

func runCreate(cmd *cobra.Command, args []string) error {
	previewID, _ := cmd.Flags().GetString("preview-id")
	sellCurrency, _ := cmd.Flags().GetString("sell-currency")
	sellAmount, _ := cmd.Flags().GetString("sell-amount")
	buyCurrency, _ := cmd.Flags().GetString("buy-currency")
	buyAmount, _ := cmd.Flags().GetString("buy-amount")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body := gateapi.FlashSwapOrderRequest{
		PreviewId:    previewID,
		SellCurrency: sellCurrency,
		SellAmount:   sellAmount,
		BuyCurrency:  buyCurrency,
		BuyAmount:    buyAmount,
	}
	reqBody, _ := json.Marshal(body)
	result, httpResp, err := c.FlashSwapAPI.CreateFlashSwapOrder(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/flash_swap/orders", string(reqBody)))
		return nil
	}
	return p.Print(result)
}

func runPreviewV1(cmd *cobra.Command, args []string) error {
	sellAsset, _ := cmd.Flags().GetString("sell-asset")
	buyAsset, _ := cmd.Flags().GetString("buy-asset")
	sellAmount, _ := cmd.Flags().GetString("sell-amount")
	buyAmount, _ := cmd.Flags().GetString("buy-amount")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.PreviewFlashSwapOrderV1Opts{}
	if sellAmount != "" {
		opts.SellAmount = optional.NewString(sellAmount)
	}
	if buyAmount != "" {
		opts.BuyAmount = optional.NewString(buyAmount)
	}

	result, httpResp, err := c.FlashSwapAPI.PreviewFlashSwapOrderV1(c.Context(), sellAsset, buyAsset, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/flash_swap/orders/preview_v1", ""))
		return nil
	}
	return p.Print(result)
}

func runCreateV1(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.FlashSwapOrderCreateReq
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}
	result, httpResp, err := c.FlashSwapAPI.CreateFlashSwapOrderV1(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/flash_swap/orders_v1", jsonStr))
		return nil
	}
	return p.Print(result)
}

func runPreviewManyToOne(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.FlashSwapMultiCurrencyManyToOneOrderPreviewReq
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}
	result, httpResp, err := c.FlashSwapAPI.PreviewFlashSwapMultiCurrencyManyToOneOrder(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/flash_swap/multi_currency/many_to_one/preview", jsonStr))
		return nil
	}
	return p.Print(result)
}

func runCreateManyToOne(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.FlashSwapMultiCurrencyManyToOneOrderCreateReq
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}
	result, httpResp, err := c.FlashSwapAPI.CreateFlashSwapMultiCurrencyManyToOneOrder(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/flash_swap/multi_currency/many_to_one", jsonStr))
		return nil
	}
	return p.Print(result)
}

func runPreviewOneToMany(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.FlashSwapMultiCurrencyOneToManyOrderPreviewReq
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}
	result, httpResp, err := c.FlashSwapAPI.PreviewFlashSwapMultiCurrencyOneToManyOrder(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/flash_swap/multi_currency/one_to_many/preview", jsonStr))
		return nil
	}
	return p.Print(result)
}

func runCreateOneToMany(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.FlashSwapMultiCurrencyOneToManyOrderCreateReq
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}
	result, httpResp, err := c.FlashSwapAPI.CreateFlashSwapMultiCurrencyOneToManyOrder(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/flash_swap/multi_currency/one_to_many", jsonStr))
		return nil
	}
	return p.Print(result)
}
