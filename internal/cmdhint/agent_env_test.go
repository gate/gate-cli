//go:build agent

package cmdhint

import "testing"

func TestAgentModeEnabled(t *testing.T) {
	t.Setenv("GATE_CLI_AGENT", "1")
	t.Setenv("GATE_AI_AGENT", "")
	if !AgentModeEnabled() {
		t.Fatal("expected enabled with GATE_CLI_AGENT=1")
	}
	t.Setenv("GATE_CLI_AGENT", "")
	t.Setenv("GATE_AI_AGENT", "true")
	if !AgentModeEnabled() {
		t.Fatal("expected enabled with GATE_AI_AGENT=true")
	}
	t.Setenv("GATE_AI_AGENT", "")
	if AgentModeEnabled() {
		t.Fatal("expected disabled when env unset")
	}
}
