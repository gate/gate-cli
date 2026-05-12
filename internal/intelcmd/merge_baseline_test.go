package intelcmd

import (
	"testing"

	"github.com/gate/gate-cli/internal/intelfacade"
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

func TestMergeToolBaselineIntoStripsStaleRequiredWhenBaselineOmitsRequired(t *testing.T) {
	t.Parallel()
	out := map[string]toolschema.ToolSummary{
		"info_platformmetrics_get_platform_history": {
			Name:           "info_platformmetrics_get_platform_history",
			HasInputSchema: true,
			InputSchema: map[string]interface{}{
				"type":       "object",
				"required":   []interface{}{"platform_name"},
				"properties": map[string]interface{}{"platform_name": map[string]interface{}{"type": "string"}},
			},
		},
	}
	names := []string{"info_platformmetrics_get_platform_history"}
	baselineFor := func(name string) map[string]interface{} {
		return map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{"platform_name": map[string]interface{}{"type": "string"}},
		}
	}
	MergeToolBaselineInto(out, names, baselineFor)
	s := out["info_platformmetrics_get_platform_history"]
	m := s.InputSchema.(map[string]interface{})
	if _, ok := m["required"]; ok {
		t.Fatalf("expected stale required removed, got %#v", m["required"])
	}
}

func TestInputSchemaForMissingRequiredCheck_InfoPlatformHistoryStaleRequired(t *testing.T) {
	t.Parallel()
	server := map[string]interface{}{
		"type":     "object",
		"required": []interface{}{"platform_name"},
		"properties": map[string]interface{}{
			"platform_name": map[string]interface{}{"type": "string"},
			"exchange_slug": map[string]interface{}{"type": "string"},
		},
	}
	patched := InputSchemaForMissingRequiredCheck("info", "info_platformmetrics_get_platform_history", server)
	pm, ok := patched.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map schema, got %T", patched)
	}
	if _, ok := pm["required"]; ok {
		t.Fatalf("expected required stripped for MissingRequiredArguments check")
	}
	args := map[string]interface{}{"exchange_slug": "binance"}
	if miss := toolschema.MissingRequiredArguments(args, patched); len(miss) != 0 {
		t.Fatalf("unexpected missing: %v", miss)
	}
	// Baseline still omits top-level required (conditional_required in spec only).
	bl := intelfacade.InfoBaselineInputSchema("info_platformmetrics_get_platform_history")
	if _, ok := bl["required"]; ok {
		t.Fatalf("baseline must omit top-level required for this tool")
	}
}
