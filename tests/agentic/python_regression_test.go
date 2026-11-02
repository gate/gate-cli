//go:build agentic_e2e

package agentic_test

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Historical process suites do not match test_*.py discovery. Reuse TestMain's
// actual standard/Agent binary so the unified E2E entry cannot silently omit
// file containment, CEX optional parameters or Bot lifecycle assertions.
func TestHistoricalPythonCLI(t *testing.T) {
	for _, script := range []string{"intel_args_file_cli.py", "cex_cli.py", "bot_cli.py", "test_agentic_scope_cli.py"} {
		t.Run(script, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
			defer cancel()
			cmd := exec.CommandContext(ctx, "python3", "-B", filepath.Join("..", "e2e", script), "-v")
			cmd.Env = append(childEnv(t.TempDir(), "", false), "GATE_CLI_BINARY="+cliBinary, "PYTHONDONTWRITEBYTECODE=1")
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("historical CLI E2E failed: %v\n%s", err, out)
			} else {
				t.Logf("%s", out)
			}
		})
	}
}
