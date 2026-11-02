//go:build agentic_e2e && (darwin || linux)

package agentic_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// A fake compiler has the same child/pipe ownership as go build. Cleanup always
// kills this test-owned group, including when the pre-fix assertion fails.
func TestFixtureBuildCancellation(t *testing.T) {
	pidfile := filepath.Join(t.TempDir(), "child.pid")
	ctx, cancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", `sleep 30 & echo $! > "$1"; wait`, "fixture", pidfile)
	if err := nativeProcessGroup(cmd); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if cmd.Process != nil {
			_ = signalNativeGroup(cmd, true)
		}
	}()
	cmd.WaitDelay = 200 * time.Millisecond
	started := time.Now()
	_, err := runFixtureBuild(cmd)
	if err == nil || ctx.Err() == nil {
		t.Fatalf("missing cancellation: %v", err)
	}
	if time.Since(started) > 3*time.Second {
		t.Fatal("build cancellation exceeded bound")
	}
	raw, err := os.ReadFile(pidfile)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		t.Fatal(err)
	}
	// A killed child may briefly remain a zombie; ps distinguishes it from a
	// running compiler without relying on a particular host reaper schedule.
	status, err := exec.Command("ps", "-o", "stat=", "-p", strconv.Itoa(pid)).Output()
	if err == nil && strings.TrimSpace(string(status)) != "" && !strings.HasPrefix(strings.TrimSpace(string(status)), "Z") {
		t.Error("compiler descendant survived cancellation")
	}
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			t.Fatal(err)
		}
	}
}

func TestFixtureBuildDiagnosticsBound(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", `awk 'BEGIN {for (i=0;i<70000;i++) printf "x"}'`)
	out, err := runFixtureBuild(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 65536 {
		t.Fatalf("diagnostic bytes=%d, want bounded 65536", len(out))
	}
}
