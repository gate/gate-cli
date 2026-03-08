// Package cmdutil provides shared helpers for CLI command implementations.
// It lives in internal/ to avoid circular imports between cmd/ subpackages and cmd/root.go.
package cmdutil

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/config"
	"github.com/gate/gate-cli/internal/output"
)

// GetPrinter returns an output.Printer configured from the --format flag.
func GetPrinter(cmd *cobra.Command) *output.Printer {
	format, _ := cmd.Root().PersistentFlags().GetString("format")
	return output.New(os.Stdout, output.ParseFormat(format))
}

// GetClient builds a Gate API client from the --profile and --debug flags plus config file/env.
func GetClient(cmd *cobra.Command) (*client.Client, error) {
	profile, _ := cmd.Root().PersistentFlags().GetString("profile")
	debug, _ := cmd.Root().PersistentFlags().GetBool("debug")

	cfg, err := config.Load(config.Options{Profile: profile})
	if err != nil {
		return nil, err
	}
	cfg.Debug = debug
	return client.New(cfg)
}
