//go:build !agent

package agentfeature

import "testing"

func TestCommandsDisabledByDefault(t *testing.T) {
	if Commands {
		t.Fatal("expected Commands=false in default builds (no -tags agent)")
	}
}
