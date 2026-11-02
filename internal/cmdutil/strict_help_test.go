package cmdutil

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// Exercise the real Cobra dispatch boundary, including the flag-help early return.
func TestStrictHelpCommand(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
		bad  bool
	}{
		{"unknown-root", []string{"help", "nosuch"}, "", true},
		{"unknown-deep", []string{"help", "cex", "missing"}, "", true},
		{"retired-path", []string{"help", "cex", "cross-ex", "market", "symbols"}, "", true},
		{"leaf-extra", []string{"help", "cex", "ticker", "extra"}, "", true},
		{"unknown-flag-help", []string{"help", "nosuch", "--help"}, "", true},
		{"deep-short-help", []string{"help", "cex", "missing", "-h"}, "", true},
		{"unknown-prefix-format", []string{"--format", "json", "help", "cex", "missing", "--help"}, "", true},
		{"empty", []string{"help"}, "gate-cli", false},
		{"group", []string{"help", "cex"}, "gate-cli cex", false},
		{"leaf-no-required-flags", []string{"help", "cex", "ticker"}, "gate-cli cex ticker", false},
		{"aliases", []string{"help", "exchange", "price"}, "gate-cli cex ticker", false},
		{"profile-value", []string{"--profile", "missing", "help", "cex", "ticker"}, "gate-cli cex ticker", false},
		{"format-value", []string{"help", "cex", "ticker", "--format=json"}, "gate-cli cex ticker", false},
		{"false-help", []string{"help", "cex", "ticker", "--help=false"}, "gate-cli cex ticker", false},
		{"separator", []string{"help", "--", "cex", "ticker"}, "gate-cli cex ticker", false},
		{"separator-unknown", []string{"help", "--", "--help"}, "", true},
		{"help-command-help", []string{"help", "cex", "ticker", "--help"}, "gate-cli help", false},
		{"ordinary-flag-help", []string{"cex", "ticker", "--help"}, "gate-cli cex ticker", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := &cobra.Command{Use: "gate-cli", SilenceErrors: true, SilenceUsage: true}
			root.PersistentFlags().String("profile", "original", "")
			root.PersistentFlags().String("format", "original", "")
			group := &cobra.Command{Use: "cex", Aliases: []string{"exchange"}, Args: cobra.NoArgs}
			var businessCalls, helps int
			leaf := &cobra.Command{Use: "ticker", Aliases: []string{"price"},
				Args:   func(*cobra.Command, []string) error { businessCalls++; return fmt.Errorf("business args must not run") },
				PreRun: func(*cobra.Command, []string) { businessCalls++ },
				Run:    func(*cobra.Command, []string) { businessCalls++ },
			}
			leaf.Flags().String("currency", "", "")
			require.NoError(t, leaf.MarkFlagRequired("currency"))
			group.AddCommand(leaf)
			root.AddCommand(group)
			var rendered bytes.Buffer
			root.SetOut(&rendered)
			root.SetErr(&rendered)
			root.SetContext(context.WithValue(context.Background(), strictHelpContextKey{}, "flowed"))
			root.SetHelpFunc(func(c *cobra.Command, _ []string) {
				helps++
				require.Equal(t, "flowed", c.Context().Value(strictHelpContextKey{}))
				_, err := fmt.Fprint(&rendered, c.CommandPath())
				require.NoError(t, err)
			})
			InstallStrictHelpCommand(root)
			first := rootHelpCommand(root)
			InstallStrictHelpCommand(root)
			require.Same(t, first, rootHelpCommand(root))
			var helpCount int
			for _, c := range root.Commands() {
				if c.Name() == "help" {
					helpCount++
				}
			}
			require.Equal(t, 1, helpCount)
			err := ValidateCommandPath(root, tc.args)
			for _, name := range []string{"profile", "format"} {
				flag := root.PersistentFlags().Lookup(name)
				require.False(t, flag.Changed)
				require.Equal(t, "original", flag.Value.String())
			}
			if err == nil {
				root.SetArgs(tc.args)
				err = root.Execute()
			}
			if tc.bad {
				require.ErrorContains(t, err, "unknown command")
				require.Zero(t, helps)
				require.Empty(t, rendered.String())
			} else {
				require.NoError(t, err)
				require.Equal(t, 1, helps)
				require.Equal(t, tc.want, rendered.String())
			}
			require.Zero(t, businessCalls)
		})
	}
}

type strictHelpContextKey struct{}

// Keep Cobra's default help completion callback and dynamically registered __complete.
func TestStrictHelpCompletion(t *testing.T) {
	InstallStrictHelpCommand(nil)
	InstallStrictHelpCommand(&cobra.Command{Use: "empty"})
	root := &cobra.Command{Use: "gate-cli"}
	group := &cobra.Command{Use: "cex"}
	group.AddCommand(&cobra.Command{Use: "ticker", Short: "ticker details", Run: func(*cobra.Command, []string) {}})
	root.AddCommand(group)
	InstallStrictHelpCommand(root)
	help := rootHelpCommand(root)
	require.NotNil(t, help.ValidArgsFunction)
	completions, directive := help.ValidArgsFunction(help, []string{"cex"}, "tick")
	if len(completions) != 1 || !strings.HasPrefix(completions[0], "ticker") || directive != cobra.ShellCompDirectiveNoFileComp {
		t.Fatalf("completions=%v directive=%v", completions, directive)
	}
	args := []string{"__complete", "help", "cex", ""}
	require.NoError(t, ValidateCommandPath(root, args))
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	require.NoError(t, root.Execute())
	require.Contains(t, out.String(), "ticker")
}
