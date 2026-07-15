package mcpspec

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
		if p.bundled == filepath.Join(root, "internal", "mcpspec", "bundled", "info-mcp-tools-inputs-logic.json") {
			var specDoc, bundledDoc interface{}
			if err := json.Unmarshal(a, &specDoc); err != nil {
				t.Fatalf("parse spec %s: %v", p.spec, err)
			}
			if err := json.Unmarshal(b, &bundledDoc); err != nil {
				t.Fatalf("parse bundled %s: %v", p.bundled, err)
			}
			ok, err := infoSpecEqualExcludingDescriptions(specDoc, bundledDoc)
			if err != nil {
				t.Fatal(err)
			}
			if !ok {
				t.Fatalf("bundled out of sync with local spec on logic/fields (descriptions are bundled-only); run scripts/patch-info-spec-descriptions.py — %s vs %s", p.spec, p.bundled)
			}
			continue
		}
		if string(a) != string(b) {
			t.Fatalf("bundled out of sync with spec; cp specs/mcp/*.json internal/mcpspec/bundled/ — compare %s vs %s", p.spec, p.bundled)
		}
	}
}

func TestBundledSocialInsightCoinSemantics(t *testing.T) {
	t.Parallel()
	doc, err := NewsToolsArgs()
	if err != nil {
		t.Fatal(err)
	}
	root := doc.(map[string]interface{})
	wanted := map[string]bool{
		"news_feed_get_mention_burst": false,
		"news_feed_get_hot_topics":    false,
	}
	for _, rawTool := range root["tools"].([]interface{}) {
		tool := rawTool.(map[string]interface{})
		name, _ := tool["name"].(string)
		if _, ok := wanted[name]; !ok {
			continue
		}
		wanted[name] = true
		inputRules := tool["input_rules"].(map[string]interface{})
		for _, rawParam := range inputRules["params"].([]interface{}) {
			param := rawParam.(map[string]interface{})
			if param["name"] != "coin" {
				continue
			}
			notes, _ := param["notes"].(string)
			if strings.Contains(strings.ToLower(notes), "validated by") || strings.Contains(strings.ToLower(notes), "recognized by") {
				t.Errorf("%s coin notes overstate validation: %q", name, notes)
			}
			if !strings.Contains(notes, "hide_reason=no_data") {
				t.Errorf("%s coin notes miss unknown/no-data behavior: %q", name, notes)
			}
		}
	}
	for name, found := range wanted {
		if !found {
			t.Errorf("bundled spec missing %s", name)
		}
	}
}

func TestStripInfoToolDescriptions(t *testing.T) {
	t.Parallel()
	in := map[string]interface{}{
		"meta": map[string]interface{}{
			"tool_entry_keys": map[string]interface{}{
				"description":    "en",
				"description_zh": "zh",
			},
		},
		"tools": []interface{}{
			map[string]interface{}{
				"tool_name":      "info_coin_get_coin_info",
				"description":    "en tool",
				"description_zh": "zh tool",
				"logic":          "stay",
			},
		},
	}
	out, err := stripInfoToolDescriptions(in)
	if err != nil {
		t.Fatal(err)
	}
	m := out.(map[string]interface{})
	keys := m["meta"].(map[string]interface{})["tool_entry_keys"].(map[string]interface{})
	if _, ok := keys["description"]; ok {
		t.Fatal("meta.tool_entry_keys should drop description")
	}
	tool := m["tools"].([]interface{})[0].(map[string]interface{})
	if _, ok := tool["description"]; ok {
		t.Fatal("tool should drop description")
	}
	if tool["logic"] != "stay" {
		t.Fatalf("logic kept: %v", tool["logic"])
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
