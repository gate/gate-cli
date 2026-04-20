package intelfacade

import (
	"testing"

	"github.com/gate/gate-cli/internal/mcpspec"
)

// Baseline-only CLI ergonomics keys not present in MCP spec fields.
var infoBaselineExtraSpecKeys = map[string]map[string]struct{}{
	"info_coin_get_coin_info": {"symbol": {}},
}

// TestInfoBaselineCoversSpecInputFields ensures every spec input field for inventory
// tools has a matching baseline property so cobra -h lists the flag (CR-815).
func TestInfoBaselineCoversSpecInputFields(t *testing.T) {
	t.Parallel()
	doc, err := mcpspec.InfoInputsLogic()
	if err != nil {
		t.Fatal(err)
	}
	root, ok := doc.(map[string]interface{})
	if !ok {
		t.Fatalf("spec root type %T", doc)
	}
	raw, ok := root["tools"].([]interface{})
	if !ok {
		t.Fatal("spec missing tools array")
	}
	specFieldsByTool := make(map[string][]string, len(raw))
	for _, item := range raw {
		tm, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := tm["tool_name"].(string)
		if name == "" {
			continue
		}
		fields, _ := tm["fields"].([]interface{})
		var names []string
		for _, f := range fields {
			fm, ok := f.(map[string]interface{})
			if !ok {
				continue
			}
			n, _ := fm["name"].(string)
			if n != "" {
				names = append(names, n)
			}
		}
		specFieldsByTool[name] = names
	}

	for _, tool := range InfoToolBaseline {
		specNames, ok := specFieldsByTool[tool]
		if !ok {
			t.Fatalf("tool %q in InfoToolBaseline but absent from MCP spec JSON", tool)
		}
		if len(specNames) == 0 && tool != "info_marketsnapshot_get_market_overview" && tool != "info_macro_get_macro_summary" {
			t.Fatalf("spec has empty fields for tool %q (unexpected)", tool)
		}
		bl := InfoBaselineInputSchema(tool)
		if bl == nil {
			t.Fatalf("missing baseline schema for %s", tool)
		}
		props, ok := bl["properties"].(map[string]interface{})
		if !ok {
			t.Fatalf("%s: baseline missing properties", tool)
		}
		for _, fn := range specNames {
			if _, ok := props[fn]; !ok {
				t.Errorf("%s: spec field %q missing from InfoBaselineInputSchemas (will not appear in -h)", tool, fn)
			}
		}
		extraOK := infoBaselineExtraSpecKeys[tool]
		for k := range props {
			if extraOK != nil {
				if _, ok := extraOK[k]; ok {
					continue
				}
			}
			found := false
			for _, fn := range specNames {
				if fn == k {
					found = true
					break
				}
			}
			if !found {
				t.Logf("%s: baseline-only property %q (not in spec fields)", tool, k)
			}
		}
	}
}

func TestInfoSpecExtraToolsDocumented(t *testing.T) {
	t.Parallel()
	doc, err := mcpspec.InfoInputsLogic()
	if err != nil {
		t.Fatal(err)
	}
	root := doc.(map[string]interface{})
	raw := root["tools"].([]interface{})
	specTools := make(map[string]struct{}, len(raw))
	for _, item := range raw {
		tm := item.(map[string]interface{})
		if n, _ := tm["tool_name"].(string); n != "" {
			specTools[n] = struct{}{}
		}
	}
	var missingInInventory []string
	for name := range specTools {
		found := false
		for _, inv := range InfoToolBaseline {
			if inv == name {
				found = true
				break
			}
		}
		if !found {
			missingInInventory = append(missingInInventory, name)
		}
	}
	if len(missingInInventory) > 0 {
		t.Logf("spec documents tools not in InfoToolBaseline (no CLI leaf -h): %v", missingInInventory)
	}
}
