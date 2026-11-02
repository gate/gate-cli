package contracts

import (
	"context"
	"encoding/json"
	"net/netip"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Check actual Git matching in an isolated repository: secret files are ignored,
// while public templates, implementation, fixtures and release inputs stay visible.
func TestPublicationIgnoreRules(t *testing.T) {
	root := publicationRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	// Git hooks and developer shells can export repository selectors. Keep a
	// foreign sentinel outside the fixture and prove the child never creates it.
	foreign := t.TempDir()
	foreignGit := filepath.Join(foreign, "foreign.git")
	foreignIndex := filepath.Join(foreign, "index")
	t.Setenv("GIT_DIR", foreignGit)
	t.Setenv("GIT_WORK_TREE", filepath.Join(foreign, "worktree"))
	t.Setenv("GIT_INDEX_FILE", foreignIndex)
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "core.bare")
	t.Setenv("GIT_CONFIG_VALUE_0", "true")
	defer func() {
		for _, name := range []string{foreignGit, foreignIndex} {
			if _, err := os.Stat(name); !os.IsNotExist(err) {
				t.Errorf("Git touched a foreign repository selector: %s (%v)", filepath.Base(name), err)
			}
		}
	}()
	if out, err := publicationGit(t, dir, "-c", "core.hooksPath="+os.DevNull, "init", "--quiet"); err != nil {
		t.Fatalf("init: %v %s", err, out)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		path    string
		ignored bool
	}{
		{"build/gate-cli", true}, {"build/nested/.token", true}, {"dist/archive.zip", true},
		{".cr-temp-review.diff", true}, {".tmp-check", true}, {".cache/.private", true},
		{".env", true}, {".env.local", true}, {"deploy/.env.production", true},
		{".gate-cli/config.yaml", true}, {"scratch/.gate-cli/tokens.yaml", true},
		{"keys/local.pem", true}, {"keys/local.key", true}, {"backup/account.sqlite3", true},
		{"backup/account.dump", true}, {"capture/session.har", true}, {"capture/session.pcapng", true},
		{".git-credentials", true}, {".netrc", true},
		{"testdata/integration.yaml", true}, {".claude/settings.local.json", true},
		{".cursor/hooks/state/learning.json", true}, {"nested/.DS_Store", true},
		{"artifact.test", true}, {"coverage.html", true}, {"docs/config/nested/.hidden", true},
		{".git", true}, {"tests/internal/__pycache__/runner.cpython-313.pyc", true}, {"tests/e2e/__pycache__/suite.pyc", true}, {"tests/e2e/scope_contract.py", false}, {"testdata/signing.key.example", false}, {".env.example", false}, {"deploy/.env.production.example", false},
		{"testdata/integration.yaml.example", false}, {"cmd/cex/spot/order.go", false},
		{"internal/operations/interface.json", false}, {"tests/agentic/testdata/recovery_interruption.go.fixture", false},
		{"tools/release-notes/main.go", false}, {"docs/quickstart.md", false},
		{".github/workflows/release.yaml", false}, {".goreleaser.yaml", false},
	} {
		t.Run(tc.path, func(t *testing.T) {
			out, err := publicationGit(t, dir, "-c", "core.excludesFile="+os.DevNull, "check-ignore", "--no-index", "--", tc.path)
			ignored := err == nil
			if err != nil {
				if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 {
					t.Fatalf("check-ignore: %v %s", err, out)
				}
			}
			if ignored != tc.ignored {
				t.Fatalf("ignored=%v want %v", ignored, tc.ignored)
			}
		})
	}
}

// Public test inputs use reserved examples rather than a real environment.
// Malformed URLs remain rejection fixtures; no production URL policy is changed.
func TestPublicationFixtureHosts(t *testing.T) {
	root := publicationRoot(t)
	pattern := regexp.MustCompile("(?i)(?:https?|ftp)://[^\\s\"'`<>\\\\]+")
	for _, name := range []string{"internal/client/http_base_test.go", "internal/config/yamledit_test.go", "internal/oauth2/agentic_contract_test.go", "internal/oauth2/discovery_test.go", "internal/oauth2/normalize_test.go", "internal/oauth2/prod_token_test.go"} {
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(root, name))
			if err != nil {
				t.Fatal(err)
			}
			for _, text := range pattern.FindAllString(string(raw), -1) {
				parsed, err := url.Parse(strings.TrimRight(text, ",;)"))
				if err != nil {
					continue
				}
				host := strings.ToLower(parsed.Hostname())
				if host == "" {
					continue
				}
				if !publicationExampleHost(host) {
					t.Errorf("fixture contains a non-example authority at %s", name)
				}
			}
		})
	}
	raw, err := os.ReadFile(filepath.Join(root, "tests/agentic/fixtures/agentic_scope_contract.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if _, ok := fixture["source_url"]; ok {
		t.Error("scope fixture still embeds a source URL")
	}
	if _, ok := fixture["source_title"]; ok {
		t.Error("scope fixture still embeds internal source metadata")
	}
}

func publicationExampleHost(host string) bool {
	if host == "localhost" || host == "api.gateio.ws" || host == "www.gate.com" || host == "www.gate.io" || host == "x" {
		return true
	}
	// RFC-reserved examples remain legitimate discovery rejection fixtures.
	if host == "example.com" || strings.HasSuffix(host, ".example.com") {
		return true
	}
	if strings.HasSuffix(host, ".example.test") || strings.HasSuffix(host, ".example.invalid") || strings.HasSuffix(host, ".invalid") {
		return true
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}
	if ip.IsLoopback() {
		return true
	}
	return host == "10.0.0.1" || host == "10.0.0.2" || netip.MustParsePrefix("192.0.2.0/24").Contains(ip) || netip.MustParsePrefix("2001:db8::/32").Contains(ip)
}

// Guides must remain usable without private review pages or missing local scripts.
func TestPublicationGuideReferences(t *testing.T) {
	root := publicationRoot(t)
	pattern := regexp.MustCompile(`docs/(?:qa|cr|tech|plans|operations)/|specs/|scripts/|\.cursor/`)
	for _, name := range []string{"AGENTS.md", "CLAUDE.md", ".agent/skills/gate-cli-add-module/SKILL.md", "docs/integration-test-plan.md", "testdata/integration.yaml.example", "tests/agentic/README.auth-status-earn.md", "tests/agentic/README.market-regressions.md", "tests/agentic/README.autoinvest-asset-type.md", "tests/agentic/README.cex-common.md", "tests/agentic/README.fixed-earn-filters.md"} {
		raw, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if pattern.Match(raw) {
			t.Errorf("%s references non-public local material", name)
		}
	}
}

func publicationRoot(t *testing.T) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test root")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(source), "../.."))
}

// Bound Git execution and discard inherited repository/config overrides so the
// matching test cannot read or mutate a developer's active Git workspace.
func publicationGit(t *testing.T, dir string, args ...string) ([]byte, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.WaitDelay = time.Second
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "GIT_") {
			cmd.Env = append(cmd.Env, entry)
		}
	}
	cmd.Env = append(cmd.Env, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_SYSTEM="+os.DevNull, "GIT_TERMINAL_PROMPT=0")
	return cmd.CombinedOutput()
}
