package cmdutil

import (
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"testing"
)

// Construction errors must fail immediately; valid declarations enforce user intent.
func TestMustRequireFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "write", RunE: func(*cobra.Command, []string) error { return nil }}
	cmd.Flags().String("amount", "", "")
	require.NotPanics(t, func() { MustRequireFlags(cmd, "amount") })
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	require.Error(t, err)
	require.Contains(t, err.Error(), `required flag(s) "amount" not set`)
	cmd.SetArgs([]string{"--amount", "1"})
	require.NoError(t, cmd.Execute())
	require.Panics(t, func() { MustRequireFlags(cmd, "missing") })
	require.NotPanics(t, func() { MustRequireFlags(cmd) })
}
