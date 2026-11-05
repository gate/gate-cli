// Package cex registers all Gate CEX (centralized exchange) CLI command groups
// under the `gate-cli cex ...` prefix.
package cex

import (
	"github.com/gate/gate-cli/cmd/auth"
	"github.com/gate/gate-cli/cmd/cex/stock"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/cmd/cex/account"
	"github.com/gate/gate-cli/cmd/cex/alpha"
	"github.com/gate/gate-cli/cmd/cex/assetswap"
	"github.com/gate/gate-cli/cmd/cex/bot"
	"github.com/gate/gate-cli/cmd/cex/delivery"
	"github.com/gate/gate-cli/cmd/cex/earn"
	flashswap "github.com/gate/gate-cli/cmd/cex/flash_swap"
	"github.com/gate/gate-cli/cmd/cex/futures"
	"github.com/gate/gate-cli/cmd/cex/margin"
	"github.com/gate/gate-cli/cmd/cex/mcl"
	"github.com/gate/gate-cli/cmd/cex/options"
	"github.com/gate/gate-cli/cmd/cex/rebate"
	"github.com/gate/gate-cli/cmd/cex/spot"
	"github.com/gate/gate-cli/cmd/cex/square"
	"github.com/gate/gate-cli/cmd/cex/tradfi"
	"github.com/gate/gate-cli/cmd/cex/unified"
	"github.com/gate/gate-cli/cmd/cex/wallet"
)

// Cmd is the parent command for CEX API command groups (spot, futures, wallet, …).
// CLI config remains at the root: gate-cli config …
var Cmd = &cobra.Command{
	Use: "cex",
	// 分组自身展示help，未知子命令必须非零，避免下线入口被误判成功。
	Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	Short: "Gate CEX (centralized exchange) commands",
	Long:  "Spot, futures, wallet, earn, and other Gate Exchange API operations. Example: gate-cli cex spot market candlesticks --pair BTC_USDT",
}

func init() {
	Cmd.AddCommand(auth.Cmd)
	Cmd.AddCommand(spot.Cmd)
	Cmd.AddCommand(futures.Cmd)
	Cmd.AddCommand(tradfi.Cmd)
	Cmd.AddCommand(stock.Cmd)
	// Launch products are retired; keep their API operations blocked in the shared transport.
	Cmd.AddCommand(alpha.Cmd)
	Cmd.AddCommand(account.Cmd)
	Cmd.AddCommand(wallet.Cmd)
	Cmd.AddCommand(options.Cmd)
	Cmd.AddCommand(delivery.Cmd)
	Cmd.AddCommand(margin.Cmd)
	Cmd.AddCommand(unified.Cmd)
	// Sub-account administration is retired; wallet balance/transfer commands remain.
	Cmd.AddCommand(earn.Cmd)
	Cmd.AddCommand(flashswap.Cmd)
	Cmd.AddCommand(mcl.Cmd)
	// CrossEx, P2P, activity, coupon and welfare implementations are retained,
	// but are not published in this release. The API transport also denies them.
	Cmd.AddCommand(rebate.Cmd)
	Cmd.AddCommand(square.Cmd)
	Cmd.AddCommand(assetswap.Cmd)
	Cmd.AddCommand(bot.Cmd)
}
