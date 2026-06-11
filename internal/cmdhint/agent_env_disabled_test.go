//go:build !agent

package cmdhint

import "testing"

func TestAgentModeEnabledFalseWithoutBuildTag(t *testing.T) {
	t.Setenv("GATE_CLI_AGENT", "1")
	t.Setenv("GATE_AI_AGENT", "")
	if AgentModeEnabled() {
		t.Fatal("AgentModeEnabled must be false without -tags agent")
	}
}
