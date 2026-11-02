//go:build agentic_e2e && (darwin || linux)

package agentic_test

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

// An actual OS PTY verifies no-echo, final persistence and restored terminal
// state. Python's stdlib supplies openpty; there is no mocked terminal or SDK.
func TestCEXCompatHiddenTTY(t *testing.T) {
	for _, scenario := range []string{"paste", "secret-eof", "empty", "ctrl-c", "early-eof", "oversize", "term"} {
		t.Run(scenario, func(t *testing.T) {
			home := t.TempDir()
			ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, "python3", "testdata/hidden_input_pty.py", cliBinary, home, scenario)
			command.Env = childEnv(home, "http://127.0.0.1:1", false)
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("PTY %v %s", err, output)
			}
		})
	}
}
