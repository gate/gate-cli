package toolschema

// MissingRequiredArguments returns required schema fields not present in arguments.
// Empty string, empty array and nil are treated as missing.
func MissingRequiredArguments(arguments map[string]interface{}, schemaAny interface{}) []string {
	schema, ok := schemaAny.(map[string]interface{})
	if !ok {
		return nil
	}
	requiredRaw, ok := schema["required"].([]interface{})
	if !ok || len(requiredRaw) == 0 {
		return nil
	}
	missing := []string{}
	for _, r := range requiredRaw {
		key, _ := r.(string)
		if key == "" {
			continue
		}
		v, exists := arguments[key]
		if !exists || isMissingValue(v) {
			missing = append(missing, key)
		}
	}
	return missing
}

func isMissingValue(v interface{}) bool {
	if v == nil {
		return true
	}
	switch vv := v.(type) {
	case string:
		return vv == ""
	case []interface{}:
		return len(vv) == 0
	case []string:
		return len(vv) == 0
	}
	return false
}
