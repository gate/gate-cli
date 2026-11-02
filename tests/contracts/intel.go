package contracts

import (
	_ "embed"
	"encoding/json"
	"testing"
)

// intelJSON freezes the public CLI path, remote name and independently authored
// valid inputs. Never generate expected routes from the runtime command tree.
//
//go:embed intel.json
var intelJSON []byte

//go:embed intel_boundaries.json
var intelBoundariesJSON []byte

// IntelCase is a complete argument contract shared by tree and process tests.
type IntelCase struct {
	Path           string                 `json:"path"`
	Tool           string                 `json:"tool"`
	Arguments      map[string]interface{} `json:"arguments"`
	Reply          map[string]interface{} `json:"reply"`
	PrettyContains []string               `json:"pretty_contains"`
}

// IntelCases returns fresh maps so a scenario cannot mutate another's fixture.
func IntelCases(t testing.TB) []IntelCase {
	t.Helper()
	var cases []IntelCase
	if err := json.Unmarshal(intelJSON, &cases); err != nil {
		t.Fatal(err)
	}
	return cases
}

// IntelBoundary carries a business expectation shared by validator unit tests
// and real-process tests; it never obtains expected validity from the validator.
type IntelBoundary struct {
	IntelCase
	Name    string `json:"name"`
	Invalid bool   `json:"invalid"`
}

// IntelBoundaries returns fresh inputs, including empty arrays and Unicode.
func IntelBoundaries(t testing.TB) []IntelBoundary {
	t.Helper()
	var cases []IntelBoundary
	if err := json.Unmarshal(intelBoundariesJSON, &cases); err != nil {
		t.Fatal(err)
	}
	return cases
}
