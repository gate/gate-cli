package spot

import (
	"fmt"

	"github.com/antihax/optional"
	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/spf13/cobra"
)

var povCmd = &cobra.Command{Use: "pov", Short: "Spot participation-of-volume orders"}

// These commands use the shared SDK transport, authentication and error handling.
func init() {
	Cmd.AddCommand(povCmd)
	spotPovCreateCmd := &cobra.Command{Use: "create", Short: "Create a spot participation-of-volume order", Args: cobra.NoArgs, RunE: runSpotPovCreate}
	spotPovCreateCmd.Flags().String("pair", "", "Currency pair, e.g. BTC_USDT")
	cmdutil.MustRequireFlags(spotPovCreateCmd, "pair")
	spotPovCreateCmd.Flags().String("side", "", "buy or sell")
	cmdutil.MustRequireFlags(spotPovCreateCmd, "side")
	spotPovCreateCmd.Flags().String("amount", "", "Base currency amount as a decimal string")
	cmdutil.MustRequireFlags(spotPovCreateCmd, "amount")
	spotPovCreateCmd.Flags().Int32("participation-rate", 0, "Participation percentage: 5, 10, 20 or 40")
	cmdutil.MustRequireFlags(spotPovCreateCmd, "participation-rate")
	spotPovCreateCmd.Flags().String("ttl", "", "1h, 6h, 12h or 1d through 7d")
	cmdutil.MustRequireFlags(spotPovCreateCmd, "ttl")
	spotPovCreateCmd.Flags().String("limit-price", "", "Limit price; omit to use the market price")
	spotPovCreateCmd.Flags().String("trigger-price", "", "Trigger price; omit for immediate activation")
	spotPovCreateCmd.Flags().String("text", "", "Custom ID: t- followed by at most 28 letters, digits, _, - or .")
	povCmd.AddCommand(spotPovCreateCmd)
	spotPovListCmd := &cobra.Command{Use: "list", Short: "List spot POV orders", Args: cobra.NoArgs, RunE: runSpotPovList}
	spotPovListCmd.Flags().String("status", "open", "open or finished")
	spotPovListCmd.Flags().String("pair", "", "Currency pair, e.g. BTC_USDT")
	spotPovListCmd.Flags().String("side", "", "Filter by buy or sell")
	spotPovListCmd.Flags().Int32("page", 1, "Page number, 1 to 100")
	spotPovListCmd.Flags().Int32("limit", 20, "Records per page, 1 to 1000")
	povCmd.AddCommand(spotPovListCmd)
	spotPovGetCmd := &cobra.Command{Use: "get", Short: "Get a spot POV order", Args: cobra.NoArgs, RunE: runSpotPovGet}
	spotPovGetCmd.Flags().String("id", "", "Order ID")
	cmdutil.MustRequireFlags(spotPovGetCmd, "id")
	povCmd.AddCommand(spotPovGetCmd)
	spotPovCancelCmd := &cobra.Command{Use: "cancel", Short: "Cancel a spot POV order", Args: cobra.NoArgs, RunE: runSpotPovCancel}
	spotPovCancelCmd.Flags().String("id", "", "Order ID")
	cmdutil.MustRequireFlags(spotPovCancelCmd, "id")
	povCmd.AddCommand(spotPovCancelCmd)
	spotPovCancelAllCmd := &cobra.Command{Use: "cancel-all", Short: "Cancel all spot POV orders; omit pair for all pairs", Args: cobra.NoArgs, RunE: runSpotPovCancelAll}
	spotPovCancelAllCmd.Flags().String("pair", "", "Currency pair, e.g. BTC_USDT")
	povCmd.AddCommand(spotPovCancelAllCmd)
	spotRPIOrderBookCmd := &cobra.Command{Use: "rpi-orderbook", Short: "Get the RPI order book", Args: cobra.NoArgs, RunE: runSpotRPIOrderBook}
	spotRPIOrderBookCmd.Flags().String("pair", "", "Currency pair, e.g. BTC_USDT")
	cmdutil.MustRequireFlags(spotRPIOrderBookCmd, "pair")
	spotRPIOrderBookCmd.Flags().String("interval", "", "Price aggregation interval")
	spotRPIOrderBookCmd.Flags().Int32("limit", 20, "Depth limit, 1 to 100")
	spotRPIOrderBookCmd.Flags().Bool("with-id", false, "Include order book ID")
	marketCmd.AddCommand(spotRPIOrderBookCmd)
}

func runSpotPovCreate(cmd *cobra.Command, args []string) error {
	pair, _ := cmd.Flags().GetString("pair")
	side, _ := cmd.Flags().GetString("side")
	amount, _ := cmd.Flags().GetString("amount")
	participationRate, _ := cmd.Flags().GetInt32("participation-rate")
	ttl, _ := cmd.Flags().GetString("ttl")
	limitPrice, _ := cmd.Flags().GetString("limit-price")
	triggerPrice, _ := cmd.Flags().GetString("trigger-price")
	text, _ := cmd.Flags().GetString("text")
	if err := cmdutil.TradingToken("--pair", pair); err != nil {
		return err
	}
	if _, err := cmdutil.TradingDecimal("--amount", amount, true); err != nil {
		return err
	}
	if side != "buy" && side != "sell" {
		return fmt.Errorf("--side must be buy or sell")
	}
	if participationRate != 5 && participationRate != 10 && participationRate != 20 && participationRate != 40 {
		return fmt.Errorf("--participation-rate must be 5, 10, 20 or 40")
	}
	validTTL := map[string]bool{"1h": true, "6h": true, "12h": true, "1d": true, "2d": true, "3d": true, "4d": true, "5d": true, "6d": true, "7d": true}
	if !validTTL[ttl] {
		return fmt.Errorf("invalid --ttl")
	}
	if limitPrice != "" {
		if _, err := cmdutil.TradingDecimal("--limit-price", limitPrice, true); err != nil {
			return err
		}
	}
	if triggerPrice != "" {
		if _, err := cmdutil.TradingDecimal("--trigger-price", triggerPrice, true); err != nil {
			return err
		}
	}
	if text != "" {
		if len(text) < 3 || len(text) > 30 || text[:2] != "t-" {
			return fmt.Errorf("--text requires t- and 1 to 28 identifier characters")
		}
		if err := cmdutil.TradingToken("--text", text); err != nil {
			return err
		}
	}
	req := gateapi.SpotPovOrderCreator{CurrencyPair: pair, Side: side, Amount: amount, ParticipationRate: participationRate, Ttl: ttl, LimitPrice: limitPrice, TriggerPrice: triggerPrice, Text: text}

	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}
	result, resp, err := c.SpotAPI.CreateSpotPovOrder(c.Context(), req)
	if err != nil {
		return client.CommandError(err, resp, "POST", "/api/v4/spot/pov_orders", "")
	}
	return cmdutil.GetPrinter(cmd).Print(result)
}

func runSpotPovList(cmd *cobra.Command, args []string) error {
	status, _ := cmd.Flags().GetString("status")
	pair, _ := cmd.Flags().GetString("pair")
	side, _ := cmd.Flags().GetString("side")
	page, _ := cmd.Flags().GetInt32("page")
	limit, _ := cmd.Flags().GetInt32("limit")
	if status != "open" && status != "finished" {
		return fmt.Errorf("--status must be open or finished")
	}
	if side != "" && side != "buy" && side != "sell" {
		return fmt.Errorf("--side must be buy or sell")
	}
	if page < 1 || page > 100 {
		return fmt.Errorf("--page must be between 1 and 100")
	}
	if limit < 1 || limit > 1000 {
		return fmt.Errorf("--limit must be between 1 and 1000")
	}
	opts := &gateapi.ListSpotPovOrdersOpts{Page: optional.NewInt32(page), Limit: optional.NewInt32(limit)}
	if pair != "" {
		opts.CurrencyPair = optional.NewString(pair)
	}
	if side != "" {
		opts.Side = optional.NewString(side)
	}

	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}
	result, resp, err := c.SpotAPI.ListSpotPovOrders(c.Context(), status, opts)
	if err != nil {
		return client.CommandError(err, resp, "GET", "/api/v4/spot/pov_orders", "")
	}
	return cmdutil.GetPrinter(cmd).Print(result)
}

func runSpotPovGet(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetString("id")
	if err := cmdutil.TradingToken("--id", id); err != nil {
		return err
	}

	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}
	result, resp, err := c.SpotAPI.GetSpotPovOrder(c.Context(), id)
	if err != nil {
		return client.CommandError(err, resp, "GET", "/api/v4/spot/pov_orders/"+id, "")
	}
	return cmdutil.GetPrinter(cmd).Print(result)
}

func runSpotPovCancel(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetString("id")
	if err := cmdutil.TradingToken("--id", id); err != nil {
		return err
	}

	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}
	result, resp, err := c.SpotAPI.CancelSpotPovOrder(c.Context(), id)
	if err != nil {
		return client.CommandError(err, resp, "DELETE", "/api/v4/spot/pov_orders/"+id, "")
	}
	return cmdutil.GetPrinter(cmd).Print(result)
}

func runSpotPovCancelAll(cmd *cobra.Command, args []string) error {
	pair, _ := cmd.Flags().GetString("pair")
	opts := &gateapi.CancelSpotPovOrdersOpts{}
	if pair != "" {
		opts.CurrencyPair = optional.NewString(pair)
	}

	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}
	result, resp, err := c.SpotAPI.CancelSpotPovOrders(c.Context(), opts)
	if err != nil {
		return client.CommandError(err, resp, "DELETE", "/api/v4/spot/pov_orders", "")
	}
	return cmdutil.GetPrinter(cmd).Print(result)
}

func runSpotRPIOrderBook(cmd *cobra.Command, args []string) error {
	pair, _ := cmd.Flags().GetString("pair")
	interval, _ := cmd.Flags().GetString("interval")
	limit, _ := cmd.Flags().GetInt32("limit")
	withId, _ := cmd.Flags().GetBool("with-id")
	if err := cmdutil.TradingToken("--pair", pair); err != nil {
		return err
	}
	if limit < 1 || limit > 100 {
		return fmt.Errorf("--limit must be between 1 and 100")
	}
	opts := &gateapi.ListSpotRpiOrderBookOpts{Limit: optional.NewInt32(limit)}
	if interval != "" {
		opts.Interval = optional.NewString(interval)
	}
	if cmd.Flags().Changed("with-id") {
		opts.WithId = optional.NewBool(withId)
	}

	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	result, resp, err := c.SpotAPI.ListSpotRpiOrderBook(c.Context(), pair, opts)
	if err != nil {
		return client.CommandError(err, resp, "GET", "/api/v4/spot/rpi_order_book", "")
	}
	return cmdutil.GetPrinter(cmd).Print(result)
}
