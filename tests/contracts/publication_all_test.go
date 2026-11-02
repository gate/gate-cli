package contracts

import (
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Walk the entire tracked public tree, including tests and embedded resources.
// Internal evidence and separately maintained integration fixtures are excluded
// explicitly, rather than silently limiting the guard to previously fixed files.
func TestPublicationEntireRetainedTree(t *testing.T) {
	root := publicationRoot(t)
	paths, err := publicationGit(t, root, "ls-files", "-z")
	if err != nil {
		t.Fatal(err)
	}
	urls := regexp.MustCompile("(?i)(?:https?|wss?|ftp)://[^\\s\"'`<>\\\\（）。，；：]+")
	metadata := regexp.MustCompile(`"source_(?:repo|of_truth)"\s*:`)
	storage := regexp.MustCompile(`\b(?:dim|dwd|dws)_[a-zA-Z0-9_]+\b`)
	for _, name := range strings.Split(string(paths), "\x00") {
		if name == "" || publicationInternalPath(name) {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		if storage.Match(raw) || (strings.HasSuffix(name, ".json") && metadata.Match(raw)) {
			t.Errorf("implementation metadata in %s", name)
		}
		for _, text := range urls.FindAllString(string(raw), -1) {
			u, err := url.Parse(strings.TrimRight(text, ",;)）：。"))
			if err != nil || u.Hostname() == "" {
				continue
			}
			host := strings.ToLower(u.Hostname())
			if publicationExampleHost(host) || strings.HasSuffix(host, ".example") || strings.HasSuffix(host, ".test") || host == "example.test" {
				continue
			}
			switch host {
			case "api-testnet.gateapi.io", "api.gatemcp.ai", "api.github.com", "github.com", "raw.githubusercontent.com", "gate.com", "keepachangelog.com", "playwright.dev", "demo.playwright.dev":
				continue
			}
			if ip, e := netip.ParseAddr(host); e == nil && ip.IsUnspecified() {
				continue
			}
			t.Errorf("unreviewed network authority in %s", name)
		}
	}
	fixture, err := os.ReadFile(filepath.Join(root, "internal/oauth2/discovery_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(fixture), `default_profile: sample\ndefault_settle:`) {
		t.Error("discovery fixture must use the documented sample profile")
	}
}

func publicationInternalPath(name string) bool {
	for _, prefix := range []string{"docs/qa/", "docs/cr/", "docs/tech/", "docs/operations/", "docs/plans/", "docs/superpowers/", "doc/tech/", "docs/config/", "tests/internal/"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return name == "tests/agentic/dev_read_live_test.go"
}

// The public entry point must run its complete self-contained suite. Private
// integration tests are preserved separately and must not be disguised as skips.
func TestPublicationPublicTestEntry(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(publicationRoot(t), "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "CLI_GATEWAY_TESTS") || strings.Contains(string(raw), "-skip") {
		t.Error("public test entry still hides integration dependencies behind skipped tests")
	}
}
