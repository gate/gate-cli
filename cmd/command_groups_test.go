package cmd

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// The reviewed baseline inventory is independent of validation logic. New groups
// must update the shared unit/E2E contract instead of silently losing coverage.
func TestCommandGroupInventory(t *testing.T) {
	raw, err := os.ReadFile("../tests/agentic/fixtures/command_groups.json")
	require.NoError(t, err)
	var fixture [][]string
	require.NoError(t, json.Unmarshal(raw, &fixture))
	require.Greater(t, len(fixture), 84)
	setupRootForExecute()
	rootCmd.InitDefaultCompletionCmd()
	var actual, expected []string
	var walk func(*cobra.Command, []string)
	walk = func(c *cobra.Command, path []string) {
		if c != rootCmd && !c.IsAvailableCommand() {
			return
		}
		if !c.HasSubCommands() {
			return
		}
		require.False(t, c.DisableFlagParsing, "group cannot bypass shared parsing: %s", c.CommandPath())
		if c.Runnable() {
			require.NotNil(t, c.Args, "runnable group must declare its positional contract: %s", c.CommandPath())
		}
		actual = append(actual, strings.Join(path, " "))
		for _, child := range c.Commands() {
			next := append(append([]string(nil), path...), child.Name())
			walk(child, next)
		}
	}
	walk(rootCmd, nil)
	for _, path := range fixture {
		expected = append(expected, strings.Join(path, " "))
	}
	sort.Strings(actual)
	sort.Strings(expected)
	require.Equal(t, expected, actual)
}
