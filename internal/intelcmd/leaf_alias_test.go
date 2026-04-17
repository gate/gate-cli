package intelcmd

import (
	"strings"
	"testing"

	"github.com/gate/gate-cli/internal/toolschema"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestNewLeafAliasCommandExampleMentionsBackend(t *testing.T) {
	c := NewLeafAliasCommand(LeafAliasConfig{
		BackendCLI: "info",
		Use:        "shortcut",
		ToolName:   "info_group_toolname",
		RunE: func(*cobra.Command, []string) error {
			return nil
		},
	})
	require.Contains(t, c.Example, "gate-cli info")
	require.True(t, strings.Contains(c.Example, "shortcut"))
}

func TestLoadToolSchemasFromCacheInvokesMerge(t *testing.T) {
	var saw bool
	out := LoadToolSchemasFromCache("info", func(m map[string]toolschema.ToolSummary) {
		saw = true
		m["probe"] = toolschema.ToolSummary{Name: "probe"}
	})
	require.True(t, saw)
	require.Contains(t, out, "probe")
}
