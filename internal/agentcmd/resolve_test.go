//go:build agent

package agentcmd_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gate/gate-cli/cmd"
	"github.com/gate/gate-cli/internal/cmdhint"
)

func TestNewAgentResolveCmdRegistered(t *testing.T) {
	t.Parallel()
	var found bool
	for _, c := range cmd.Root().Commands() {
		if c.Name() == "agent-resolve" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestMatchAgentLeavesKline(t *testing.T) {
	t.Parallel()
	got := cmdhint.MatchAgentLeaves("market kline BTC", 1)
	require.NotEmpty(t, got)
	require.Equal(t, "market_kline", got[0].Intent)
}
