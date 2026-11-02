package cmd

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/gate/gate-cli/internal/agentfeature"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

type commandFlagContract struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Default  string `json:"default"`
	Required bool   `json:"required,omitempty"`
}

type commandContract struct {
	Path    string                `json:"path"`
	Use     string                `json:"use"`
	Aliases []string              `json:"aliases,omitempty"`
	Flags   []commandFlagContract `json:"flags"`
}

// The checked-in public command contract is shared with subprocess E2E tests.
// A new/removed command or changed flag must update both its contract and tests.
func loadCommandContracts(t *testing.T) []commandContract {
	t.Helper()
	raw, err := os.ReadFile("../tests/agentic/fixtures/command_contracts.json")
	require.NoError(t, err)
	var cases []commandContract
	require.NoError(t, json.Unmarshal(raw, &cases))
	require.Len(t, cases, 411)
	if agentfeature.Commands {
		raw, err := os.ReadFile("../tests/agentic/fixtures/agent_command_contracts.json")
		require.NoError(t, err)
		var agentCases []commandContract
		require.NoError(t, json.Unmarshal(raw, &agentCases))
		require.Len(t, agentCases, 3)
		cases = append(cases, agentCases...)
	}
	return cases
}

func TestAllPublicCommandContracts(t *testing.T) {
	var actual []string
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		// Startup creates Cobra's generic help/completion commands. Their own
		// historical tests own that UI; the frozen business inventory excludes them.
		if c.Parent() == rootCmd && (c.Name() == "help" || c.Name() == "completion") {
			return
		}
		if c.Hidden {
			return
		}
		// This inventory covers common and CEX commands; Intel has a separate inventory.
		if c.Parent() == rootCmd && (c.Name() == "info" || c.Name() == "news") {
			return
		}
		if c.Parent() == rootCmd && (c.Name() == "agent-leaves" || c.Name() == "agent-validate") {
			return // These two commands exclusively describe Intel capabilities.
		}
		if c.Runnable() && !c.HasSubCommands() {
			path := strings.TrimPrefix(c.CommandPath(), "gate-cli ")
			actual = append(actual, path)
		}
		for _, child := range c.Commands() {
			walk(child)
		}
	}
	walk(rootCmd)
	var expected []string
	for _, tc := range loadCommandContracts(t) {
		expected = append(expected, tc.Path)
		t.Run(tc.Path, func(t *testing.T) {
			c, rest, err := rootCmd.Find(strings.Fields(tc.Path))
			require.NoError(t, err)
			require.Empty(t, rest)
			require.Equal(t, tc.Use, c.Use)
			require.Equal(t, tc.Aliases, c.Aliases)
			var flags []commandFlagContract
			c.LocalNonPersistentFlags().VisitAll(func(f *pflag.Flag) {
				if !f.Hidden {
					flags = append(flags, commandFlagContract{f.Name, f.Value.Type(), f.DefValue, len(f.Annotations[cobra.BashCompOneRequiredFlag]) > 0})
				}
			})
			if flags == nil {
				flags = []commandFlagContract{}
			}
			require.Equal(t, tc.Flags, flags)
			for _, alias := range tc.Aliases {
				parts := strings.Fields(tc.Path)
				parts[len(parts)-1] = alias
				resolved, remaining, err := rootCmd.Find(parts)
				require.NoError(t, err)
				require.Empty(t, remaining)
				require.Same(t, c, resolved, "alias must preserve its canonical handler")
			}
		})
	}
	sort.Strings(actual)
	sort.Strings(expected)
	require.Equal(t, expected, actual, "public command inventory changed; add/remove corresponding business tests")
}
