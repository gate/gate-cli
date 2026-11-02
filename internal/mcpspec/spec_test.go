package mcpspec

import (
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

// The tracked public contract is authoritative; local drafts cannot change it.
func TestBundledMatchesSpecs(t *testing.T) {
	assertPublicInputContract(t)
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
