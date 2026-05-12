package intelcmd

import (
	"github.com/gate/gate-cli/internal/intelfacade"
	"github.com/gate/gate-cli/internal/toolschema"
)

func baselineInputSchemaForMissingCheck(backend, toolName string) map[string]interface{} {
	switch backend {
	case "info":
		return intelfacade.InfoBaselineInputSchema(toolName)
	case "news":
		return intelfacade.NewsBaselineInputSchema(toolName)
	default:
		return nil
	}
}

// InputSchemaForMissingRequiredCheck returns schema for toolschema.MissingRequiredArguments.
// When the committed baseline omits top-level JSON Schema "required" but MCP describe/list
// still carries a stale "required" array, this returns a shallow clone with "required" removed
// so XOR / conditional_required tools (e.g. platform history) stay callable with exchange_slug only.
func InputSchemaForMissingRequiredCheck(backend, toolName string, schema interface{}) interface{} {
	baseline := baselineInputSchemaForMissingCheck(backend, toolName)
	if baseline == nil {
		return schema
	}
	m, ok := schema.(map[string]interface{})
	if !ok {
		return schema
	}
	cloned := shallowCloneStringMap(m)
	if patchInputSchemaRequiredFromBaseline(cloned, baseline) {
		return cloned
	}
	return schema
}

func shallowCloneStringMap(m map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// patchInputSchemaRequiredFromBaseline drops stale JSON Schema "required" from MCP cache
// when the committed baseline intentionally omits "required" (e.g. XOR / conditional fields
// documented only in specs). When baseline lists "required", the cached server array is kept
// so we do not strip server-only constraints.
func patchInputSchemaRequiredFromBaseline(dst map[string]interface{}, baseline map[string]interface{}) bool {
	if _, baselineHas := baseline["required"]; baselineHas {
		return false
	}
	if _, had := dst["required"]; had {
		delete(dst, "required")
		return true
	}
	return false
}

// MergeToolBaselineInto fills or patches entries in out using static baseline schemas (CR-811).
func MergeToolBaselineInto(out map[string]toolschema.ToolSummary, toolNames []string, baselineFor func(string) map[string]interface{}) {
	for _, name := range toolNames {
		baseline := baselineFor(name)
		if baseline == nil {
			continue
		}
		existing, ok := out[name]
		if !ok {
			out[name] = toolschema.ToolSummary{
				Name:           name,
				HasInputSchema: true,
				InputSchema:    baseline,
			}
			continue
		}
		if !existing.HasInputSchema || toolschema.IsEmptyInputSchema(existing.InputSchema) {
			existing.HasInputSchema = true
			existing.InputSchema = baseline
			out[name] = existing
			continue
		}
		if m, ok := existing.InputSchema.(map[string]interface{}); ok {
			cloned := shallowCloneStringMap(m)
			if patchInputSchemaRequiredFromBaseline(cloned, baseline) {
				existing.InputSchema = cloned
				out[name] = existing
			}
		}
	}
}
