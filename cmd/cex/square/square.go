package square

import (
	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

// Cmd is the root command for the square module.
var Cmd = &cobra.Command{
	Use:   "square",
	Short: "Gate Square commands",
}

func init() {
	aiSearchCmd := &cobra.Command{
		Use:   "ai-search",
		Short: "AI MCP dynamic search (public)",
		RunE:  runAiSearch,
	}
	aiSearchCmd.Flags().String("keyword", "", "Search keyword (e.g., BTC, ETH)")
	aiSearchCmd.Flags().String("currency", "", "Filter by currency code")
	aiSearchCmd.Flags().Int32("time-range", 0, "0=unlimited, 1=last day, 2=last week, 3=last month")
	aiSearchCmd.Flags().Int32("sort", 0, "0=most popular, 1=latest")
	aiSearchCmd.Flags().Int32("limit", 0, "Return count (1-50, default 10)")
	aiSearchCmd.Flags().Int32("page", 0, "Page number")

	liveReplayCmd := &cobra.Command{
		Use:   "live-replay",
		Short: "AI assistant live stream/replay search (public)",
		RunE:  runLiveReplay,
	}
	liveReplayCmd.Flags().String("tag", "", "Business type: Market Analysis, Hot Topics, Blockchain, Others")
	liveReplayCmd.Flags().String("coin", "", "Filter by currency (e.g., BTC, ETH)")
	liveReplayCmd.Flags().String("sort", "", "Sort: hot (default), new")
	liveReplayCmd.Flags().Int32("limit", 0, "Return count (1-10, default 3)")

	Cmd.AddCommand(aiSearchCmd, liveReplayCmd)
}

func runAiSearch(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.ListSquareAiSearchOpts{}
	if v, _ := cmd.Flags().GetString("keyword"); v != "" {
		opts.Keyword = optional.NewString(v)
	}
	if v, _ := cmd.Flags().GetString("currency"); v != "" {
		opts.Currency = optional.NewString(v)
	}
	if v, _ := cmd.Flags().GetInt32("time-range"); v > 0 {
		opts.TimeRange = optional.NewInt32(v)
	}
	if v, _ := cmd.Flags().GetInt32("sort"); v > 0 {
		opts.Sort = optional.NewInt32(v)
	}
	if v, _ := cmd.Flags().GetInt32("limit"); v > 0 {
		opts.Limit = optional.NewInt32(v)
	}
	if v, _ := cmd.Flags().GetInt32("page"); v > 0 {
		opts.Page = optional.NewInt32(v)
	}

	result, httpResp, err := c.SquareAPI.ListSquareAiSearch(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/social/message/search", ""))
		return nil
	}
	return p.Print(result)
}

func runLiveReplay(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.ListLiveReplayOpts{}
	if v, _ := cmd.Flags().GetString("tag"); v != "" {
		opts.Tag = optional.NewString(v)
	}
	if v, _ := cmd.Flags().GetString("coin"); v != "" {
		opts.Coin = optional.NewString(v)
	}
	if v, _ := cmd.Flags().GetString("sort"); v != "" {
		opts.Sort = optional.NewString(v)
	}
	if v, _ := cmd.Flags().GetInt32("limit"); v > 0 {
		opts.Limit = optional.NewInt32(v)
	}

	result, httpResp, err := c.SquareAPI.ListLiveReplay(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/social/live/tag_coin_live_replay", ""))
		return nil
	}
	return p.Print(result)
}
