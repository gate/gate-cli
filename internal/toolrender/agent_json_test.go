//go:build agent

package toolrender

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/gate/gate-cli/internal/output"
)

func TestPrintJSONToolResultAgentIncludesMeta(t *testing.T) {
	t.Setenv("GATE_CLI_AGENT", "1")
	var out, errOut bytes.Buffer
	p := output.NewWithStderr(&out, &errOut, output.FormatJSON)
	env := map[string]interface{}{
		"data": map[string]interface{}{"ok": true},
		"meta": map[string]interface{}{
			"freshness_hints": []string{"stale"},
		},
	}
	if err := printJSONToolResult(p, env); err != nil {
		t.Fatal(err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}
	if _, ok := parsed["meta"]; !ok {
		t.Fatalf("want meta wrapper, got %s", out.String())
	}
}

func TestPrintJSONToolResultAgentEmptyMeta(t *testing.T) {
	t.Setenv("GATE_CLI_AGENT", "1")
	var out bytes.Buffer
	p := output.NewWithStderr(&out, &bytes.Buffer{}, output.FormatJSON)
	env := map[string]interface{}{
		"data": map[string]interface{}{"ok": true},
	}
	if err := printJSONToolResult(p, env); err != nil {
		t.Fatal(err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		t.Fatal(err)
	}
	if _, ok := parsed["meta"]; !ok {
		t.Fatalf("agent mode should always include meta key: %s", out.String())
	}
}
