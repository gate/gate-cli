package intelcmd

import "github.com/gate/gate-cli/internal/toolschema"

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
		}
	}
}
