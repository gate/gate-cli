//go:build agentic_e2e

package agentic_test

import (
	"bytes"
	"context"
	"os/exec"
	"testing"
	"time"
)

// sameHomeCLI reuses an isolated profile across real CLI subprocesses.
func sameHomeCLI(t *testing.T, home, base string, args ...string) ([]byte, []byte, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 12*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, cliBinary, args...)
	cmd.Env = childEnv(home, base, false)
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	err := cmd.Run()
	return out.Bytes(), stderr.Bytes(), err
}
