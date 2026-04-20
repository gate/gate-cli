package configcmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/gate/gate-cli/internal/config"
)

// writeConfig writes a YAML config file under dir/.gate-cli/config.yaml and
// returns the config dir path.
func writeConfig(t *testing.T, fc fileLayout) string {
	t.Helper()
	home := t.TempDir()
	cfgDir := filepath.Join(home, ".gate-cli")
	require.NoError(t, os.MkdirAll(cfgDir, 0700))
	data, err := yaml.Marshal(fc)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(cfgDir, "config.yaml"), data, 0600))
	t.Setenv("HOME", home)
	return home
}

// readConfig reads the YAML config file from the test home dir.
func readConfig(t *testing.T, home string) fileLayout {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(home, ".gate-cli", "config.yaml"))
	require.NoError(t, err)
	var fc fileLayout
	require.NoError(t, yaml.Unmarshal(data, &fc))
	return fc
}

// runSetCmd executes the setCmd cobra command with the given args.
func runSetCmd(t *testing.T, args ...string) error {
	t.Helper()
	setCmd.ResetFlags()
	setCmd.Flags().String("profile", "default", "Profile to update")
	return setCmd.RunE(setCmd, args)
}

// --- runSet / default_profile resolution ---

// TestRunSet_UsesDefaultProfileFromConfig verifies that "config set" without
// --profile writes to the profile named in default_profile, not hardcoded "default".
func TestRunSet_UsesDefaultProfileFromConfig(t *testing.T) {
	home := writeConfig(t, fileLayout{
		DefaultProfile: "myprofile",
		DefaultSettle:  "usdt",
		Profiles: map[string]profileEntry{
			"myprofile": {APIKey: "oldkey", APISecret: "oldsecret"},
		},
	})

	err := runSetCmd(t, "api-key", "newkey")
	require.NoError(t, err)

	fc := readConfig(t, home)
	assert.Equal(t, "newkey", fc.Profiles["myprofile"].APIKey, "should update the default_profile, not 'default'")
	assert.Empty(t, fc.Profiles["default"], "should NOT create a 'default' profile")
}

// TestRunSet_ExplicitProfileFlagTakesPrecedence verifies that an explicit
// --profile flag always wins over default_profile.
func TestRunSet_ExplicitProfileFlagTakesPrecedence(t *testing.T) {
	home := writeConfig(t, fileLayout{
		DefaultProfile: "myprofile",
		DefaultSettle:  "usdt",
		Profiles: map[string]profileEntry{
			"myprofile": {APIKey: "a", APISecret: "b"},
			"other":     {APIKey: "x", APISecret: "y"},
		},
	})

	setCmd.ResetFlags()
	setCmd.Flags().String("profile", "default", "Profile to update")
	require.NoError(t, setCmd.Flags().Set("profile", "other"))
	err := setCmd.RunE(setCmd, []string{"api-key", "updated"})
	require.NoError(t, err)

	fc := readConfig(t, home)
	assert.Equal(t, "updated", fc.Profiles["other"].APIKey)
	assert.Equal(t, "a", fc.Profiles["myprofile"].APIKey, "myprofile should be untouched")
}

// TestRunSet_DefaultFallsBackToDefaultWhenNoDefaultProfile verifies that when
// default_profile is empty, --profile defaults to "default" as before.
func TestRunSet_DefaultFallsBackToDefaultWhenNoDefaultProfile(t *testing.T) {
	home := writeConfig(t, fileLayout{
		DefaultSettle: "usdt",
		Profiles: map[string]profileEntry{
			"default": {APIKey: "origkey", APISecret: "origsecret"},
		},
	})

	err := runSetCmd(t, "api-key", "changedkey")
	require.NoError(t, err)

	fc := readConfig(t, home)
	assert.Equal(t, "changedkey", fc.Profiles["default"].APIKey)
}

func TestRunInit_WritesDefaultIntelURLs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	input, err := os.CreateTemp(t.TempDir(), "config-init-input-*.txt")
	require.NoError(t, err)
	_, err = input.WriteString("\n\n\n")
	require.NoError(t, err)
	_, err = input.Seek(0, 0)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = input.Close()
	})

	oldStdin := os.Stdin
	os.Stdin = input
	t.Cleanup(func() {
		os.Stdin = oldStdin
	})

	require.NoError(t, runInit(initCmd, nil))

	fc := readConfig(t, home)
	assert.Equal(t, config.DefaultIntelInfoMCPURL, fc.Intel.InfoMCPURL)
	assert.Equal(t, config.DefaultIntelNewsMCPURL, fc.Intel.NewsMCPURL)
}

// --- maskSecrets (P2-1) ---

func TestMaskSecrets_BasicYAML(t *testing.T) {
	input := `profiles:
  default:
    api_key: mykey
    api_secret: mysecret`
	out := maskSecrets(input)
	assert.Contains(t, out, "api_secret: ****")
	assert.NotContains(t, out, "mysecret")
	assert.Contains(t, out, "api_key: ****")
	assert.NotContains(t, out, "mykey")
}

func TestMaskSecrets_PreservesIndentation(t *testing.T) {
	input := "    api_secret: supersecret"
	out := maskSecrets(input)
	assert.Equal(t, "    api_secret: ****", out)
}

func TestMaskSecrets_NoCredentialFields(t *testing.T) {
	input := "base_url: https://example.com\ndefault_settle: usdt"
	out := maskSecrets(input)
	assert.Equal(t, input, out)
}

func TestMaskSecrets_MultipleProfiles(t *testing.T) {
	input := `profiles:
  prod:
    api_key: prodkey
    api_secret: prodsecret
  test:
    api_key: testkey
    api_secret: testsecret`
	out := maskSecrets(input)
	assert.NotContains(t, out, "prodsecret")
	assert.NotContains(t, out, "testsecret")
	assert.NotContains(t, out, "prodkey")
	assert.NotContains(t, out, "testkey")
}

func TestMaskSecrets_APIKeyMasked(t *testing.T) {
	input := "    api_key: abc123"
	out := maskSecrets(input)
	assert.Equal(t, "    api_key: ****", out)
	assert.NotContains(t, out, "abc123")
}

func TestMaskSecrets_EmptySecret(t *testing.T) {
	// Empty value should still be masked (no accidental leaks of empty-string marker).
	input := "    api_secret: \"\""
	out := maskSecrets(input)
	assert.Equal(t, "    api_secret: ****", out)
}
