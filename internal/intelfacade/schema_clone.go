package intelfacade

const maxSchemaCloneDepth = 64

func deepCloneJSONSchemaValue(v interface{}, depth int) interface{} {
	if depth > maxSchemaCloneDepth {
		return v
	}
	switch x := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(x))
		for k, vv := range x {
			out[k] = deepCloneJSONSchemaValue(vv, depth+1)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(x))
		for i, vv := range x {
			out[i] = deepCloneJSONSchemaValue(vv, depth+1)
		}
		return out
	default:
		return x
	}
}

func deepCloneSchemaMap(in map[string]interface{}) map[string]interface{} {
	if in == nil {
		return nil
	}
	out := deepCloneJSONSchemaValue(in, 0)
	m, ok := out.(map[string]interface{})
	if !ok {
		return nil
	}
	return m
}
