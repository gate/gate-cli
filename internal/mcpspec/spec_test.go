package mcpspec

import (
	"os"
	"path/filepath"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if st, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && !st.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found from cwd")
		}
		dir = parent
	}
}

func TestBundledMatchesSpecs(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	pairs := []struct {
		spec, bundled string
	}{
		{
			filepath.Join(root, "specs", "mcp", "info-mcp-tools-inputs-logic.json"),
			filepath.Join(root, "internal", "mcpspec", "bundled", "info-mcp-tools-inputs-logic.json"),
		},
		{
			filepath.Join(root, "specs", "mcp", "news-tools-args-and-logic.json"),
			filepath.Join(root, "internal", "mcpspec", "bundled", "news-tools-args-and-logic.json"),
		},
	}
	for _, p := range pairs {
		// specs/ is gitignored, so this parity check only runs where the
		// author keeps the source spec. CI has the bundled copy only and
		// must skip rather than Fatal.
		a, err := os.ReadFile(p.spec)
		if err != nil {
			if os.IsNotExist(err) {
				t.Skipf("skip parity: spec %s not present (expected in CI)", p.spec)
			}
			t.Fatalf("read spec %s: %v", p.spec, err)
		}
		b, err := os.ReadFile(p.bundled)
		if err != nil {
			t.Fatalf("read bundled %s: %v", p.bundled, err)
		}
		if string(a) != string(b) {
			t.Fatalf("bundled out of sync with spec; cp specs/mcp/*.json internal/mcpspec/bundled/ — compare %s vs %s", p.spec, p.bundled)
		}
	}
}

func TestInfoInputsLogicParses(t *testing.T) {
	t.Parallel()
	v, err := InfoInputsLogic()
	if err != nil {
		t.Fatal(err)
	}
	m, ok := v.(map[string]interface{})
	if !ok || m["tools"] == nil {
		t.Fatalf("unexpected shape %#v", v)
	}
}

func TestNewsToolsArgsParses(t *testing.T) {
	t.Parallel()
	v, err := NewsToolsArgs()
	if err != nil {
		t.Fatal(err)
	}
	m, ok := v.(map[string]interface{})
	if !ok || m["tools"] == nil {
		t.Fatalf("unexpected shape %#v", v)
	}
}
