// Package cex registers all Gate CEX (centralized exchange) CLI command groups
// under the `gate-cli cex ...` prefix.
package cex

import (
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/cmd/cex/account"
	"github.com/gate/gate-cli/cmd/cex/activity"
	"github.com/gate/gate-cli/cmd/cex/alpha"
	"github.com/gate/gate-cli/cmd/cex/assetswap"
	"github.com/gate/gate-cli/cmd/cex/bot"
	"github.com/gate/gate-cli/cmd/cex/coupon"
	crossex "github.com/gate/gate-cli/cmd/cex/cross_ex"
	"github.com/gate/gate-cli/cmd/cex/delivery"
	"github.com/gate/gate-cli/cmd/cex/earn"
	flashswap "github.com/gate/gate-cli/cmd/cex/flash_swap"
	"github.com/gate/gate-cli/cmd/cex/futures"
	"github.com/gate/gate-cli/cmd/cex/launch"
	"github.com/gate/gate-cli/cmd/cex/margin"
	"github.com/gate/gate-cli/cmd/cex/mcl"
	"github.com/gate/gate-cli/cmd/cex/options"
	"github.com/gate/gate-cli/cmd/cex/p2p"
	"github.com/gate/gate-cli/cmd/cex/rebate"
	"github.com/gate/gate-cli/cmd/cex/spot"
	"github.com/gate/gate-cli/cmd/cex/square"
	subaccount "github.com/gate/gate-cli/cmd/cex/sub_account"
	"github.com/gate/gate-cli/cmd/cex/tradfi"
	"github.com/gate/gate-cli/cmd/cex/unified"
	"github.com/gate/gate-cli/cmd/cex/wallet"
	"github.com/gate/gate-cli/cmd/cex/welfare"
	"github.com/gate/gate-cli/cmd/cex/withdrawal"
)

// Cmd is the parent command for CEX API command groups (spot, futures, wallet, …).
// CLI config remains at the root: gate-cli config …
var Cmd = &cobra.Command{
	Use:   "cex",
	Short: "Gate CEX (centralized exchange) commands",
	Long:  "Spot, futures, wallet, earn, and other Gate Exchange API operations. Example: gate-cli cex spot market candlesticks --pair BTC_USDT",
}

func init() {
	Cmd.AddCommand(spot.Cmd)
	Cmd.AddCommand(futures.Cmd)
	Cmd.AddCommand(tradfi.Cmd)
	Cmd.AddCommand(alpha.Cmd)
	Cmd.AddCommand(account.Cmd)
	Cmd.AddCommand(wallet.Cmd)
	Cmd.AddCommand(options.Cmd)
	Cmd.AddCommand(delivery.Cmd)
	Cmd.AddCommand(margin.Cmd)
	Cmd.AddCommand(unified.Cmd)
	Cmd.AddCommand(subaccount.Cmd)
	Cmd.AddCommand(earn.Cmd)
	Cmd.AddCommand(flashswap.Cmd)
	Cmd.AddCommand(mcl.Cmd)
	Cmd.AddCommand(crossex.Cmd)
	Cmd.AddCommand(p2p.Cmd)
	Cmd.AddCommand(rebate.Cmd)
	Cmd.AddCommand(withdrawal.Cmd)
	Cmd.AddCommand(activity.Cmd)
	Cmd.AddCommand(coupon.Cmd)
	Cmd.AddCommand(launch.Cmd)
	Cmd.AddCommand(square.Cmd)
	Cmd.AddCommand(welfare.Cmd)
	Cmd.AddCommand(assetswap.Cmd)
	Cmd.AddCommand(bot.Cmd)
}
