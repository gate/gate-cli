package intelfacade

import "testing"

// CR-830: drift in "required" vs "properties" breaks flag wiring at runtime; keep them aligned.
func TestInfoBaselineRequiredFieldsAreProperties(t *testing.T) {
	t.Parallel()
	for tool, sch := range InfoBaselineInputSchemas {
		assertRequiredKeysInProperties(t, tool, sch)
	}
}

func TestNewsBaselineRequiredFieldsAreProperties(t *testing.T) {
	t.Parallel()
	for tool, sch := range NewsBaselineInputSchemas {
		assertRequiredKeysInProperties(t, tool, sch)
	}
}

func assertRequiredKeysInProperties(t *testing.T, tool string, sch map[string]interface{}) {
	t.Helper()
	props, ok := sch["properties"].(map[string]interface{})
	if !ok {
		return
	}
	req, ok := sch["required"].([]interface{})
	if !ok || len(req) == 0 {
		return
	}
	for _, r := range req {
		key, ok := r.(string)
		if !ok {
			t.Fatalf("%s: required entry not a string: %v", tool, r)
		}
		if _, ok := props[key]; !ok {
			t.Fatalf("%s: required field %q missing from properties", tool, key)
		}
	}
}
