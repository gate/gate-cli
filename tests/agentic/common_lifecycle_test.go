//go:build agentic_e2e

package agentic_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gate/gate-cli/internal/agentfeature"
)

// Common commands share one isolated HOME so later commands must observe the
// actual persisted result. Only local fixture configuration is ever mutated.
func TestCommonCommandLifecycle(t *testing.T) {
	home := t.TempDir()
	binary := cliBinary
	invoke := func(input string, args ...string) ([]byte, []byte, error) {
		t.Helper()
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()
		child := exec.CommandContext(ctx, binary, args...)
		env := childEnv(home, "", false)
		// Preflight/doctor discover this exact tested binary, not a host install.
		for i, item := range env {
			if strings.HasPrefix(item, "PATH=") {
				env[i] = "PATH=" + filepath.Dir(binary) + string(os.PathListSeparator) + strings.TrimPrefix(item, "PATH=")
			}
		}
		child.Env = env
		child.Stdin = strings.NewReader(input)
		var out, stderr bytes.Buffer
		child.Stdout, child.Stderr = &out, &stderr
		err := child.Run()
		return out.Bytes(), stderr.Bytes(), err
	}
	ok := func(input string, args ...string) []byte {
		t.Helper()
		out, stderr, err := invoke(input, args...)
		if err != nil {
			t.Fatalf("%v: %v stdout=%s stderr=%s", args, err, out, stderr)
		}
		return out
	}
	version := ok("", "version", "--details", "--format", "json")
	if !json.Valid(version) || !bytes.Contains(version, []byte(`"schema_version":1`)) {
		t.Fatalf("invalid build details %s", version)
	}
	ok("scenario\nsynthetic-scenario-key\nsynthetic-scenario-secret\n", "config", "init")
	path := filepath.Join(home, ".gate-cli", "config.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte("synthetic-scenario-key")) {
		t.Fatal("init did not persist selected profile")
	}
	ok("", "config", "set", "api-key", "synthetic-rotated-key", "--profile", "scenario")
	ok("", "config", "set", "base-url", "http://127.0.0.1:1", "--profile", "scenario")
	listed := ok("", "config", "list")
	for _, secret := range []string{"synthetic-rotated-key", "synthetic-scenario-key", "synthetic-scenario-secret"} {
		if bytes.Contains(listed, []byte(secret)) {
			t.Fatalf("list exposed %s", secret)
		}
	}
	check := ok("", "config", "check", "--format", "json")
	if !bytes.Contains(check, []byte(`"status":"pass"`)) {
		t.Fatalf("private config permission check=%s", check)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"config", "set", "unknown-key", "x"}, {"config", "set", "api-key"}, {"config", "init"}} {
		_, _, err := invoke("", args...)
		if err == nil {
			t.Fatalf("invalid/incomplete configuration operation accepted: %v", args)
		}
		after, readErr := os.ReadFile(path)
		if readErr != nil || !bytes.Equal(before, after) {
			t.Fatalf("failed operation corrupted configuration: %v", readErr)
		}
	}
	// Doctor is intentionally limited to local checks: no external Intel service
	// participates in the common/CEX scope of this task.
	doctor := ok("", "doctor", "--check", "config", "--format", "json")
	if !json.Valid(doctor) || !bytes.Contains(doctor, []byte(`"status":"pass"`)) {
		t.Fatalf("local doctor=%s", doctor)
	}
	// The latest feature ships a release default. Test that path first, then
	// explicitly build dev to retain the historical negative version gate.
	releaseReport := ok("", "doctor", "--check", "version", "--format", "json")
	if !bytes.Contains(releaseReport, []byte(`"status":"pass"`)) {
		t.Fatalf("release version rejected: %s", releaseReport)
	}
	_, source, _, _ := runtime.Caller(0)
	binary = filepath.Join(t.TempDir(), "gate-cli")
	buildCtx, buildCancel := context.WithTimeout(t.Context(), 3*time.Minute)
	build := exec.CommandContext(buildCtx, cliBuildGo, "build", "-p", "1", "-tags", "agentic_e2e", "-ldflags", "-X github.com/gate/gate-cli/internal/version.Version=dev", "-o", binary, ".")
	build.Env = append(os.Environ(), "GOTMPDIR="+t.TempDir())
	build.Dir = filepath.Clean(filepath.Join(filepath.Dir(source), "../.."))
	buildOut, buildErr := runFixtureBuild(build)
	buildCancel()
	if buildErr != nil {
		t.Fatalf("build dev version fixture: %v %s", buildErr, buildOut)
	}
	versionReport, versionErr, versionFailure := invoke("", "doctor", "--check", "version", "--format", "json")
	if versionFailure == nil || !json.Valid(versionReport) || !bytes.Contains(versionErr, []byte("DOCTOR_FAILED")) {
		t.Fatalf("development version was accepted: %v %s %s", versionFailure, versionReport, versionErr)
	}
	preflight, preflightErr, preflightFailure := invoke("", "preflight", "--format", "json")
	if preflightFailure == nil || len(preflight) != 0 || !bytes.Contains(preflightErr, []byte("PREFLIGHT_BLOCKED")) {
		t.Fatalf("development preflight gate lost: %v %s %s", preflightFailure, preflight, preflightErr)
	}
	binary = cliBinary
	// Migration is a common command. Preview, confirmation failure and apply
	// must preserve unrelated entries. The current migration contract deliberately
	// leaves TOML/JSON edits as a manual patch even with --apply --yes.
	provider := filepath.Join(home, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(provider), 0700); err != nil {
		t.Fatal(err)
	}
	original := []byte("model = \"keep-model\"\n[mcp_servers.gate-info]\ncommand = \"fixture-only\"\n[mcp_servers.other]\ncommand = \"keep-command\"\n")
	if err := os.WriteFile(provider, original, 0600); err != nil {
		t.Fatal(err)
	}
	preview, previewErr, previewFailure := invoke("", "migrate", "--dry-run", "--provider", "codex", "--format", "json")
	var previewExit *exec.ExitError
	if !errors.As(previewFailure, &previewExit) || previewExit.ExitCode() != 10 || !bytes.Contains(preview, []byte("manual_patch")) {
		t.Fatalf("migration preview warning lost: %v %s %s", previewFailure, preview, previewErr)
	}
	after, err := os.ReadFile(provider)
	if err != nil || !bytes.Equal(original, after) {
		t.Fatal("migration preview changed file")
	}
	_, stderr, err := invoke("", "migrate", "--apply", "--provider", "codex", "--format", "json")
	if err == nil || !bytes.Contains(stderr, []byte("CONFIRMATION_REQUIRED")) {
		t.Fatalf("migration confirmation lost: %v %s", err, stderr)
	}
	backup := filepath.Join(home, "backups")
	report, stderr, err := invoke("", "migrate", "--apply", "--yes", "--provider", "codex", "--backup-dir", backup, "--format", "json")
	var warning *exec.ExitError
	if !errors.As(err, &warning) || warning.ExitCode() != 10 || !bytes.Contains(report, []byte("manual_patch")) {
		t.Fatalf("manual migration contract lost: %v %s %s", err, report, stderr)
	}
	after, err = os.ReadFile(provider)
	if err != nil || !bytes.Equal(after, original) {
		t.Fatalf("migration changed unrelated state: %v %s", err, after)
	}
	backedUp := false
	if err := filepath.WalkDir(backup, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if bytes.Equal(raw, original) {
				backedUp = true
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !backedUp {
		t.Fatal("manual migration lost the exact recovery backup")
	}
	stable := append([]byte{}, after...)
	report, stderr, err = invoke("", "migrate", "--apply", "--yes", "--provider", "codex", "--backup-dir", backup, "--format", "json")
	if !errors.As(err, &warning) || warning.ExitCode() != 10 || !bytes.Contains(report, []byte("manual_patch")) {
		t.Fatalf("repeat migration warning lost: %v %s %s", err, report, stderr)
	}
	after, err = os.ReadFile(provider)
	if err != nil || !bytes.Equal(stable, after) {
		t.Fatal("repeated migration was not idempotent")
	}
}

func TestCommonAgentCEXDiscovery(t *testing.T) {
	for _, args := range [][]string{
		{"agent-index", "--domain", "cex", "--details", "--command", "cex futures order long"},
		{"agent-search", "--domain", "cex", "--query", "futures long", "--limit", "2"},
		{"agent-resolve", "--domain", "cex", "--query", "cex futures order close", "--limit", "1"},
	} {
		t.Run(args[0], func(t *testing.T) {
			out, stderr, err := runCLI(t, "http://127.0.0.1:1", false, append([]string{"--format", "json", "--max-output-bytes", "0"}, args...)...)
			if !agentfeature.Commands {
				if err == nil || len(out) != 0 {
					t.Fatalf("agent-only command unexpectedly published: %v %s %s", err, out, stderr)
				}
				return
			}
			if err != nil || !json.Valid(out) {
				t.Fatalf("CEX discovery failed: %v %s %s", err, out, stderr)
			}
			if args[0] == "agent-search" {
				// Search promises bounded keyword matches within the selected
				// domain, not a particular natural-language ranking result.
				var result struct {
					Domain   string
					Commands []string
				}
				if json.Unmarshal(out, &result) != nil || result.Domain != "cex" || len(result.Commands) == 0 || len(result.Commands) > 2 {
					t.Fatalf("CEX search bounds/domain lost: %s", out)
				}
				for _, command := range result.Commands {
					if !strings.HasPrefix(command, "gate-cli cex ") {
						t.Fatalf("CEX search escaped domain: %s", command)
					}
				}
			} else if !bytes.Contains(out, []byte("cex futures order")) {
				t.Fatalf("canonical CEX discovery target lost: %s", out)
			}
			if args[0] == "agent-index" && (!bytes.Contains(out, []byte("contract")) || !bytes.Contains(out, []byte("size"))) {
				t.Fatalf("CEX parameters missing: %s", out)
			}
		})
	}
}
