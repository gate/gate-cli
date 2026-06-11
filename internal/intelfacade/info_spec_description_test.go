package intelfacade

import (
	"strings"
	"testing"

	"github.com/gate/gate-cli/internal/mcpspec"
)

func TestInfoBaselineToolsHaveEnglishDescription(t *testing.T) {
	t.Parallel()
	doc, err := mcpspec.InfoInputsLogic()
	if err != nil {
		t.Fatal(err)
	}
	root := doc.(map[string]interface{})
	byName := map[string]map[string]interface{}{}
	for _, item := range root["tools"].([]interface{}) {
		tm := item.(map[string]interface{})
		if n, _ := tm["tool_name"].(string); n != "" {
			byName[n] = tm
		}
	}
	for _, tool := range InfoToolBaseline {
		tm, ok := byName[tool]
		if !ok {
			t.Fatalf("missing spec entry for %s", tool)
		}
		desc, _ := tm["description"].(string)
		if strings.TrimSpace(desc) == "" {
			t.Errorf("%s: empty description", tool)
		}
		if !strings.HasPrefix(strings.TrimSpace(desc), "[Read]") {
			t.Errorf("%s: description should start with [Read]: %q", tool, desc)
		}
		if zh, _ := tm["description_zh"].(string); strings.TrimSpace(zh) != "" {
			t.Errorf("%s: embedded bundled spec must not ship description_zh", tool)
		}
	}
}
