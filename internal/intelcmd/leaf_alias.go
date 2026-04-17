package intelcmd

import (
	"strings"

	"github.com/spf13/cobra"
)

// LeafAliasConfig builds one info/news-style shortcut leaf command (CR-811).
type LeafAliasConfig struct {
	// BackendCLI is the top-level Cobra command name for examples, e.g. "info" or "news".
	BackendCLI string
	Use        string
	ToolName   string
	RunE       func(cmd *cobra.Command, args []string) error
}

// NewLeafAliasCommand returns a leaf alias command with shared help text and fallback flags.
func NewLeafAliasCommand(cfg LeafAliasConfig) *cobra.Command {
	parts := strings.Split(cfg.ToolName, "_")
	group := cfg.BackendCLI
	if len(parts) >= 2 {
		group = parts[1]
	}
	cmd := &cobra.Command{
		Use:   cfg.Use,
		Short: "Shortcut for " + cfg.ToolName,
		Long: "Calls " + cfg.ToolName + ". Flat flags come from a static baseline plus any extra fields from the intel backend; use --params/--args-json/--args-file as JSON fallback.",
		Example: "  gate-cli " + cfg.BackendCLI + " " + group + " " + cfg.Use + " --format json\n" +
			"  gate-cli " + cfg.BackendCLI + " " + group + " " + cfg.Use + " --params '{\"key\":\"value\"}'",
		Args: cobra.NoArgs,
		RunE: cfg.RunE,
	}
	AddFallbackArgFlags(cmd)
	return cmd
}
