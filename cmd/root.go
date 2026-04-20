package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	cexcmd "github.com/gate/gate-cli/cmd/cex"
	configcmd "github.com/gate/gate-cli/cmd/config"
	"github.com/gate/gate-cli/cmd/doctor"
	"github.com/gate/gate-cli/cmd/info"
	"github.com/gate/gate-cli/cmd/migrate"
	"github.com/gate/gate-cli/cmd/news"
	"github.com/gate/gate-cli/cmd/preflight"
	"github.com/gate/gate-cli/internal/exitcode"
	"github.com/gate/gate-cli/internal/intelcmd"
	"github.com/gate/gate-cli/internal/version"
)

var rootCmd = &cobra.Command{
	Use:     "gate-cli",
	Short:   "Gate API command-line interface",
	Long:    "gate-cli wraps the Gate API for easy use from the terminal and in scripts. CEX commands live under `cex` (e.g. gate-cli cex spot market candlesticks --pair BTC_USDT).",
	Version: version.Version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		emitFormatCompatNotice(cmd)
		normalizeMaxOutputBytesFlag(cmd)
	},
}

const (
	formatDeprecationEnv = "GATE_CLI_SUPPRESS_FORMAT_NOTICE"
	formatNoticeForceEnv = "GATE_CLI_FORMAT_NOTICE_FORCE" // non-empty: emit compat notice even when stderr is not a TTY (tests only)
)

func setupRootForExecute() {
	intelcmd.SilenceCommandTree(rootCmd)
}

func Execute() {
	setupRootForExecute()
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
	rootCmd.AddCommand(cexcmd.Cmd)
	rootCmd.AddCommand(news.Cmd)
	rootCmd.AddCommand(info.Cmd)
	rootCmd.AddCommand(preflight.Cmd)
	rootCmd.AddCommand(doctor.Cmd)
	rootCmd.AddCommand(migrate.Cmd)
}

func defaultMaxOutputBytes() int64 {
	raw := strings.TrimSpace(os.Getenv("GATE_MAX_OUTPUT_BYTES"))
	if raw == "" {
		return 0
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v < 0 {
		_, _ = fmt.Fprintf(os.Stderr, "Warning: invalid GATE_MAX_OUTPUT_BYTES=%q; fallback to unlimited output\n", raw)
		return 0
	}
	return v
}

// normalizeMaxOutputBytesFlag enforces a non-negative --max-output-bytes (CR-107).
// Negative or unreadable values warn on stderr and clamp to 0 (unlimited), matching env validation behavior.
func normalizeMaxOutputBytesFlag(cmd *cobra.Command) {
	if cmd == nil {
		return
	}
	cmd = cmd.Root()
	if cmd.PersistentFlags().Lookup("max-output-bytes") == nil {
		return
	}
	v, err := cmd.PersistentFlags().GetInt64("max-output-bytes")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Warning: invalid --max-output-bytes: %v; using unlimited (0)\n", err)
		_ = cmd.PersistentFlags().Set("max-output-bytes", "0")
		return
	}
	if v < 0 {
		_, _ = fmt.Fprintf(os.Stderr, "Warning: negative --max-output-bytes (%d) is invalid; using unlimited (0)\n", v)
		_ = cmd.PersistentFlags().Set("max-output-bytes", "0")
	}
}

func suppressFormatCompatNoticeFor(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	for x := cmd; x != nil; x = x.Parent() {
		switch strings.ToLower(x.Name()) {
		case "version", "help", "completion":
			return true
		}
	}
	r := cmd.Root()
	if r != nil {
		if f := r.PersistentFlags().Lookup("help"); f != nil && f.Changed {
			return true
		}
	}
	if f := cmd.Flags().Lookup("help"); f != nil && f.Changed {
		return true
	}
	return false
}

func emitFormatCompatNotice(cmd *cobra.Command) {
	if cmd == nil {
		return
	}
	if suppressFormatCompatNoticeFor(cmd) {
		return
	}
	root := cmd.Root()
	if strings.TrimSpace(os.Getenv(formatDeprecationEnv)) != "" {
		return
	}
	f := root.PersistentFlags().Lookup("format")
	if f != nil && f.Changed {
		return
	}
	force := strings.TrimSpace(os.Getenv(formatNoticeForceEnv)) != ""
	if !force && !isatty.IsTerminal(os.Stderr.Fd()) {
		return
	}
	_, _ = io.WriteString(os.Stderr, "Notice: default --format is pretty. Set --format explicitly in scripts for stable output.\n")
}
