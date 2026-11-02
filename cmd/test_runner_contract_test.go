package cmd

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Exercise the real Make expansion without recursively running tests. Both
// builds and the historical package entry must remain in the default workflow.
func TestCommonCEXUnifiedRunner(t *testing.T) {
	child := exec.CommandContext(t.Context(), "make", "-n", "-j8", "test-cli", "GO=go")
	child.Dir = ".."
	raw, err := child.CombinedOutput()
	require.NoError(t, err, "%s", raw)
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	require.Len(t, lines, 5, "default workflow must retain unit/local/auth/agent-unit/agent-E2E stages")
	require.Contains(t, lines[0], "./cmd/cex/...")
	require.Contains(t, lines[1], "-tags agentic_e2e")
	require.Contains(t, lines[2], "-run '^TestAuth'")
	require.Contains(t, lines[3], "-tags agent ")
	require.Contains(t, lines[4], "-tags agentic_e2e,agent")
	for _, line := range lines {
		require.Contains(t, line, "-p 1 -parallel 2 -count=1")
		require.NotContains(t, line, "integration", "live trading must remain an explicit separate entry")
	}
}
