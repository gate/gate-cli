package wallet

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestRoot builds a minimal cobra root mirroring cmd/root.go's persistent
// flags so runXxx handlers can invoke cmdutil.GetClient/GetPrinter.
func newTestRoot(t *testing.T) *cobra.Command {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GATE_API_KEY", "")
	t.Setenv("GATE_API_SECRET", "")
	root := &cobra.Command{Use: "gate-cli"}
	root.PersistentFlags().String("format", "json", "")
	root.PersistentFlags().String("profile", "default", "")
	root.PersistentFlags().Bool("debug", false, "")
	root.PersistentFlags().Bool("verbose", false, "")
	root.PersistentFlags().String("api-key", "", "")
	root.PersistentFlags().String("api-secret", "", "")
	return root
}

const cobraBashCompOneRequiredFlag = "cobra_annotation_bash_completion_one_required_flag"

// findBalanceSub returns the wallet.balance.<name> leaf or nil.
func findBalanceSub(name string) (found bool, hasFlag func(string) bool, flagRequired func(string) bool) {
	for _, c := range Cmd.Commands() {
		if c.Name() != "balance" {
			continue
		}
		for _, sub := range c.Commands() {
			if sub.Name() != name {
				continue
			}
			return true,
				func(f string) bool { return sub.Flag(f) != nil },
				func(f string) bool {
					ff := sub.Flag(f)
					if ff == nil {
						return false
					}
					return len(ff.Annotations[cobraBashCompOneRequiredFlag]) > 0
				}
		}
	}
	return false, nil, nil
}

func TestWalletBalanceSubHasPaginationFlags(t *testing.T) {
	// After SDK v7.2.71 sync, `wallet balance sub` gained --page and --limit flags
	// backing the ListSubAccountBalancesOpts.Page/Limit fields.
	found, hasFlag, flagRequired := findBalanceSub("sub")
	require.True(t, found, "wallet balance sub subcommand should be registered")

	for _, name := range []string{"sub-uid", "page", "limit"} {
		assert.True(t, hasFlag(name), "wallet balance sub should expose --%s", name)
		assert.False(t, flagRequired(name), "--%s must remain optional", name)
	}
}

// TestRunWalletSubBalances_RequiresAuth drives the runner directly so the new
// --page / --limit flag plumbing is exercised before the auth gate trips, but
// still fails with a clear API-key error under an empty credential env.
func TestRunWalletSubBalances_RequiresAuth(t *testing.T) {
	root := newTestRoot(t)
	cmd := &cobra.Command{Use: "sub"}
	cmd.Flags().String("sub-uid", "12345", "")
	cmd.Flags().Int32("page", 2, "")
	cmd.Flags().Int32("limit", 50, "")
	root.AddCommand(cmd)

	err := runWalletSubBalances(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

// Zero-value page/limit must not cause the handler to panic or misconfigure
// opts; the auth gate should still be reached and returns the API-key error.
func TestRunWalletSubBalances_ZeroPaginationFlags(t *testing.T) {
	root := newTestRoot(t)
	cmd := &cobra.Command{Use: "sub"}
	cmd.Flags().String("sub-uid", "", "")
	cmd.Flags().Int32("page", 0, "")
	cmd.Flags().Int32("limit", 0, "")
	root.AddCommand(cmd)

	err := runWalletSubBalances(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

func TestWalletBalanceOtherSubcommandsStillPresent(t *testing.T) {
	// Guard against accidental regressions while adding new flags.
	want := []string{"sub", "sub-margin", "sub-futures", "sub-cross-margin", "small", "small-history"}
	for _, name := range want {
		found, _, _ := findBalanceSub(name)
		assert.True(t, found, "wallet balance should still expose %q", name)
	}
}
