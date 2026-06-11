//go:build !agent

package agentcmd_test

import (
	"strings"
	"testing"

	"github.com/gate/gate-cli/cmd"
)

func TestNoAgentCommandsRegisteredByDefault(t *testing.T) {
	for _, c := range cmd.Root().Commands() {
		if strings.HasPrefix(c.Name(), "agent-") {
			t.Fatalf("unexpected agent command on root in default build: %s", c.Name())
		}
	}
}
