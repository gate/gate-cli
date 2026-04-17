package intelcmd

import (
	"testing"

	"github.com/gate/gate-cli/internal/toolschema"
)

func TestMergeToolBaselineIntoFillsMissing(t *testing.T) {
	t.Parallel()
	out := map[string]toolschema.ToolSummary{}
	names := []string{"t1"}
	baselineFor := func(name string) map[string]interface{} {
		if name != "t1" {
			return nil
		}
		return map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{"x": map[string]interface{}{"type": "string"}},
		}
	}
	MergeToolBaselineInto(out, names, baselineFor)
	s, ok := out["t1"]
	if !ok || !s.HasInputSchema || toolschema.IsEmptyInputSchema(s.InputSchema) {
		t.Fatalf("expected schema, got %#v", s)
	}
}
