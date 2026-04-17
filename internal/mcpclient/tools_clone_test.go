package mcpclient

import (
	"testing"
)

func TestCloneToolsForList_IsolatesNestedInputSchema(t *testing.T) {
	orig := []Tool{{
		Name: "t1",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"nested": map[string]interface{}{"k": 1.0},
			},
		},
	}}
	out := cloneToolsForList(orig)
	m := out[0].InputSchema.(map[string]interface{})
	props := m["properties"].(map[string]interface{})
	nested := props["nested"].(map[string]interface{})
	nested["k"] = 99.0

	props0 := orig[0].InputSchema.(map[string]interface{})["properties"].(map[string]interface{})
	nested0 := props0["nested"].(map[string]interface{})
	if nested0["k"] != 1.0 {
		t.Fatalf("expected original nested k=1 after mutating clone, got %v", nested0["k"])
	}
}
