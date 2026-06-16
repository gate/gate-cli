package preflight

import (
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/cmdhint"
	"github.com/gate/gate-cli/internal/cmdutil"
	"github.com/gate/gate-cli/internal/exitcode"
	"github.com/gate/gate-cli/internal/intelcmd"
	"github.com/gate/gate-cli/internal/migration"
	"github.com/gate/gate-cli/internal/output"
	"github.com/gate/gate-cli/internal/version"
)

// Cmd is the preflight command.
var Cmd = &cobra.Command{
	Use:           "preflight",
	Short:         "Run CLI-first preflight check for Gate info/news",
	RunE:          runPreflight,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	Cmd.Flags().Bool("fallback-enabled", true, "Enable MCP fallback route when CLI is not installed")
}

func runPreflight(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	if p.IsTable() {
		p.PrintError(output.UnsupportedTableFormatError())
		return exitcode.New(exitcode.RenderOrInternal, intelcmd.ErrSilenced)
	}
	fallbackEnabled, _ := cmd.Flags().GetBool("fallback-enabled")
	result := migration.BuildPreflight(migration.PreflightOptions{
		FallbackEnabled: fallbackEnabled,
		Version:         version.Version,
	})
	payload := interface{}(result)
	if p.IsJSON() && cmdhint.AgentModeEnabled() {
		payload = map[string]interface{}{
			"route":                 result.Route,
			"action_code":           result.ActionCode,
			"cli_installed":         result.CLIInstalled,
			"legacy_mcp_detected":   result.LegacyMCPDetected,
			"blocking_reason":       result.BlockingReason,
			"user_message":          result.UserMessage,
			"suggested_next_action": cmdhint.AgentPreflightNextAction(result.Route),
			"agent_resolve_hint":    cmdhint.AgentResolveHint("intel preflight"),
		}
	}
	if result.Route == "BLOCK" {
		ge := &output.GateError{
			Status:  422,
			Label:   "PREFLIGHT_BLOCKED",
			Message: result.UserMessage,
		}
		output.FillAgentErrorConvergence(ge)
		if cmdhint.AgentModeEnabled() {
			ge.SuggestedNextAction = cmdhint.AgentPreflightNextAction(result.Route)
		}
		p.PrintError(ge)
		return exitcode.New(exitcode.Failure, intelcmd.ErrSilenced)
	}
	if err := p.Print(payload); err != nil {
		return exitcode.New(exitcode.RenderOrInternal, err)
	}
	return nil
}
