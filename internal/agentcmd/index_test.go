//go:build agent

package agentcmd_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gate/gate-cli/cmd"
	"github.com/gate/gate-cli/internal/cmdindex"
)

func TestNewAgentIndexCmdRegistered(t *testing.T) {
	t.Parallel()
	var found bool
	for _, c := range cmd.Root().Commands() {
		if c.Name() == "agent-index" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestAgentIndexHasManyLeaves(t *testing.T) {
	t.Parallel()
	leaves := cmdindex.CollectLeaves(cmd.Root())
	require.Greater(t, len(leaves), 100, "expected substantial leaf catalog")
}
