package mcpclient

import "encoding/json"

// cloneToolsForList returns a deep copy of tools so InputSchema maps are not shared
// with the JSON decoder's internal structures or across cache entries (CR-309).
func cloneToolsForList(in []Tool) []Tool {
	if len(in) == 0 {
		return []Tool{}
	}
	out := make([]Tool, len(in))
	for i := range in {
		out[i] = Tool{
			Name:        in[i].Name,
			Description: in[i].Description,
			InputSchema: cloneInputSchemaAny(in[i].InputSchema),
		}
	}
	return out
}

// cloneInputSchemaAny deep-copies schema-shaped JSON via round-trip (CR-308: no manual
// recursive walk; encoding/json applies safe nesting limits on decode).
func cloneInputSchemaAny(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var raw interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return v
	}
	return raw
}
