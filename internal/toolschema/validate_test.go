package toolschema

import "testing"

func TestMissingRequiredArguments(t *testing.T) {
	schema := map[string]interface{}{
		"required": []interface{}{"query", "limit"},
	}
	args := map[string]interface{}{
		"query": "",
	}
	missing := MissingRequiredArguments(args, schema)
	if len(missing) != 2 {
		t.Fatalf("expected 2 missing args, got %v", missing)
	}
}

func TestMissingRequiredArguments_NoSchemaRequired(t *testing.T) {
	schema := map[string]interface{}{
		"properties": map[string]interface{}{},
	}
	args := map[string]interface{}{}
	missing := MissingRequiredArguments(args, schema)
	if len(missing) != 0 {
		t.Fatalf("expected no missing args, got %v", missing)
	}
}
