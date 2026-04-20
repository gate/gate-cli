package intelcmd

import (
	"strings"

	"github.com/spf13/cobra"
)

// AnnotationIntelToolName is the cobra.Command.Annotations key holding the MCP tool name
// (e.g. info_markettrend_get_indicator_history) for intel leaf aliases. Used by tests and tooling.
const AnnotationIntelToolName = "gate-cli.intel.tool-name"

// LeafAliasConfig builds one info/news-style shortcut leaf command (CR-811).
type LeafAliasConfig struct {
	// BackendCLI is the top-level Cobra command name for examples, e.g. "info" or "news".
	BackendCLI string
	Use        string
	ToolName   string
	RunE       func(cmd *cobra.Command, args []string) error
	// LongAppend is optional MCP-spec narrative (logic, field notes) appended to the standard leaf Long.
	LongAppend string
}

// NewLeafAliasCommand returns a leaf alias command with shared help text and fallback flags.
func NewLeafAliasCommand(cfg LeafAliasConfig) *cobra.Command {
	parts := strings.Split(cfg.ToolName, "_")
	group := cfg.BackendCLI
	if len(parts) >= 2 {
		group = parts[1]
	}
	long := "MCP tool " + cfg.ToolName + ". Prefer flat flags below; --params, --args-json, and --args-file are JSON fallbacks for uncommon fields.\n" +
		"Per-field notes in this help: set GATE_INTEL_LEAF_HELP=full (default omits the Parameters block to avoid duplicating flag lines)."
	if strings.TrimSpace(cfg.LongAppend) != "" {
		long = long + "\n\n---\n\n" + strings.TrimSpace(cfg.LongAppend)
	}
	cmd := &cobra.Command{
		Use:   cfg.Use,
		Short: "Shortcut for " + cfg.ToolName,
		Long:  long,
		Example: "  gate-cli " + cfg.BackendCLI + " " + group + " " + cfg.Use + " --format json\n" +
			"  gate-cli " + cfg.BackendCLI + " " + group + " " + cfg.Use + " --params '{\"key\":\"value\"}'",
		Args: cobra.NoArgs,
		RunE: cfg.RunE,
	}
	if cfg.ToolName != "" {
		if cmd.Annotations == nil {
			cmd.Annotations = map[string]string{}
		}
		cmd.Annotations[AnnotationIntelToolName] = cfg.ToolName
	}
	AddFallbackArgFlags(cmd)
	return cmd
}
