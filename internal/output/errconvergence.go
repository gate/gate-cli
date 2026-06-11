package output

import "github.com/gate/gate-cli/internal/agentfeature"

// FillAgentErrorType sets ErrorType when omitted so PrintError JSON is agent-convergable.
func FillAgentErrorType(ge *GateError) {
	if ge == nil || ge.ErrorType != "" {
		return
	}
	ge.ErrorType = ClassifyCLIError(ge.Status, ge.Label, ge.Message)
}

// FillAgentErrorConvergence sets error_type, retryable, and suggested_next_action for agent stderr JSON.
func FillAgentErrorConvergence(ge *GateError) {
	if ge == nil {
		return
	}
	FillAgentErrorType(ge)
	if ge.ErrorType == "" {
		ge.ErrorType = "UNKNOWN"
	}
	if ge.SuggestedNextAction == "" {
		ge.SuggestedNextAction = defaultSuggestedNextAction(ge.ErrorType, ge.Label)
	}
	ge.Retryable = defaultRetryable(ge.ErrorType, ge.Label)
}

func defaultRetryable(errorType, label string) bool {
	switch label {
	case "PREFLIGHT_BLOCKED", "DOCTOR_FAILED", "MIGRATE_FAILED":
		return false
	}
	switch errorType {
	case "INVALID_ARGS":
		return true
	default:
		return false
	}
}

func defaultSuggestedNextAction(errorType, label string) string {
	switch label {
	case "PREFLIGHT_BLOCKED":
		return "fix CLI install/version or Intel MCP config; do not retry the same preflight until resolved"
	case "DOCTOR_FAILED":
		return "fix failing doctor checks (config, connectivity, legacy MCP); do not retry doctor until resolved"
	case "MIGRATE_FAILED":
		return "fix migrate failures (backup, provider config); do not retry migrate until resolved"
	}
	switch errorType {
	case "INVALID_ARGS":
		return "fix flags or query; you may check the leaf command --help once, then retry with corrected args"
	case "AUTH_ERROR":
		return "stop retrying; configure GATE_API_KEY/GATE_API_SECRET or Intel GATE_INTEL_* bearer tokens"
	case "PERMISSION_DENIED":
		return "stop retrying; verify API key permissions or Intel bearer scope"
	case "EMPTY_RESULT":
		return "answer that no records matched; do not blindly retry the same command"
	case "NETWORK_ERROR":
		return "stop business retries; report service unavailable or retry later without changing args"
	case "COMMAND_NOT_FOUND":
		return agentfeature.DiscoveryResolveOrLeavesAction()
	default:
		return "inspect stderr message and gate_cli_diagnostic if present; avoid repeated identical retries"
	}
}
