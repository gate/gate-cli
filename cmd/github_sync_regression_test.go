package cmd

import (
	"strings"
	"testing"

	"github.com/gate/gate-cli/internal/intelcmd"
	"github.com/gate/gate-cli/internal/toolargs"
	"github.com/gate/gate-cli/tests/contracts"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

// The two histories independently moved CEX commands. Check the combined tree,
// including hidden Intel invokes, so a merge cannot silently shadow a sibling.
func TestGitHubSyncCommandOwnership(t *testing.T) {
	var walk func(*cobra.Command)
	walk = func(parent *cobra.Command) {
		owners := map[string]*cobra.Command{}
		for _, child := range parent.Commands() {
			for _, name := range append([]string{child.Name()}, child.Aliases...) {
				if owner, exists := owners[name]; exists {
					require.Same(t, owner, child, "ambiguous %s child %q", parent.CommandPath(), name)
				}
				owners[name] = child
			}
			walk(child)
		}
	}
	walk(rootCmd)
	for _, path := range []string{"config check", "version", "cex auth login", "cex stock order create", "cex futures trail log", "info invoke", "news invoke"} {
		c, rest, err := rootCmd.Find(strings.Fields(path))
		require.NoError(t, err)
		require.Empty(t, rest, "required command lost: %s", path)
		require.True(t, c.Runnable(), path)
	}
	for _, path := range []string{"cex launch", "cex sub-account", "cex p2p", "cex cross-ex", "cex welfare"} {
		_, rest, err := rootCmd.Find(strings.Fields(path))
		require.True(t, err != nil || len(rest) != 0, "retired group restored: %s", path)
	}
}

// Every public Intel leaf must retain its independent wire-name binding; a
// valid invoke test alone cannot detect an omitted or misregistered leaf.
func TestGitHubSyncIntelLeafContracts(t *testing.T) {
	want := map[string]string{}
	for _, c := range contracts.IntelCases(t) {
		leaf, rest, err := rootCmd.Find(strings.Fields(c.Path))
		require.NoError(t, err)
		require.Empty(t, rest, c.Path)
		require.True(t, leaf.Runnable(), c.Path)
		require.Equal(t, c.Tool, leaf.Annotations[intelcmd.AnnotationIntelToolName], c.Path)
		want[c.Tool] = c.Path
	}
	seen := map[string]bool{}
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		if tool := c.Annotations[intelcmd.AnnotationIntelToolName]; tool != "" {
			require.Contains(t, want, tool, "leaf has no independent process contract")
			require.False(t, seen[tool], "wire tool bound to multiple canonical leaves: %s", tool)
			seen[tool] = true
		}
		for _, child := range c.Commands() {
			walk(child)
		}
	}
	walk(rootCmd)
	require.Len(t, seen, 50)
}

// Use the actual inherited root flags with the GitHub Intel argument merger.
// Trading credentials are CLI controls; explicit JSON keys retain their value.
func TestGitHubSyncIntelGlobalIsolation(t *testing.T) {
	for _, domain := range []string{"info", "news"} {
		for _, mode := range []string{"normal", "explicit JSON", "malformed JSON"} {
			t.Run(domain+"/"+mode, func(t *testing.T) {
				c, rest, err := rootCmd.Find([]string{domain, "invoke"})
				require.NoError(t, err)
				require.Empty(t, rest)
				seen := map[*pflag.Flag]bool{}
				for p := c; p != nil; p = p.Parent() {
					save := func(f *pflag.Flag) {
						if seen[f] {
							return
						}
						seen[f] = true
						original, changed := f.Value.String(), f.Changed
						if slice, ok := f.Value.(pflag.SliceValue); ok {
							values := append([]string(nil), slice.GetSlice()...)
							t.Cleanup(func() { require.NoError(t, slice.Replace(values)); f.Changed = changed })
						} else {
							t.Cleanup(func() { require.NoError(t, f.Value.Set(original)); f.Changed = changed })
						}
					}
					p.Flags().VisitAll(save)
					p.PersistentFlags().VisitAll(save)
				}
				raw := `{"query":"BTC"}`
				if mode == "explicit JSON" {
					raw = `{"query":"BTC","profile":"business-profile"}`
				}
				if mode == "malformed JSON" {
					raw = `{"query":`
				}
				require.NoError(t, c.ParseFlags([]string{"--name", "fixture-tool", "--args-json", raw, "--format", "json", "--debug", "--verbose", "--max-output-bytes", "1024", "--profile", "trading-profile", "--api-key", "synthetic-key", "--api-secret", "synthetic-secret"}))
				args, err := toolargs.MergeFromCommand(c, toolargs.MergeOptions{ReservedFlags: intelcmd.ReservedInvokeFlags()})
				if mode == "malformed JSON" {
					require.Error(t, err)
					require.Nil(t, args)
					return
				}
				require.NoError(t, err)
				want := map[string]interface{}{"query": "BTC"}
				if mode == "explicit JSON" {
					want["profile"] = "business-profile"
				}
				require.Equal(t, want, args)
			})
		}
	}
}
