//go:build agent

package cmdhint

import "github.com/gate/gate-cli/internal/agentfeature"

// AgentModeEnabled reports whether GateAI/agent-oriented CLI defaults are active.
func AgentModeEnabled() bool {
	return agentfeature.RuntimeActive()
}
