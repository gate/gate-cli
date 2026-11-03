package contracts

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Export policy protects private evidence without excluding Go's internal product packages.
func TestPublicationPublicExportPathPolicy(t *testing.T) {
	for _, tc := range []struct {
		name    string
		private bool
	}{
		{"docs/cr/review.md", true}, {"docs/qa/nested/.evidence", true},
		{"docs/tech/design.md", true}, {"docs/operations/deploy.md", true},
		{"docs/plans/plan.md", true}, {"docs/superpowers/spec.md", true},
		{"doc/tech/design.md", true}, {"docs/config/nested/.hidden", true},
		{"tests/internal/live_test.go", true}, {"tests/agentic/dev_read_live_test.go", true},
		{"internal/client/client.go", false}, {"internal/operations/interface.json", false},
		{"docs/quickstart.md", false}, {"testdata/integration.yaml.example", false},
		{"tests/agentic/sdk_v179_test.go", false}, {"docs/cr-guide.md", false},
		{"docs/qa-public/guide.md", false}, {"docs/technical/guide.md", false},
		{"docs/operations-guide.md", false}, {"docs/plans-public/guide.md", false},
		{"docs/superpowers-public/guide.md", false}, {"doc/technical/guide.md", false},
		{"docs/configuration/guide.md", false}, {"tests/internalized/unit_test.go", false},
		{"tests/agentic/dev_read_live_test.go.example", false}, {"", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := publicationInternalPath(tc.name); got != tc.private {
				t.Errorf("private=%v want %v", got, tc.private)
			}
		})
	}
}

// Inspect the actual index: ignore rules alone cannot remove already tracked evidence.
func TestPublicationPublicExportTrackedPaths(t *testing.T) {
	paths, err := publicationGit(t, publicationRoot(t), "ls-files", "-z")
	if err != nil {
		t.Fatalf("list public index: %v %s", err, paths)
	}
	for _, name := range strings.Split(string(paths), "\x00") {
		if publicationInternalPath(name) {
			t.Errorf("private path remains tracked: %s", name)
		}
	}
}

// Run Git's matcher with the shipped rules in a fresh repository, so global
// excludes and tracked-state exceptions cannot hide incorrect publication rules.
func TestPublicationPublicExportIgnoreRules(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(publicationRoot(t), ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if out, err := publicationGit(t, dir, "init", "--quiet"); err != nil {
		t.Fatalf("init fixture: %v %s", err, out)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		ignored bool
	}{
		{"docs/cr/review.md", true}, {"docs/qa/nested/.evidence", true},
		{"docs/tech/design.md", true}, {"docs/operations/deploy.md", true},
		{"docs/plans/plan.md", true}, {"docs/superpowers/spec.md", true},
		{"doc/tech/design.md", true}, {"docs/config/nested/.hidden", true},
		{"tests/internal/live_test.go", true}, {"tests/agentic/dev_read_live_test.go", true},
		{"internal/client/client.go", false}, {"internal/operations/interface.json", false},
		{"docs/quickstart.md", false}, {"testdata/integration.yaml.example", false},
		{"docs/cr-guide.md", false}, {"docs/qa-public/guide.md", false},
		{"docs/technical/guide.md", false}, {"docs/operations-guide.md", false},
		{"docs/plans-public/guide.md", false}, {"docs/superpowers-public/guide.md", false},
		{"doc/technical/guide.md", false}, {"docs/configuration/guide.md", false},
		{"tests/internalized/unit_test.go", false},
		{"tests/agentic/dev_read_live_test.go.example", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := publicationGit(t, dir, "-c", "core.excludesFile="+os.DevNull, "check-ignore", "--no-index", "--", tc.name)
			ignored := err == nil
			if err != nil {
				if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 {
					t.Fatalf("check-ignore: %v %s", err, out)
				}
			}
			if ignored != tc.ignored {
				t.Errorf("ignored=%v want %v", ignored, tc.ignored)
			}
		})
	}
}

// Inspect every outgoing commit tree; deleting private files in a later commit
// does not retract earlier blobs. Already published refs and tags are not scanned.
func TestPublicationPublicExportOutgoingHistory(t *testing.T) {
	paths, err := publicationOutgoingPrivatePaths(t, publicationRoot(t), "refs/remotes/origin/main")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 0 {
		t.Fatalf("private paths in outgoing commit trees: %v", paths)
	}
}

// Exercise the complete Git process chain with synthetic commits only; no
// network, remote fetch, production service or developer repository is mocked.
func TestPublicationPublicExportHistoryE2E(t *testing.T) {
	for _, scenario := range []string{"public-multiple-commits", "private-then-deleted", "missing-base-ref"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			git := func(args ...string) []byte {
				t.Helper()
				out, err := publicationGit(t, dir, args...)
				if err != nil {
					t.Fatalf("git %s: %v %s", args[0], err, out)
				}
				return out
			}
			writeCommit := func(name string) {
				t.Helper()
				if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, name), []byte("synthetic publication fixture\n"), 0600); err != nil {
					t.Fatal(err)
				}
				git("add", "--", name)
				git("-c", "core.hooksPath="+os.DevNull, "-c", "commit.gpgsign=false", "-c", "user.name=Publication Test", "-c", "user.email=publication@example.test", "commit", "--quiet", "-m", "Add synthetic fixture")
			}
			git("init", "--quiet")
			writeCommit("README.md")
			base := "refs/remotes/origin/main"
			if scenario != "missing-base-ref" {
				git("update-ref", base, "HEAD")
			}
			switch scenario {
			case "public-multiple-commits":
				writeCommit("internal/client/client.go")
				writeCommit("docs/quickstart.md")
			case "private-then-deleted":
				writeCommit("docs/qa/private.md")
				git("rm", "--", "docs/qa/private.md")
				git("-c", "core.hooksPath="+os.DevNull, "-c", "commit.gpgsign=false", "-c", "user.name=Publication Test", "-c", "user.email=publication@example.test", "commit", "--quiet", "-m", "Delete synthetic evidence")
				// Establish that a final-tree-only guard would incorrectly pass.
				if tree := git("ls-tree", "-r", "--name-only", "HEAD"); strings.Contains(string(tree), "docs/qa/") {
					t.Fatal("fixture still exposes private evidence in its final tree")
				}
			}
			paths, err := publicationOutgoingPrivatePaths(t, dir, base)
			if scenario == "missing-base-ref" {
				if err == nil || !strings.Contains(err.Error(), base) {
					t.Fatalf("missing baseline must fail explicitly: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "private-then-deleted" {
				if len(paths) != 1 || paths[0] != "docs/qa/private.md" {
					t.Fatalf("deleted outgoing evidence was not detected: %v", paths)
				}
			} else if len(paths) != 0 {
				t.Fatalf("public commits rejected: %v", paths)
			}
		})
	}
}

// The remote-tracking baseline must exist locally. Missing history is an error,
// never an implicit fetch or skip; release CI supplies a full-history checkout.
func publicationOutgoingPrivatePaths(t *testing.T, dir, base string) ([]string, error) {
	t.Helper()
	if out, err := publicationGit(t, dir, "rev-parse", "--verify", base+"^{commit}"); err != nil {
		return nil, fmt.Errorf("publication baseline %s unavailable: %w (%s)", base, err, strings.TrimSpace(string(out)))
	}
	commits, err := publicationGit(t, dir, "rev-list", base+"..HEAD")
	if err != nil {
		return nil, fmt.Errorf("list outgoing commits: %w (%s)", err, strings.TrimSpace(string(commits)))
	}
	private := make(map[string]struct{})
	for _, commit := range strings.Fields(string(commits)) {
		paths, err := publicationGit(t, dir, "ls-tree", "-r", "--name-only", "-z", commit)
		if err != nil {
			return nil, fmt.Errorf("inspect outgoing tree %s: %w (%s)", commit, err, strings.TrimSpace(string(paths)))
		}
		for _, name := range strings.Split(string(paths), "\x00") {
			if publicationInternalPath(name) {
				private[name] = struct{}{}
			}
		}
	}
	paths := make([]string, 0, len(private))
	for name := range private {
		paths = append(paths, name)
	}
	sort.Strings(paths)
	return paths, nil
}
