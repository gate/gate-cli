//go:build agent

package agentcmd_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gate/gate-cli/cmd"
	"github.com/gate/gate-cli/internal/cmdindex"
)

func TestAgentSearchCollectsCexLeaves(t *testing.T) {
	t.Parallel()
	var hasCex bool
	for _, c := range cmd.Root().Commands() {
		if c.Name() == "cex" {
			hasCex = true
			break
		}
	}
	require.True(t, hasCex, "cex command should be registered on root")
	hits := cmdindex.Search(cmdindex.CollectLeaves(cmd.Root()), "earn uni redeem records", 8)
	require.NotEmpty(t, hits, "expected earn uni records in search index")
}

func TestAgentSearchDomainInfo(t *testing.T) {
	t.Parallel()
	entries := cmdindex.FilterByDomain(cmdindex.CollectLeaves(cmd.Root()), "info")
	hits := cmdindex.Search(entries, "kline", 5)
	require.NotEmpty(t, hits)
	for _, h := range hits {
		require.True(t, strings.HasPrefix(h.Path, "info "), "path=%s", h.Path)
	}
}

func TestNewAgentSearchCmdRegistered(t *testing.T) {
	t.Parallel()
	var found bool
	for _, c := range cmd.Root().Commands() {
		if c.Name() == "agent-search" {
			found = true
			break
		}
	}
	require.True(t, found)
}
