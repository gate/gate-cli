package intelcmd

import "github.com/gate/gate-cli/internal/toolschema"

// LoadToolSchemasFromCache loads cached tool summaries for backend, then runs mergeBaseline
// on return (same defer order as legacy cmd/*/load*ToolSchemas) (CR-811).
func LoadToolSchemasFromCache(backend string, mergeBaseline func(out map[string]toolschema.ToolSummary)) map[string]toolschema.ToolSummary {
	out := map[string]toolschema.ToolSummary{}
	if mergeBaseline != nil {
		defer func() { mergeBaseline(out) }()
	}
	if cached, _, err := toolschema.LoadCache(backend); err == nil {
		for _, t := range cached {
			out[t.Name] = t
		}
	}
	return out
}
