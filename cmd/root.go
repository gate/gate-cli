package cmd

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/cmd/account"
	"github.com/gate/gate-cli/cmd/activity"
	"github.com/gate/gate-cli/cmd/alpha"
	configcmd "github.com/gate/gate-cli/cmd/config"
	"github.com/gate/gate-cli/cmd/coupon"
	crossex "github.com/gate/gate-cli/cmd/cross_ex"
	"github.com/gate/gate-cli/cmd/delivery"
	"github.com/gate/gate-cli/cmd/doctor"
	"github.com/gate/gate-cli/cmd/earn"
	flashswap "github.com/gate/gate-cli/cmd/flash_swap"
	"github.com/gate/gate-cli/cmd/futures"
	"github.com/gate/gate-cli/cmd/info"
	"github.com/gate/gate-cli/cmd/launch"
	"github.com/gate/gate-cli/cmd/margin"
	"github.com/gate/gate-cli/cmd/mcl"
	"github.com/gate/gate-cli/cmd/migrate"
	"github.com/gate/gate-cli/cmd/news"
	"github.com/gate/gate-cli/cmd/options"
	"github.com/gate/gate-cli/cmd/p2p"
	"github.com/gate/gate-cli/cmd/preflight"
	"github.com/gate/gate-cli/cmd/rebate"
	"github.com/gate/gate-cli/cmd/spot"
	"github.com/gate/gate-cli/cmd/square"
	subaccount "github.com/gate/gate-cli/cmd/sub_account"
	"github.com/gate/gate-cli/cmd/tradfi"
	"github.com/gate/gate-cli/cmd/unified"
	"github.com/gate/gate-cli/cmd/wallet"
	"github.com/gate/gate-cli/cmd/welfare"
	"github.com/gate/gate-cli/cmd/withdrawal"
	"github.com/gate/gate-cli/internal/exitcode"
	"github.com/gate/gate-cli/internal/intelcmd"
	"github.com/gate/gate-cli/internal/version"
)

var rootCmd = &cobra.Command{
	Use:     "gate-cli",
	Short:   "Gate API command-line interface",
	Long:    "gate-cli wraps the Gate API for easy use from the terminal and in scripts.",
	Version: version.Version,
}

const formatDeprecationEnv = "GATE_CLI_SUPPRESS_FORMAT_NOTICE"

func Execute() {
	intelcmd.SilenceCommandTree(rootCmd)
	emitFormatCompatNotice(rootCmd)
	if err := rootCmd.Execute(); err != nil {
		var codedErr *exitcode.Error
		if errors.As(err, &codedErr) {
			os.Exit(codedErr.Code)
		}
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("gate-cli version %s\n", version.Version)
		},
	})

	rootCmd.PersistentFlags().String("format", "pretty", "Output format: json, pretty (default), or table (only for tabular list-style commands)")
	rootCmd.PersistentFlags().String("profile", "default", "Config profile to use")
	rootCmd.PersistentFlags().Bool("debug", false, "Print HTTP debug summary (no auth headers/body)")
	rootCmd.PersistentFlags().Bool("verbose", false, "Print Intel MCP transport lines to stderr (info/news); does not change stdout JSON shape")
	rootCmd.PersistentFlags().Int64("max-output-bytes", defaultMaxOutputBytes(), "Maximum bytes for info/news tool command output (0 means unlimited; env: GATE_MAX_OUTPUT_BYTES)")
	rootCmd.PersistentFlags().String("api-key", "", "Gate API key (overrides config file and GATE_API_KEY env)")
	rootCmd.PersistentFlags().String("api-secret", "", "Gate API secret (overrides config file and GATE_API_SECRET env)")

	rootCmd.AddCommand(configcmd.Cmd)
	rootCmd.AddCommand(spot.Cmd)
	rootCmd.AddCommand(futures.Cmd)
	rootCmd.AddCommand(tradfi.Cmd)
	rootCmd.AddCommand(alpha.Cmd)
	rootCmd.AddCommand(account.Cmd)
	rootCmd.AddCommand(wallet.Cmd)
	rootCmd.AddCommand(news.Cmd)
	rootCmd.AddCommand(info.Cmd)
	rootCmd.AddCommand(preflight.Cmd)
	rootCmd.AddCommand(doctor.Cmd)
	rootCmd.AddCommand(migrate.Cmd)
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

func defaultMaxOutputBytes() int64 {
	raw := strings.TrimSpace(os.Getenv("GATE_MAX_OUTPUT_BYTES"))
	if raw == "" {
		return 0
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v < 0 {
		return 0
	}
	return v
}

func emitFormatCompatNotice(root *cobra.Command) {
	if root == nil {
		return
	}
	if strings.TrimSpace(os.Getenv(formatDeprecationEnv)) != "" {
		return
	}
	f := root.PersistentFlags().Lookup("format")
	if f != nil && f.Changed {
		return
	}
	_, _ = fmt.Fprintln(os.Stderr, "Notice: default --format is pretty. Set --format explicitly in scripts for stable output.")
}
