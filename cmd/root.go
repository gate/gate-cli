package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/cmd/account"
	"github.com/gate/gate-cli/cmd/activity"
	"github.com/gate/gate-cli/cmd/alpha"
	configcmd "github.com/gate/gate-cli/cmd/config"
	"github.com/gate/gate-cli/cmd/coupon"
	crossex "github.com/gate/gate-cli/cmd/cross_ex"
	"github.com/gate/gate-cli/cmd/delivery"
	"github.com/gate/gate-cli/cmd/earn"
	flashswap "github.com/gate/gate-cli/cmd/flash_swap"
	"github.com/gate/gate-cli/cmd/futures"
	"github.com/gate/gate-cli/cmd/launch"
	"github.com/gate/gate-cli/cmd/margin"
	"github.com/gate/gate-cli/cmd/mcl"
	"github.com/gate/gate-cli/cmd/options"
	"github.com/gate/gate-cli/cmd/p2p"
	"github.com/gate/gate-cli/cmd/rebate"
	"github.com/gate/gate-cli/cmd/spot"
	"github.com/gate/gate-cli/cmd/square"
	subaccount "github.com/gate/gate-cli/cmd/sub_account"
	"github.com/gate/gate-cli/cmd/tradfi"
	"github.com/gate/gate-cli/cmd/unified"
	"github.com/gate/gate-cli/cmd/wallet"
	"github.com/gate/gate-cli/cmd/welfare"
	"github.com/gate/gate-cli/cmd/withdrawal"
)

var rootCmd = &cobra.Command{
	Use:   "gate-cli",
	Short: "Gate API command-line interface",
	Long:  "gate-cli wraps the Gate API for easy use from the terminal and in scripts.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().String("format", "table", "Output format: table or json")
	rootCmd.PersistentFlags().String("profile", "default", "Config profile to use")
	rootCmd.PersistentFlags().Bool("debug", false, "Print raw HTTP request/response")
	rootCmd.PersistentFlags().String("api-key", "", "Gate API key (overrides config file and GATE_API_KEY env)")
	rootCmd.PersistentFlags().String("api-secret", "", "Gate API secret (overrides config file and GATE_API_SECRET env)")

	rootCmd.AddCommand(configcmd.Cmd)
	rootCmd.AddCommand(spot.Cmd)
	rootCmd.AddCommand(futures.Cmd)
	rootCmd.AddCommand(tradfi.Cmd)
	rootCmd.AddCommand(alpha.Cmd)
	rootCmd.AddCommand(account.Cmd)
	rootCmd.AddCommand(wallet.Cmd)
	rootCmd.AddCommand(options.Cmd)
	rootCmd.AddCommand(delivery.Cmd)
	rootCmd.AddCommand(margin.Cmd)
	rootCmd.AddCommand(unified.Cmd)
	rootCmd.AddCommand(subaccount.Cmd)
	rootCmd.AddCommand(earn.Cmd)
	rootCmd.AddCommand(flashswap.Cmd)
	rootCmd.AddCommand(mcl.Cmd)
	rootCmd.AddCommand(crossex.Cmd)
	rootCmd.AddCommand(p2p.Cmd)
	rootCmd.AddCommand(rebate.Cmd)
	rootCmd.AddCommand(withdrawal.Cmd)
	rootCmd.AddCommand(activity.Cmd)
	rootCmd.AddCommand(coupon.Cmd)
	rootCmd.AddCommand(launch.Cmd)
	rootCmd.AddCommand(square.Cmd)
	rootCmd.AddCommand(welfare.Cmd)
}
