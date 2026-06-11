//go:build agent

package agentcmd_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gate/gate-cli/cmd"
	"github.com/gate/gate-cli/internal/cmdhint"
)

func TestNewAgentLeavesCmdRegistered(t *testing.T) {
	t.Parallel()
	var found bool
	for _, c := range cmd.Root().Commands() {
		if c.Name() == "agent-leaves" {
			found = true
			break
		}
	}
	require.True(t, found, "agent-leaves should be registered on root")
}

func TestAgentLeavesPayloadShape(t *testing.T) {
	t.Parallel()
	require.Len(t, cmdhint.AgentLeaves, 31)
	require.Len(t, cmdhint.BaselineMCPCatalog(), 46)
	require.Equal(t, "market_kline", cmdhint.AgentLeaves[0].Intent)
	require.Contains(t, cmdhint.AgentLeaves[0].Command, "info markettrend get-kline")
	require.Equal(t, 200, cmdhint.AgentLeaves[0].DefaultLimit)
	for _, leaf := range cmdhint.AgentLeaves {
		require.Contains(t, []string{"info", "news"}, leaf.RequiredPrefix)
	}
	for _, leaf := range cmdhint.BaselineMCPCatalog() {
		require.NotContains(t, leaf.Intent, "_", "mcp_catalog intent must not expose MCP wire names")
		require.True(t, strings.Contains(leaf.Command, "gate-cli"), leaf.Command)
	}
}
