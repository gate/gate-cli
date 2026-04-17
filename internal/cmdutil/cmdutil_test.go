package cmdutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gate/gate-cli/internal/cmdutil"
)

// newRootCmd builds a minimal root command with the same persistent flags
// that cmd/root.go registers, so GetClient and GetSettle see a realistic
// cobra command tree.
func newRootCmd() *cobra.Command {
	root := &cobra.Command{Use: "gate-cli"}
	root.PersistentFlags().String("format", "table", "")
	root.PersistentFlags().String("profile", "default", "")
	root.PersistentFlags().Bool("debug", false, "")
	root.PersistentFlags().Bool("verbose", false, "")
	root.PersistentFlags().String("api-key", "", "")
	root.PersistentFlags().String("api-secret", "", "")
	return root
}

// newChildCmd attaches a leaf command to root and returns it.
// Tests call GetClient/GetSettle on the child so cmd.Root() resolves correctly.
func newChildCmd(root *cobra.Command) *cobra.Command {
	child := &cobra.Command{Use: "sub"}
	root.AddCommand(child)
	return child
}

func TestIntelMCPTransportDiag_DebugOrVerbose(t *testing.T) {
	root := newRootCmd()
	child := newChildCmd(root)

	en, tag := cmdutil.IntelMCPTransportDiag(child)
	assert.False(t, en)
	assert.Equal(t, "", tag)

	require.NoError(t, root.PersistentFlags().Set("verbose", "true"))
	en, tag = cmdutil.IntelMCPTransportDiag(child)
	assert.True(t, en)
	assert.Equal(t, "[verbose]", tag)

	require.NoError(t, root.PersistentFlags().Set("verbose", "false"))
	require.NoError(t, root.PersistentFlags().Set("debug", "true"))
	en, tag = cmdutil.IntelMCPTransportDiag(child)
	assert.True(t, en)
	assert.Equal(t, "[debug]", tag)

	require.NoError(t, root.PersistentFlags().Set("verbose", "true"))
	en, tag = cmdutil.IntelMCPTransportDiag(child)
	assert.True(t, en)
	assert.Equal(t, "[debug]", tag)
}

// --- GetClient flag priority ---

func TestGetClient_UsesFlagCredentials(t *testing.T) {
	// Verify that --api-key/--api-secret cobra flags are wired through to
	// config.Load and result in an authenticated client.
	// flag > env ordering is proven at the config.Load layer in
	// internal/config/config_test.go::TestFlagOverridesEnv; here we only
	// verify that GetClient reads the flags from the cobra command tree at all.
	t.Setenv("HOME", t.TempDir()) // no config file
	t.Setenv("GATE_API_KEY", "")
	t.Setenv("GATE_API_SECRET", "")

	root := newRootCmd()
	require.NoError(t, root.PersistentFlags().Set("api-key", "flag-key"))
	require.NoError(t, root.PersistentFlags().Set("api-secret", "flag-secret"))
	child := newChildCmd(root)

	c, err := cmdutil.GetClient(child)
	require.NoError(t, err)
	// env and file are both empty; IsAuthenticated=true can only come from the flags.
	assert.True(t, c.IsAuthenticated())
}

func TestGetClient_CredentialsFromEnvWhenNoFile(t *testing.T) {
	// Verify that env-var credentials are picked up when no config file exists.
	// (env > file ordering is tested at the config.Load layer in internal/config/config_test.go;
	// here we only verify the cobra→config.Load plumbing for the env path.)
	t.Setenv("HOME", t.TempDir()) // ensure no real ~/.gate-cli/config.yaml interferes
	t.Setenv("GATE_API_KEY", "env-key")
	t.Setenv("GATE_API_SECRET", "env-secret")

	root := newRootCmd()
	child := newChildCmd(root)

	c, err := cmdutil.GetClient(child)
	require.NoError(t, err)
	assert.True(t, c.IsAuthenticated())
}

func TestGetClient_NoCredentials(t *testing.T) {
	t.Setenv("GATE_API_KEY", "")
	t.Setenv("GATE_API_SECRET", "")

	root := newRootCmd()
	child := newChildCmd(root)

	c, err := cmdutil.GetClient(child)
	require.NoError(t, err)
	assert.False(t, c.IsAuthenticated())
}

func TestGetClient_FlagKeyMakesClientAuthenticated(t *testing.T) {
	t.Setenv("GATE_API_KEY", "")
	t.Setenv("GATE_API_SECRET", "")

	root := newRootCmd()
	require.NoError(t, root.PersistentFlags().Set("api-key", "flag-key"))
	require.NoError(t, root.PersistentFlags().Set("api-secret", "flag-secret"))
	child := newChildCmd(root)

	c, err := cmdutil.GetClient(child)
	require.NoError(t, err)
	assert.True(t, c.IsAuthenticated())
}

// --- GetSettle flag priority ---

func newSettleCmd(root *cobra.Command) *cobra.Command {
	child := &cobra.Command{Use: "sub"}
	child.Flags().String("settle", "usdt", "")
	root.AddCommand(child)
	return child
}

func TestGetSettle_FlagExplicitWins(t *testing.T) {
	root := newRootCmd()
	child := newSettleCmd(root)
	require.NoError(t, child.Flags().Set("settle", "btc"))

	assert.Equal(t, "btc", cmdutil.GetSettle(child))
}

func TestGetSettle_ConfigDefaultSettle(t *testing.T) {
	// Write a config file with default_settle = btc; flag is not set.
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgFile, []byte(`
default_settle: btc
profiles:
  default: {}
`), 0600)

	root := newRootCmd()
	child := newSettleCmd(root)

	// Point GetSettle at our temp config by setting the profile flag and
	// temporarily overriding the config path via the Options.ConfigFile path.
	// Since GetSettle calls config.Load with only {Profile: profile}, we need
	// to make the default config path point to our file.
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", dir)
	// Gate config lives at ~/.gate-cli/config.yaml; create the directory.
	gateDir := filepath.Join(dir, ".gate-cli")
	os.MkdirAll(gateDir, 0700)
	os.WriteFile(filepath.Join(gateDir, "config.yaml"), []byte(`
default_settle: btc
profiles:
  default: {}
`), 0600)
	defer t.Setenv("HOME", origHome)

	assert.Equal(t, "btc", cmdutil.GetSettle(child))
}

func TestGetSettle_FallbackUsdt(t *testing.T) {
	// No config file, no flag set → should fall back to "usdt".
	t.Setenv("HOME", t.TempDir()) // ensure no real config file interferes

	root := newRootCmd()
	child := newSettleCmd(root)

	assert.Equal(t, "usdt", cmdutil.GetSettle(child))
}

func TestGetSettle_FlagDefaultNotChanged(t *testing.T) {
	// Flag has its default value "usdt" but was never Set() by the user.
	// GetSettle should NOT treat this as an explicit flag and should fall
	// through to config / built-in fallback.
	t.Setenv("HOME", t.TempDir())

	root := newRootCmd()
	child := newSettleCmd(root) // flag default is "usdt", not Set()

	// With no config, falls through to built-in "usdt".
	assert.Equal(t, "usdt", cmdutil.GetSettle(child))
}
