package earn

import (
	"encoding/json"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

var stakingCmd = &cobra.Command{
	Use:   "staking",
	Short: "On-chain staking commands",
}

func init() {
	findCmd := &cobra.Command{
		Use:   "find",
		Short: "List staking coins",
		RunE:  runStakingFind,
	}
	findCmd.Flags().String("cointype", "", "Currency type: swap (voucher), lock (locked), debt (US Treasury)")

	ordersCmd := &cobra.Command{
		Use:   "orders",
		Short: "List staking orders",
		RunE:  runStakingOrders,
	}
	ordersCmd.Flags().Int32("pid", 0, "Product ID")
	ordersCmd.Flags().String("coin", "", "Currency name")
	ordersCmd.Flags().Int32("type", 0, "Type: 0=staking, 1=redemption")
	ordersCmd.Flags().Int32("page", 0, "Page number")

	awardsCmd := &cobra.Command{
		Use:   "awards",
		Short: "List staking dividend records",
		RunE:  runStakingAwards,
	}
	awardsCmd.Flags().Int32("pid", 0, "Product ID")
	awardsCmd.Flags().String("coin", "", "Currency name")
	awardsCmd.Flags().Int32("page", 0, "Page number")

	assetsCmd := &cobra.Command{
		Use:   "assets",
		Short: "List staking assets",
		RunE:  runStakingAssets,
	}
	assetsCmd.Flags().String("coin", "", "Currency name")

	swapCmd := &cobra.Command{
		Use:   "swap",
		Short: "Swap staking coin (stake or redeem)",
		RunE:  runStakingSwap,
	}
	swapCmd.Flags().String("coin", "", "Currency name (required)")
	swapCmd.Flags().Int32("side", 0, "0=stake, 1=redeem (required)")
	swapCmd.Flags().String("amount", "", "Amount (required)")
	swapCmd.Flags().Int32("pid", 0, "DeFi protocol ID")
	swapCmd.MarkFlagRequired("coin")
	swapCmd.MarkFlagRequired("side")
	swapCmd.MarkFlagRequired("amount")

	stakingCmd.AddCommand(findCmd, ordersCmd, awardsCmd, assetsCmd, swapCmd)
}

func runStakingFind(cmd *cobra.Command, args []string) error {
	cointype, _ := cmd.Flags().GetString("cointype")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var opts *gateapi.FindCoinOpts
	if cointype != "" {
		opts = &gateapi.FindCoinOpts{
			Cointype: optional.NewString(cointype),
		}
	}

	result, httpResp, err := c.EarnAPI.FindCoin(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/earn/staking/coins", ""))
		return nil
	}
	return p.Print(result)
}

func runStakingOrders(cmd *cobra.Command, args []string) error {
	pid, _ := cmd.Flags().GetInt32("pid")
	coin, _ := cmd.Flags().GetString("coin")
	typ, _ := cmd.Flags().GetInt32("type")
	page, _ := cmd.Flags().GetInt32("page")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.OrderListOpts{}
	if pid != 0 {
		opts.Pid = optional.NewInt32(pid)
	}
	if coin != "" {
		opts.Coin = optional.NewString(coin)
	}
	if cmd.Flags().Changed("type") {
		opts.Type_ = optional.NewInt32(typ)
	}
	if page != 0 {
		opts.Page = optional.NewInt32(page)
	}

	result, httpResp, err := c.EarnAPI.OrderList(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/earn/staking/orders", ""))
		return nil
	}
	return p.Print(result)
}

func runStakingAwards(cmd *cobra.Command, args []string) error {
	pid, _ := cmd.Flags().GetInt32("pid")
	coin, _ := cmd.Flags().GetString("coin")
	page, _ := cmd.Flags().GetInt32("page")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.AwardListOpts{}
	if pid != 0 {
		opts.Pid = optional.NewInt32(pid)
	}
	if coin != "" {
		opts.Coin = optional.NewString(coin)
	}
	if page != 0 {
		opts.Page = optional.NewInt32(page)
	}

	result, httpResp, err := c.EarnAPI.AwardList(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/earn/staking/awards", ""))
		return nil
	}
	return p.Print(result)
}

func runStakingAssets(cmd *cobra.Command, args []string) error {
	coin, _ := cmd.Flags().GetString("coin")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var opts *gateapi.AssetListOpts
	if coin != "" {
		opts = &gateapi.AssetListOpts{
			Coin: optional.NewString(coin),
		}
	}

	result, httpResp, err := c.EarnAPI.AssetList(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/earn/staking/assets", ""))
		return nil
	}
	return p.Print(result)
}

func runStakingSwap(cmd *cobra.Command, args []string) error {
	coin, _ := cmd.Flags().GetString("coin")
	side, _ := cmd.Flags().GetInt32("side")
	amount, _ := cmd.Flags().GetString("amount")
	pid, _ := cmd.Flags().GetInt32("pid")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body := gateapi.SwapCoin{
		Coin:   coin,
		Side:   side,
		Amount: amount,
		Pid:    pid,
	}
	bodyJSON, _ := json.Marshal(body)
	result, httpResp, err := c.EarnAPI.SwapStakingCoin(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/earn/staking/swap", string(bodyJSON)))
		return nil
	}
	return p.Print(result)
}
