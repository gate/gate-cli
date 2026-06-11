//go:build !agent

package agentfeature

import "testing"

func TestRuntimeActiveFalseWhenBuildTagOff(t *testing.T) {
	t.Setenv("GATE_CLI_AGENT", "1")
	t.Setenv("GATE_AI_AGENT", "")
	if RuntimeActive() {
		t.Fatal("RuntimeActive must be false without -tags agent even when GATE_CLI_AGENT=1")
	}
	if DefaultMaxOutputWhenUnset() != 0 {
		t.Fatalf("expected 0 default max output, got %d", DefaultMaxOutputWhenUnset())
	}
	if !RuntimeEnvEnabled() {
		t.Fatal("RuntimeEnvEnabled should still read GATE_CLI_AGENT for UA etc.")
	}
}
