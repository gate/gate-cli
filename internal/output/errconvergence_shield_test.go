//go:build !agent

package output

import (
	"strings"
	"testing"
)

func TestCommandNotFoundActionOmitsAgentCommands(t *testing.T) {
	t.Parallel()
	ge := &GateError{
		Status:    404,
		Label:     "NOT_FOUND",
		Message:   `unknown command "x"`,
		ErrorType: "COMMAND_NOT_FOUND",
	}
	FillAgentErrorConvergence(ge)
	if strings.Contains(ge.SuggestedNextAction, "agent-") {
		t.Fatalf("must not suggest agent commands without -tags agent: %q", ge.SuggestedNextAction)
	}
}
