package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// Current CEX guides must follow the checked-in SDK and must not advertise
// executable retired Launch examples. Historical QA reports are not rewritten.
func TestCEXCompatCurrentDocs(t *testing.T) {
	_, source, _, _ := runtime.Caller(0)
	root := filepath.Dir(filepath.Dir(source))
	mod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	match := regexp.MustCompile(`github.com/gate/gateapi-go/v7\s+(v[0-9.]+)`).FindSubmatch(mod)
	if len(match) != 2 {
		t.Fatal("SDK requirement not found")
	}
	retired := regexp.MustCompile(`(?m)^\s*gate-cli\s+(?:--format\s+\S+\s+)?cex\s+launch\b`)
	for _, path := range []string{"README.md", "docs/quickstart.md", "docs/cex-gap-commands.md"} {
		raw, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), string(match[1])) {
			t.Errorf("%s omits current SDK %s", path, match[1])
		}
		if retired.Match(raw) {
			t.Errorf("%s advertises retired executable command", path)
		}
	}
}
