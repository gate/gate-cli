package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

type releaseStep struct {
	Name            string            `yaml:"name"`
	Uses            string            `yaml:"uses"`
	Run             string            `yaml:"run"`
	If              string            `yaml:"if"`
	ContinueOnError bool              `yaml:"continue-on-error"`
	With            map[string]string `yaml:"with"`
	Env             map[string]string `yaml:"env"`
}

// loadReleaseSteps reads the actual workflow so the process suite detects a disconnected notes helper.
func loadReleaseSteps(t *testing.T) (releaseStep, releaseStep) {
	t.Helper()
	var workflow struct {
		Jobs map[string]struct {
			Steps []releaseStep `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(readFixture(t, "../../.github/workflows/release.yaml"), &workflow); err != nil {
		t.Fatal(err)
	}
	var prepare, publish releaseStep
	prepareIndex, publishIndex := -1, -1
	for i, step := range workflow.Jobs["release"].Steps {
		if step.Name == "Prepare public release notes" {
			prepare, prepareIndex = step, i
		}
		if strings.HasPrefix(step.Uses, "goreleaser/goreleaser-action@") {
			publish, publishIndex = step, i
		}
	}
	if prepareIndex < 0 || publishIndex <= prepareIndex || prepare.Run == "" {
		t.Fatal("release must prepare notes before publishing")
	}
	if prepare.ContinueOnError || publish.ContinueOnError || prepare.If != "" || publish.If != "" {
		t.Fatal("release must use default success gating; extraction failures must stop publication")
	}
	if prepare.Env["RELEASE_TAG"] != "${{ github.ref_name }}" || strings.Contains(prepare.Run, "${{") {
		t.Fatal("release tag must be passed through an environment variable, not shell interpolation")
	}
	return prepare, publish
}

// The remote publisher is the only mocked business boundary. The real workflow shell,
// go run command, changelog reader, validation and generated notes file execute unchanged.
func TestReleaseNotesWorkflowE2E(t *testing.T) {
	prepare, publish := loadReleaseSteps(t)
	for _, tc := range []struct {
		name, version, changelog, want string
		failure                        bool
	}{
		{"matching public section", "v1.2.3", "# Changelog\nprivate preamble\n## [v1.2.30]\n- other version\n## [v1.2.3]\n\n### Added\n\n- Public feature.\n## [v1.2.2]\n- older version\n", "### Added\n\n- Public feature.\n", false},
		{"legacy heading", "v1.2.3", "## [1.2.3]\n- Public feature.\n", "- Public feature.\n", false},
		{"previous public changelog", "v0.8.0", string(readFixture(t, "../../CHANGELOG.md")), "", false},
		{"previous minor changelog", "v0.9.0", string(readFixture(t, "../../CHANGELOG.md")), "", false},
		{"current public changelog", "v0.9.1", string(readFixture(t, "../../CHANGELOG.md")), "", false},
		// The three patch tags must each publish only the approved generic note.
		{"patch changelog v0.9.2", "v0.9.2", string(readFixture(t, "../../CHANGELOG.md")), "### Fixed\n\n- Bug fixes.\n", false},
		{"patch changelog v0.9.3", "v0.9.3", string(readFixture(t, "../../CHANGELOG.md")), "### Fixed\n\n- Bug fixes.\n", false},
		{"patch changelog v0.9.4", "v0.9.4", string(readFixture(t, "../../CHANGELOG.md")), "### Fixed\n\n- Bug fixes.\n", false},
		{"missing version", "v1.2.3", "## [v1.2.30]\n- wrong version\n", "", true},
		{"empty version", "v1.2.3", "## [v1.2.3]\n\n## [v1.2.2]\n- older\n", "", true},
		{"duplicate version", "v1.2.3", "## [v1.2.3]\n- first\n## [1.2.3]\n- second\n", "", true},
		{"missing file", "v1.2.3", "", "", true},
		{"unclosed fence", "v1.2.3", "## [v1.2.3]\n```\n## [v1.2.2]\nprivate\n", "", true},
		{"tag as data", "v1.2.3; touch injected", "## [v1.2.3]\n- public\n", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFixture(t, filepath.Join(dir, "go.mod"), string(readFixture(t, "../../go.mod")), 0o600)
			writeFixture(t, filepath.Join(dir, "tools/release-notes/main.go"), string(readFixture(t, "main.go")), 0o600)
			if tc.changelog != "" {
				writeFixture(t, filepath.Join(dir, "CHANGELOG.md"), tc.changelog, 0o600)
			}
			// A real private Git subject must not be consumed even though history is available.
			runProcess(t, dir, nil, "git", "-c", "core.hooksPath=/dev/null", "init", "--quiet")
			runProcess(t, dir, nil, "git", "-c", "user.name=Release Test", "-c", "user.email=release@example.invalid", "-c", "commit.gpgsign=false", "-c", "core.hooksPath=/dev/null", "commit", "--allow-empty", "--quiet", "-m", "PRIVATE_INTERNAL_DECISION_DO_NOT_PUBLISH")
			env := append(os.Environ(), "RELEASE_TAG="+tc.version, "RUNNER_TEMP="+dir, "GOMAXPROCS=2", "GOFLAGS=-p=2", "GOWORK=off")
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "bash", "-e", "-o", "pipefail", "-c", prepare.Run)
			cmd.Dir, cmd.Env = dir, env
			output, err := cmd.CombinedOutput()
			if ctx.Err() != nil {
				t.Fatal(ctx.Err())
			}
			if tc.failure {
				if err == nil {
					t.Fatal("invalid release was accepted")
				}
				notes, readErr := os.ReadFile(filepath.Join(dir, "release-notes.md"))
				if readErr != nil || len(notes) != 0 {
					t.Fatalf("failed extraction exposed notes: %q, %v", notes, readErr)
				}
				if _, statErr := os.Stat(filepath.Join(dir, "injected")); !os.IsNotExist(statErr) {
					t.Fatal("tag text executed as shell code")
				}
				return // GitHub's default success gating does not invoke the next action.
			}
			if err != nil {
				t.Fatalf("workflow extraction failed: %v\n%s", err, output)
			}
			args := strings.Fields(strings.ReplaceAll(publish.With["args"], "${{ runner.temp }}", dir))
			if len(args) != 3 || args[0] != "release" || args[1] != "--clean" || args[2] != "--release-notes="+filepath.Join(dir, "release-notes.md") {
				t.Fatalf("publisher does not consume the prepared file exclusively: %q", args)
			}
			// This local stand-in never contacts GitHub; it captures the action's supplied body.
			publisher := filepath.Join(dir, "publisher")
			writeFixture(t, publisher, "#!/bin/sh\nset -eu\nnotes=${3#--release-notes=}\ntest -s \"$notes\"\ncat \"$notes\" > \"$RUNNER_TEMP/published.md\"\n", 0o700)
			runProcess(t, dir, env, publisher, args...)
			body := string(readFixture(t, filepath.Join(dir, "published.md")))
			if tc.want != "" && body != tc.want {
				t.Fatalf("published body = %q; want %q", body, tc.want)
			}
			// Repository cases follow the public tag sequence; synthetic cases above
			// still prove exact version selection and rejection of neighboring text.
			if tc.name == "previous public changelog" && !strings.Contains(body, "argument isolation") {
				t.Fatalf("published history was lost: %s", body)
			}
			if tc.name == "previous minor changelog" {
				for _, token := range []string{"data.change_log", "--price-type", "cex auth", "cex stock", "UNSUPPORTED_OPERATION"} {
					if !strings.Contains(body, token) {
						t.Errorf("missing public release change %q: %s", token, body)
					}
				}
				if strings.Contains(body, "v0.11.") || strings.Contains(body, "subsequently removed") {
					t.Fatalf("internal release history reached public notes: %s", body)
				}
			}
			// The patch release must expose its own API-field correction, without
			// accidentally publishing the preceding minor release's full notes.
			if tc.name == "current public changelog" {
				for _, token := range []string{"cex futures trail stop-all", "data.orders", "timestamp", "code"} {
					if !strings.Contains(body, token) {
						t.Errorf("missing patch response-field change %q: %s", token, body)
					}
				}
				if strings.Contains(body, "cex auth login") {
					t.Fatal("patch release included the preceding minor release notes")
				}
			}
			for _, forbidden := range []string{"PRIVATE_INTERNAL_DECISION", "private preamble", "other version", "older version"} {
				if strings.Contains(body, forbidden) {
					t.Fatalf("non-release material reached the publisher: %s", forbidden)
				}
			}
		})
	}
}

// GoReleaser must consume explicit notes and replace an old body, including retries of an existing release.
func TestReleaseConfiguration(t *testing.T) {
	loadReleaseSteps(t)
	var cfg struct {
		Release struct {
			Mode   string `yaml:"mode"`
			Header string `yaml:"header"`
			Footer string `yaml:"footer"`
		} `yaml:"release"`
		Changelog struct {
			Disable bool   `yaml:"disable"`
			Use     string `yaml:"use"`
			Filters struct {
				Exclude []string `yaml:"exclude"`
			} `yaml:"filters"`
		} `yaml:"changelog"`
	}
	if err := yaml.Unmarshal(readFixture(t, "../../.goreleaser.yaml"), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Release.Mode != "replace" || cfg.Release.Header != "" || cfg.Release.Footer != "" || cfg.Changelog.Disable || cfg.Changelog.Use != "" || len(cfg.Changelog.Filters.Exclude) != 1 || cfg.Changelog.Filters.Exclude[0] != ".*" {
		t.Fatalf("release configuration can preserve, append, discard or generate notes: %+v", cfg)
	}
}

func readFixture(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func writeFixture(t *testing.T, path, data string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), mode); err != nil {
		t.Fatal(err)
	}
}

func runProcess(t *testing.T, dir string, env []string, name string, args ...string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir, cmd.Env = dir, env
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s failed: %v\n%s", name, err, output)
	}
}

// The public document is itself a release input: implementation paths,
// dependency and test reports must not leak through historical sections either.
func TestPublicChangelogExcludesDevelopmentDetails(t *testing.T) {
	text := strings.ToLower(string(readFixture(t, "../../CHANGELOG.md")))
	// Match engineering metadata without embedding private infrastructure or incident identifiers.
	for _, forbidden := range []string{"internal/", ".go`", "gateapi-go", "sdk", "goreleaser", "go test", "### tests", "### dependencies", "### unchanged", "unit tests", "test coverage", "git@", "ssh://", "source_repo", "source_of_truth", "/users/", "homebrew_tap_token"} {
		if strings.Contains(text, forbidden) {
			t.Errorf("public changelog contains development detail %q", forbidden)
		}
	}
}
