package version

import (
	"encoding/json"
	"github.com/gate/gate-cli/internal/agentfeature"
	"runtime"
	"testing"
)

// Details is a bounded whitelist, never a dump of build paths or dependencies.
func TestCEXCompatVersionDetails(t *testing.T) {
	d := Details()
	raw, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if json.Unmarshal(raw, &fields) != nil || len(fields) != 7 {
		t.Fatalf("fields %s", raw)
	}
	if fields["version"] != Version || fields["go"] != runtime.Version() || fields["os"] != runtime.GOOS || fields["arch"] != runtime.GOARCH || fields["agent_commands"] != agentfeature.Commands || fields["revision"] == "" {
		t.Fatalf("details %s", raw)
	}
}
