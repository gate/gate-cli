package intelfacade

import (
	"testing"

	"github.com/gate/gate-cli/internal/mcpspec"
)

// TestNewsBaselineCoversSpecParams ensures each news tool's spec input_rules.params
// names have a matching baseline property (so cobra -h lists the flag).
func TestNewsBaselineCoversSpecParams(t *testing.T) {
	t.Parallel()
	doc, err := mcpspec.NewsToolsArgs()
	if err != nil {
		t.Fatal(err)
	}
	root, ok := doc.(map[string]interface{})
	if !ok {
		t.Fatalf("news spec root type %T", doc)
	}
	raw, ok := root["tools"].([]interface{})
	if !ok {
		t.Fatal("news spec missing tools")
	}
	for _, item := range raw {
		tm, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := tm["name"].(string)
		if name == "" {
			continue
		}
		inBaseline := false
		for _, inv := range NewsToolBaseline {
			if inv == name {
				inBaseline = true
				break
			}
		}
		if !inBaseline {
			continue
		}
		ir, ok := tm["input_rules"].(map[string]interface{})
		if !ok {
			t.Fatalf("%s: missing input_rules", name)
		}
		params, ok := ir["params"].([]interface{})
		if !ok {
			t.Fatalf("%s: missing params", name)
		}
		bl := NewsBaselineInputSchema(name)
		if bl == nil {
			t.Fatalf("missing news baseline for %s", name)
		}
		props, ok := bl["properties"].(map[string]interface{})
		if !ok {
			t.Fatalf("%s: baseline missing properties", name)
		}
		for _, p := range params {
			pm, ok := p.(map[string]interface{})
			if !ok {
				continue
			}
			pn, _ := pm["name"].(string)
			if pn == "" {
				continue
			}
			if _, ok := props[pn]; !ok {
				t.Errorf("%s: spec param %q missing from NewsBaselineInputSchemas", name, pn)
			}
		}
	}
}
