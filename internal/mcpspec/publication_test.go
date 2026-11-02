package mcpspec

import (
	"encoding/json"
	"os"
	"reflect"
	"regexp"
	"testing"
)

// Freeze interface structure independently of explanatory text: sanitization
// must retain every tool, parameter, enum, default, bound and response field.
func TestPublicSpecificationsPreserveInputContract(t *testing.T) {
	assertPublicInputContract(t)
}

func assertPublicInputContract(t *testing.T) {
	t.Helper()
	raw, err := os.ReadFile("testdata/public-input-contract.json")
	if err != nil {
		t.Fatal(err)
	}
	var want map[string]interface{}
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	for name, load := range map[string]func() (interface{}, error){"info": InfoInputsLogic, "news": NewsToolsArgs} {
		doc, err := load()
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(publicInputContract(doc), want[name]) {
			t.Errorf("%s interface structure changed", name)
		}
	}
}

func publicInputContract(v interface{}) interface{} {
	prose := map[string]bool{"meta": true, "source_of_truth": true, "source_repo": true, "logic": true, "request_type": true, "description": true, "description_zh": true, "notes": true, "note": true, "enum_note": true, "common_values_note": true, "response_note": true, "required_policy": true, "when": true, "series_fields_note": true}
	switch value := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{})
		for key, item := range value {
			if !prose[key] {
				out[key] = publicInputContract(item)
			}
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(value))
		for i, item := range value {
			out[i] = publicInputContract(item)
		}
		return out
	default:
		return v
	}
}

// The public bundle is also printed offline and in leaf help; private storage
// names and server source metadata must not be shipped inside the executable.
func TestPublicSpecificationsHideImplementationMetadata(t *testing.T) {
	private := regexp.MustCompile(`\b(?:dim|dwd|dws)_[a-zA-Z0-9_]+\b|source_of_truth|source_repo|internal/mcphost/`)
	for _, raw := range [][]byte{infoInputsLogicJSON, newsToolsArgsJSON} {
		if private.Match(raw) {
			t.Error("bundle contains implementation metadata")
		}
	}
}
