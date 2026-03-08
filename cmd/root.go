package cmd

import (
	"os"

	"github.com/spf13/cobra"

	configcmd "github.com/gate/gate-cli/cmd/config"
	"github.com/gate/gate-cli/cmd/futures"
	"github.com/gate/gate-cli/cmd/spot"
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
}
