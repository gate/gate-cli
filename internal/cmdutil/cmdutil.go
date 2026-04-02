// Package cmdutil provides shared helpers for CLI command implementations.
// It lives in internal/ to avoid circular imports between cmd/ subpackages and cmd/root.go.
package cmdutil

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/config"
	"github.com/gate/gate-cli/internal/output"
	"github.com/gate/gate-cli/internal/useragent"
)

// GetPrinter returns an output.Printer configured from the --format flag.
func GetPrinter(cmd *cobra.Command) *output.Printer {
	format, _ := cmd.Root().PersistentFlags().GetString("format")
	return output.New(os.Stdout, output.ParseFormat(format))
}

// GetClient builds a Gate API client.
// Priority: --api-key/--api-secret flag > GATE_API_KEY/GATE_API_SECRET env > config file.
func GetClient(cmd *cobra.Command) (*client.Client, error) {
	root := cmd.Root().PersistentFlags()
	profile, _ := root.GetString("profile")
	debug, _ := root.GetBool("debug")
	apiKey, _ := root.GetString("api-key")
	apiSecret, _ := root.GetString("api-secret")

	cfg, err := config.Load(config.Options{
		Profile:       profile,
		FlagAPIKey:    apiKey,
		FlagAPISecret: apiSecret,
	})
	if err != nil {
		return nil, err
	}
	cfg.Debug = debug

	cmdPath := useragent.ExtractCmdPath(cmd.CommandPath())
	return client.New(cfg, cmdPath)
}

// GetSettle returns the futures settlement currency for the given command.
//
// Priority:
//  1. --settle flag if explicitly set by the user
//  2. default_settle from the config file
//  3. "usdt" as the built-in fallback
func GetSettle(cmd *cobra.Command) string {
	if cmd.Flags().Changed("settle") {
		s, _ := cmd.Flags().GetString("settle")
		return s
	}

	root := cmd.Root().PersistentFlags()
	profile, _ := root.GetString("profile")
	cfg, err := config.Load(config.Options{Profile: profile})
	if err == nil && cfg.DefaultSettle != "" {
		return cfg.DefaultSettle
	}

	return "usdt"
}
