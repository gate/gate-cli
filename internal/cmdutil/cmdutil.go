// Package cmdutil provides shared helpers for CLI command implementations.
// It lives in internal/ to avoid circular imports between cmd/ subpackages and cmd/root.go.
package cmdutil

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/config"
	"github.com/gate/gate-cli/internal/output"
	"github.com/gate/gate-cli/internal/toolconfig"
	"github.com/gate/gate-cli/internal/useragent"
)

// GetPrinter returns an output.Printer configured from the --format flag.
func GetPrinter(cmd *cobra.Command) *output.Printer {
	format, _ := cmd.Root().PersistentFlags().GetString("format")
	return output.New(os.Stdout, output.ParseFormat(format))
}

// IntelMCPTransportDiag reports whether info/news MCP clients should emit RPC transport
// summaries to stderr, and which prefix to use (--debug wins over --verbose; PRD §3.7.13).
func IntelMCPTransportDiag(cmd *cobra.Command) (enabled bool, tag string) {
	root := cmd.Root().PersistentFlags()
	d, _ := root.GetBool("debug")
	v, _ := root.GetBool("verbose")
	if d {
		return true, "[debug]"
	}
	if v {
		return true, "[verbose]"
	}
	return false, ""
}

// IntelMCPBaseURLs returns effective Info/News MCP base URLs (env overrides config file "intel").
func IntelMCPBaseURLs(cmd *cobra.Command) (infoURL, newsURL string, err error) {
	root := cmd.Root().PersistentFlags()
	profile, _ := root.GetString("profile")
	apiKey, _ := root.GetString("api-key")
	apiSecret, _ := root.GetString("api-secret")
	cfg, err := config.Load(config.Options{
		Profile:       profile,
		FlagAPIKey:    apiKey,
		FlagAPISecret: apiSecret,
	})
	if err != nil {
		return "", "", err
	}
	infoURL, newsURL = config.EffectiveIntelMCPURLs(cfg.Intel)
	return infoURL, newsURL, nil
}

// ResolveIntelEndpoint loads ~/.gate-cli/config.yaml (respecting profile and API key flags)
// and resolves the MCP endpoint for the given intel backend ("info" or "news").
// Non-empty GATE_INTEL_* environment variables override file defaults.
func ResolveIntelEndpoint(cmd *cobra.Command, backend string) (*toolconfig.ResolvedEndpoint, error) {
	root := cmd.Root().PersistentFlags()
	profile, _ := root.GetString("profile")
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
	return toolconfig.Resolve(toolconfig.ResolveOptions{Backend: backend, IntelFile: cfg.Intel})
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
