//go:build agent

package agentfeature

import "testing"

func TestDefaultMaxOutputWhenUnsetAgentMode(t *testing.T) {
	t.Run("agent env default when bytes unset", func(t *testing.T) {
		t.Setenv("GATE_CLI_AGENT", "1")
		t.Setenv("GATE_AI_AGENT", "")
		if got := DefaultMaxOutputWhenUnset(); got != DefaultMaxOutputBytes {
			t.Fatalf("expected %d, got %d", DefaultMaxOutputBytes, got)
		}
	})

	t.Run("inactive when env unset", func(t *testing.T) {
		t.Setenv("GATE_CLI_AGENT", "")
		t.Setenv("GATE_AI_AGENT", "")
		if got := DefaultMaxOutputWhenUnset(); got != 0 {
			t.Fatalf("expected 0, got %d", got)
		}
	})
}
