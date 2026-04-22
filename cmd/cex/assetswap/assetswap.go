// Package assetswap exposes Gate's Portfolio Optimization (asset-swap) APIs
// via the `gate-cli cex assetswap ...` command group. Introduced when syncing
// with gateapi-go/v7 v7.2.71 to close the CLI gap against the assetswap MCP
// tool set.
package assetswap

import (
	"encoding/json"
	"fmt"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

// Cmd is the root command for the asset-swap module.
var Cmd = &cobra.Command{
	Use:   "assetswap",
	Short: "Asset-swap (portfolio optimization) commands",
}

func init() {
	assetsCmd := &cobra.Command{
		Use:   "assets",
		Short: "List supported asset-swap assets (public, no auth required)",
		RunE:  runAssets,
	}

	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Get asset-swap config and recommended strategies (public, no auth required)",
		RunE:  runConfig,
	}

	evaluateCmd := &cobra.Command{
		Use:   "evaluate",
		Short: "Evaluate user portfolio for asset-swap (auth required)",
		RunE:  runEvaluate,
	}
	evaluateCmd.Flags().Int32("max-value", 0, "Maximum evaluate value")
	evaluateCmd.Flags().String("cursor", "", "Pagination cursor")
	evaluateCmd.Flags().Int32("size", 0, "Page size")

	orderCmd := &cobra.Command{
		Use:   "order",
		Short: "Asset-swap order commands",
	}

	orderCreateCmd := &cobra.Command{
		Use:   "create",
		Short: "Create an asset-swap order",
		RunE:  runOrderCreate,
	}
	orderCreateCmd.Flags().String("json", "", `JSON body: {"from":[{"asset":"BTC","amount":"0.1"}],"to":[{"asset":"USDT","amount":"10000"}]} (required)`)
	orderCreateCmd.MarkFlagRequired("json")

	orderPreviewCmd := &cobra.Command{
		Use:   "preview",
		Short: "Preview an asset-swap order (auth required)",
		RunE:  runOrderPreview,
	}
	orderPreviewCmd.Flags().String("json", "", `JSON body: {"from":[{"asset":"BTC","amount":"0.1"}],"to":[{"asset":"USDT","ratio":"0.5"}]} (required)`)
	orderPreviewCmd.MarkFlagRequired("json")

	orderListCmd := &cobra.Command{
		Use:   "list",
		Short: "List asset-swap orders (auth required)",
		RunE:  runOrderList,
	}
	orderListCmd.Flags().Int32("from", 0, "Start time")
	orderListCmd.Flags().Int32("to", 0, "End time")
	orderListCmd.Flags().Int32("status", 0, "Order status")
	orderListCmd.Flags().Int32("offset", 0, "Pagination offset")
	orderListCmd.Flags().Int32("size", 0, "Page size")
	orderListCmd.Flags().Int32("sort-mode", 0, "Sort mode")
	orderListCmd.Flags().Int32("order-by", 0, "Order by")

	orderGetCmd := &cobra.Command{
		Use:   "get <order-id>",
		Short: "Get asset-swap order detail by ID (auth required)",
		Args:  cobra.ExactArgs(1),
		RunE:  runOrderGet,
	}

	orderCmd.AddCommand(orderCreateCmd, orderPreviewCmd, orderListCmd, orderGetCmd)
	Cmd.AddCommand(assetsCmd, configCmd, evaluateCmd, orderCmd)
}

func runAssets(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.AssetswapAPI.ListAssetSwapAssets(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/asset-swap/asset/list", ""))
		return nil
	}
	return p.Print(result)
}

func runConfig(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.AssetswapAPI.GetAssetSwapConfig(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/asset-swap/config", ""))
		return nil
	}
	return p.Print(result)
}

func runEvaluate(cmd *cobra.Command, args []string) error {
	maxValue, _ := cmd.Flags().GetInt32("max-value")
	cursor, _ := cmd.Flags().GetString("cursor")
	size, _ := cmd.Flags().GetInt32("size")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.EvaluateAssetSwapOpts{}
	if maxValue != 0 {
		opts.MaxEvaluateValue = optional.NewInt32(maxValue)
	}
	if cursor != "" {
		opts.Cursor = optional.NewString(cursor)
	}
	if size != 0 {
		opts.Size = optional.NewInt32(size)
	}

	result, httpResp, err := c.AssetswapAPI.EvaluateAssetSwap(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/asset-swap/evaluate", ""))
		return nil
	}
	return p.Print(result)
}

func runOrderCreate(cmd *cobra.Command, args []string) error {
	rawJSON, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.OrderCreateV1Req
	if err := json.Unmarshal([]byte(rawJSON), &body); err != nil {
		return fmt.Errorf("invalid --json body: %w", err)
	}

	result, httpResp, err := c.AssetswapAPI.CreateAssetSwapOrderV1(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/asset-swap/order/v1", rawJSON))
		return nil
	}
	return p.Print(result)
}

func runOrderPreview(cmd *cobra.Command, args []string) error {
	rawJSON, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.OrderPreviewV1Req
	if err := json.Unmarshal([]byte(rawJSON), &body); err != nil {
		return fmt.Errorf("invalid --json body: %w", err)
	}

	result, httpResp, err := c.AssetswapAPI.PreviewAssetSwapOrderV1(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/asset-swap/order/preview/v1", rawJSON))
		return nil
	}
	return p.Print(result)
}

func runOrderList(cmd *cobra.Command, args []string) error {
	from, _ := cmd.Flags().GetInt32("from")
	to, _ := cmd.Flags().GetInt32("to")
	status, _ := cmd.Flags().GetInt32("status")
	offset, _ := cmd.Flags().GetInt32("offset")
	size, _ := cmd.Flags().GetInt32("size")
	sortMode, _ := cmd.Flags().GetInt32("sort-mode")
	orderBy, _ := cmd.Flags().GetInt32("order-by")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListAssetSwapOrdersV1Opts{}
	if from != 0 {
		opts.From = optional.NewInt32(from)
	}
	if to != 0 {
		opts.To = optional.NewInt32(to)
	}
	if status != 0 {
		opts.Status = optional.NewInt32(status)
	}
	if offset != 0 {
		opts.Offset = optional.NewInt32(offset)
	}
	if size != 0 {
		opts.Size = optional.NewInt32(size)
	}
	if sortMode != 0 {
		opts.SortMode = optional.NewInt32(sortMode)
	}
	if orderBy != 0 {
		opts.OrderBy = optional.NewInt32(orderBy)
	}

	result, httpResp, err := c.AssetswapAPI.ListAssetSwapOrdersV1(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/asset-swap/order/list/v1", ""))
		return nil
	}
	return p.Print(result)
}

func runOrderGet(cmd *cobra.Command, args []string) error {
	orderID := args[0]
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.AssetswapAPI.GetAssetSwapOrderV1(c.Context(), orderID)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/asset-swap/order/detail/v1/"+orderID, ""))
		return nil
	}
	return p.Print(result)
}
