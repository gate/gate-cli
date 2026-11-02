package cmdutil

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// Pure groups must reject unresolved words before Cobra's non-runnable help return.
// The parser must consume flags using their actual arity without touching live values.
func TestPureCommandGroupPaths(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		bad  bool
	}{
		{"unknown", []string{"cex", "nosuch"}, true},
		{"deep", []string{"cex", "market", "nosuch"}, true},
		{"alias", []string{"exchange", "market", "nosuch"}, true},
		{"relative-help", []string{"cex", "help", "market"}, true},
		{"false-help", []string{"cex", "market", "nosuch", "--help=false"}, true},
		{"false-short-help", []string{"cex", "market", "nosuch", "-h=false"}, true},
		{"separator", []string{"cex", "market", "--", "nosuch"}, true},
		{"literal-help-word", []string{"cex", "market", "--", "--help"}, true},
		{"flag-value-not-topic", []string{"--profile", "nosuch", "cex", "market", "bad-child"}, true},
		{"bare-root", []string{}, false}, {"bare-group", []string{"cex"}, false},
		{"bare-nested", []string{"cex", "market"}, false},
		{"valid-help", []string{"cex", "market", "-h"}, false},
		{"string-value", []string{"cex", "market", "--profile", "nosuch"}, false},
		{"slice-values", []string{"--tag", "first,second", "cex", "market", "--tag=nosuch"}, false},
		{"boolean-value", []string{"cex", "market", "--verbose=false"}, false},
		{"leaf-positionals", []string{"cex", "market", "query", "first", "second"}, false},
	} {
		for _, policy := range []string{"nil", "no-args", "arbitrary"} {
			t.Run(tc.name+"/"+policy, func(t *testing.T) {
				root := &cobra.Command{Use: "gate-cli", SilenceErrors: true, SilenceUsage: true}
				root.PersistentFlags().String("profile", "original", "")
				root.PersistentFlags().StringSlice("tag", nil, "")
				root.PersistentFlags().BoolP("verbose", "v", false, "")
				group := &cobra.Command{Use: "cex", Aliases: []string{"exchange"}}
				market := &cobra.Command{Use: "market"}
				for _, c := range []*cobra.Command{group, market} {
					switch policy {
					case "no-args":
						c.Args = cobra.NoArgs
					case "arbitrary":
						c.Args = cobra.ArbitraryArgs
					}
				}
				var calls, helps int
				var received []string
				market.AddCommand(&cobra.Command{Use: "query", Args: cobra.ExactArgs(2), Run: func(_ *cobra.Command, args []string) { calls++; received = args }})
				group.AddCommand(market)
				root.AddCommand(group)
				var out bytes.Buffer
				root.SetOut(&out)
				root.SetErr(&out)
				root.SetHelpFunc(func(*cobra.Command, []string) { helps++ })
				err := ValidateCommandPath(root, tc.args)
				for name, want := range map[string]string{"profile": "original", "tag": "[]", "verbose": "false"} {
					flag := root.PersistentFlags().Lookup(name)
					require.False(t, flag.Changed)
					require.Equal(t, want, flag.Value.String())
				}
				if tc.bad {
					require.ErrorContains(t, err, "unknown command")
					require.Zero(t, helps)
					require.Zero(t, calls)
					require.Empty(t, out.String())
					return
				}
				require.NoError(t, err)
				root.SetArgs(tc.args)
				require.NoError(t, root.Execute())
				if tc.name == "leaf-positionals" {
					require.Equal(t, []string{"first", "second"}, received)
					require.Equal(t, 1, calls)
					require.Zero(t, helps)
				} else {
					require.Zero(t, calls)
					require.Equal(t, 1, helps)
				}
			})
		}
	}
}

// Normal runnable parents own positional arguments; preflight cannot execute their Args twice.
func TestCommandPathPreservesArgumentOwners(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		root := &cobra.Command{Use: "root", SilenceErrors: true, SilenceUsage: true}
		var validations, runs int
		group := &cobra.Command{Use: "parent", DisableFlagParsing: disabled,
			Args: func(_ *cobra.Command, args []string) error { validations++; return cobra.ExactArgs(1)(nil, args) },
			Run:  func(_ *cobra.Command, args []string) { runs++; require.Equal(t, []string{"position"}, args) },
		}
		group.AddCommand(&cobra.Command{Use: "child"})
		root.AddCommand(group)
		args := []string{"parent", "position"}
		require.NoError(t, ValidateCommandPath(root, args))
		require.Zero(t, validations)
		root.SetArgs(args)
		require.NoError(t, root.Execute())
		require.Equal(t, 1, validations)
		require.Equal(t, 1, runs)
	}
	// Parse failures still belong to Cobra, rather than being mistaken for help success.
	for _, args := range [][]string{{"group", "--profile"}, {"group", "--bad-flag"}} {
		root := &cobra.Command{Use: "root", SilenceErrors: true, SilenceUsage: true}
		root.PersistentFlags().String("profile", "", "")
		group := &cobra.Command{Use: "group"}
		group.AddCommand(&cobra.Command{Use: "leaf"})
		root.AddCommand(group)
		require.NoError(t, ValidateCommandPath(root, args))
		root.SetArgs(args)
		require.Error(t, root.Execute())
	}
}

// Framework-generated completion is also a group; initialization must be idempotent
// and must respect the caller's disabled or custom completion contract.
func TestCommandPathCompletionPolicy(t *testing.T) {
	root := &cobra.Command{Use: "root", SilenceErrors: true, SilenceUsage: true}
	root.AddCommand(&cobra.Command{Use: "leaf", Run: func(*cobra.Command, []string) {}})
	for i := 0; i < 2; i++ {
		require.ErrorContains(t, ValidateCommandPath(root, []string{"completion", "nosuch"}), "unknown command")
	}
	var count int
	for _, c := range root.Commands() {
		if c.Name() == "completion" {
			count++
		}
	}
	require.Equal(t, 1, count)
	InstallStrictHelpCommand(root)
	require.NoError(t, ValidateCommandPath(root, []string{"help", "completion", "bash", "--help"}))
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"__complete", "leaf", ""})
	require.NoError(t, ValidateCommandPath(root, []string{"__complete", "leaf", ""}))
	require.NoError(t, root.Execute())
	require.Contains(t, out.String(), ":")

	for _, custom := range []bool{false, true} {
		root := &cobra.Command{Use: "root"}
		root.AddCommand(&cobra.Command{Use: "leaf"})
		if custom {
			own := &cobra.Command{Use: "completion", Args: cobra.ExactArgs(1), Run: func(*cobra.Command, []string) {}}
			root.AddCommand(own)
			require.NoError(t, ValidateCommandPath(root, []string{"completion", "position"}))
			require.False(t, own.HasSubCommands())
		} else {
			root.CompletionOptions.DisableDefaultCmd = true
			require.NoError(t, ValidateCommandPath(root, []string{"leaf", "--help"}))
			for _, c := range root.Commands() {
				require.NotEqual(t, "completion", c.Name())
			}
		}
	}
}

// Diagnostic wording remains the usual Cobra unknown-command contract.
func TestPureGroupDiagnosticPrefix(t *testing.T) {
	root := &cobra.Command{Use: "root"}
	group := &cobra.Command{Use: "group", Args: cobra.NoArgs}
	group.AddCommand(&cobra.Command{Use: "leaf"})
	root.AddCommand(group)
	err := ValidateCommandPath(root, []string{"group", "bad"})
	require.Error(t, err)
	require.True(t, strings.HasPrefix(err.Error(), `unknown command "bad" for "root group"`))
}
