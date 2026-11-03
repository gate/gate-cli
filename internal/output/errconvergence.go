package output

import (
	"net/http"

	"github.com/gate/gate-cli/internal/agentfeature"
	"github.com/gate/gate-cli/internal/operations"
)

// FillAgentErrorType sets ErrorType when omitted so PrintError JSON is agent-convergable.
func FillAgentErrorType(ge *GateError) {
	if ge == nil || ge.ErrorType != "" {
		return
	}
	// Reuse the canonical operation catalog instead of fuzzy URL/message
	// matching. Unknown metadata keeps UNKNOWN, never a fabricated account
	// initialization instruction. Explicit types and auth statuses still win.
	if ge.Status == http.StatusBadRequest && ge.Label == "USER_NOT_FOUND" && ge.Request != nil && ge.Request.Method == http.MethodGet {
		op, err := operations.Lookup(ge.Request.Method, ge.Request.URL)
		if err == nil && op != nil {
			switch op.OperationID {
			case "listFuturesAccounts", "listFuturesOrders", "listDeliveryAccounts", "listDeliveryOrders":
				ge.ErrorType = "ACCOUNT_NOT_READY"
				return
			}
		}
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
	// A validation label does not prove a failed write was never committed.
	// Apply this last-mile guard before default advice can suggest replaying it.
	if ge.Request != nil && ge.Request.Method != "" && ge.Request.Method != http.MethodGet && ge.Request.Method != http.MethodHead && ((ge.Status >= 200 && ge.Status < 300) || ge.Status >= 500) {
		ge.Retryable = false
		if ge.SuggestedNextAction == "" {
			ge.SuggestedNextAction = "query the original order or business reference before submitting any further write"
		}
		return
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
	case "ACCOUNT_NOT_READY":
		return "stop retrying; verify that the product account is enabled or initialized through the official account workflow; do not change flags or transfer funds automatically"
	case "UNSUPPORTED_OPERATION":
		// API-order TP/SL updates remain unsupported after correcting arguments.
		return "stop retrying; API orders do not support editing, adding or deleting TP/SL conditions; changing flags cannot enable this operation"
	case "INVALID_ARGS":
		return "fix flags or query; you may check the leaf command --help once, then retry with corrected args"
	case "AUTH_ERROR":
		return "stop retrying; configure GATE_API_KEY/GATE_API_SECRET or Intel GATE_INTEL_* bearer tokens"
	case "PERMISSION_DENIED":
		return "stop retrying; verify API key permissions or Intel bearer scope"
	case "EMPTY_RESULT":
		return "answer that no records matched; do not blindly retry the same command"
	case "API_NOT_FOUND":
		// Do not turn a proxy's generic 404 into a fabricated account result.
		return "stop business retries; verify the API origin, gateway route and installed command contract; do not report an empty account result"
	case "NETWORK_ERROR":
		return "stop business retries; report service unavailable or retry later without changing args"
	case "COMMAND_NOT_FOUND":
		return agentfeature.DiscoveryResolveOrLeavesAction()
	default:
		return "inspect stderr message and gate_cli_diagnostic if present; avoid repeated identical retries"
	}
}
