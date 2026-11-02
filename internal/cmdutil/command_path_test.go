package cmdutil

import (
	"github.com/spf13/cobra"
	"strings"
	"testing"
)

// Validate help paths without consuming flag values or changing execution state.
func TestValidateCommandPath(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		bad  bool
	}{
		{"unknown", []string{"cex", "missing", "--help"}, true}, {"nested", []string{"cex", "missing", "market", "-h"}, true},
		{"unknown-root-deferred", []string{"missing", "--help"}, false},
		{"dynamic-completion-deferred", []string{"__complete", "cex", ""}, false}, {"false-help", []string{"cex", "missing", "--help=false"}, false},
		{"group", []string{"cex", "--help"}, false}, {"leaf", []string{"cex", "ticker", "--help"}, false},
		{"alias", []string{"exchange", "ticker", "-h"}, false}, {"value-is-command", []string{"--profile", "missing", "cex", "--help"}, false},
		{"slice", []string{"cex", "--tag", "first,second", "--help"}, false}, {"equals", []string{"--profile=missing", "cex", "--help=true"}, false},
		{"end-options", []string{"cex", "--", "--help"}, false}, {"unknown-flag-deferred", []string{"cex", "--bad-flag", "--help"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := &cobra.Command{Use: "gate-cli"}
			root.PersistentFlags().String("profile", "original", "")
			root.PersistentFlags().StringSlice("tag", nil, "")
			group := &cobra.Command{Use: "cex", Aliases: []string{"exchange"}, Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error { return c.Help() }}
			group.AddCommand(&cobra.Command{Use: "ticker", RunE: func(*cobra.Command, []string) error { return nil }})
			root.AddCommand(group)
			err := ValidateCommandPath(root, tc.args)
			if (err != nil) != tc.bad {
				t.Errorf("bad=%v error=%v", tc.bad, err)
			}
			if tc.bad && !strings.Contains(err.Error(), "unknown command") {
				t.Errorf("diagnostic=%v", err)
			}
			if root.PersistentFlags().Lookup("profile").Changed || root.PersistentFlags().Lookup("tag").Changed {
				t.Error("preflight changed live flags")
			}
			profile, _ := root.PersistentFlags().GetString("profile")
			tags, _ := root.PersistentFlags().GetStringSlice("tag")
			if profile != "original" || len(tags) != 0 {
				t.Errorf("preflight mutated values: profile=%s tags=%v", profile, tags)
			}
		})
	}
}

// Pure groups, runnable positional parents and parser-disabled commands retain their contracts.
func TestHelpGroupBoundaries(t *testing.T) {
	if err := ValidateCommandPath(nil, []string{"--help"}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name                    string
		runnable, disabled, bad bool
	}{
		{"pure-group", false, false, true}, {"positional-parent", true, false, false}, {"custom-parser", false, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := &cobra.Command{Use: "root"}
			group := &cobra.Command{Use: "group", DisableFlagParsing: tc.disabled}
			if tc.runnable {
				group.RunE = func(*cobra.Command, []string) error { return nil }
			}
			group.AddCommand(&cobra.Command{Use: "leaf"})
			root.AddCommand(group)
			err := ValidateCommandPath(root, []string{"group", "missing", "--help"})
			if (err != nil) != tc.bad {
				t.Errorf("bad=%v error=%v", tc.bad, err)
			}
		})
	}
}
