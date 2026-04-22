package launch

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockGateServer stands up an httptest server returning fixed JSON for every
// request and points GATE_BASE_URL at it.
func mockGateServer(t *testing.T, body string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("GATE_BASE_URL", srv.URL)
}

func silenceStdout(t *testing.T) {
	t.Helper()
	oldOut := os.Stdout
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	require.NoError(t, err)
	os.Stdout = devNull
	t.Cleanup(func() {
		os.Stdout = oldOut
		_ = devNull.Close()
	})
}

// newTestRoot provides a minimal cobra root so runXxx handlers can call
// cmdutil.GetClient/GetPrinter during tests. Env is isolated so no real
// credentials leak in.
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

// findSubcommand walks a cobra parent and returns the named child.
func findSubcommand(parent *cobra.Command, name string) *cobra.Command {
	for _, c := range parent.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

// findLaunchChild returns Cmd.<name> or nil.
func findLaunchChild(name string) *cobra.Command {
	return findSubcommand(Cmd, name)
}

func TestCandyDropSubtreeRegistered(t *testing.T) {
	candy := findLaunchChild("candy-drop")
	require.NotNil(t, candy, "launch.Cmd should expose candy-drop subtree")

	want := map[string]bool{
		"activities":     false,
		"rules":          false,
		"register":       false,
		"progress":       false,
		"participations": false,
		"airdrops":       false,
	}
	for _, sub := range candy.Commands() {
		if _, ok := want[sub.Name()]; ok {
			want[sub.Name()] = true
		}
	}
	for name, found := range want {
		assert.True(t, found, "candy-drop should expose %q", name)
	}
}

func TestCandyDropRegisterRequiresCurrency(t *testing.T) {
	candy := findLaunchChild("candy-drop")
	require.NotNil(t, candy)

	register := findSubcommand(candy, "register")
	require.NotNil(t, register)

	curFlag := register.Flag("currency")
	require.NotNil(t, curFlag, "register should have --currency flag")
	assert.NotEmpty(t, curFlag.Annotations[cobraBashCompOneRequiredFlag],
		"register --currency should be required")

	// activity-id is optional per SDK contract (ActivityId int64 `json:"activity_id,omitempty"`).
	actID := register.Flag("activity-id")
	require.NotNil(t, actID)
	assert.Empty(t, actID.Annotations[cobraBashCompOneRequiredFlag],
		"register --activity-id must be optional")
}

func TestCandyDropActivitiesOptionalFlags(t *testing.T) {
	candy := findLaunchChild("candy-drop")
	require.NotNil(t, candy)

	activities := findSubcommand(candy, "activities")
	require.NotNil(t, activities)

	for _, name := range []string{"status", "rule-name", "register-status", "currency", "limit", "offset"} {
		f := activities.Flag(name)
		require.NotNil(t, f, "activities should expose --%s", name)
		assert.Empty(t, f.Annotations[cobraBashCompOneRequiredFlag],
			"--%s must be optional", name)
	}
}

// --- Public (no-auth) RunE coverage via httptest ---

// runCandyDropActivities skips RequireAuth; exercise the full opts-builder
// branch matrix by passing every optional flag.
func TestRunCandyDropActivities_MockServer_AllFlags(t *testing.T) {
	mockGateServer(t, `[]`)
	silenceStdout(t)
	root := newTestRoot(t)
	cmd := &cobra.Command{Use: "activities"}
	cmd.Flags().String("status", "active", "")
	cmd.Flags().String("rule-name", "alpha", "")
	cmd.Flags().String("register-status", "registered", "")
	cmd.Flags().String("currency", "BTC", "")
	cmd.Flags().Int32("limit", 20, "")
	cmd.Flags().Int32("offset", 5, "")
	root.AddCommand(cmd)

	err := runCandyDropActivities(cmd, nil)
	assert.NoError(t, err)
}

// Same handler with all optional flags at their zero value.
func TestRunCandyDropActivities_MockServer_NoFlags(t *testing.T) {
	mockGateServer(t, `[]`)
	silenceStdout(t)
	root := newTestRoot(t)
	cmd := &cobra.Command{Use: "activities"}
	cmd.Flags().String("status", "", "")
	cmd.Flags().String("rule-name", "", "")
	cmd.Flags().String("register-status", "", "")
	cmd.Flags().String("currency", "", "")
	cmd.Flags().Int32("limit", 0, "")
	cmd.Flags().Int32("offset", 0, "")
	root.AddCommand(cmd)

	err := runCandyDropActivities(cmd, nil)
	assert.NoError(t, err)
}

func TestRunCandyDropRules_MockServer(t *testing.T) {
	mockGateServer(t, `{}`)
	silenceStdout(t)
	root := newTestRoot(t)
	cmd := &cobra.Command{Use: "rules"}
	cmd.Flags().Int64("activity-id", 100, "")
	cmd.Flags().String("currency", "USDT", "")
	root.AddCommand(cmd)

	err := runCandyDropRules(cmd, nil)
	assert.NoError(t, err)
}

// --- RunE error-path coverage (auth-required subset) ---

func TestRunCandyDropRegister_RequiresAuth(t *testing.T) {
	root := newTestRoot(t)
	cmd := &cobra.Command{Use: "register"}
	cmd.Flags().String("currency", "BTC", "")
	cmd.Flags().Int64("activity-id", 12345, "")
	root.AddCommand(cmd)

	err := runCandyDropRegister(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

func TestRunCandyDropProgress_RequiresAuth(t *testing.T) {
	root := newTestRoot(t)
	cmd := &cobra.Command{Use: "progress"}
	cmd.Flags().Int64("activity-id", 0, "")
	cmd.Flags().String("currency", "", "")
	root.AddCommand(cmd)

	err := runCandyDropProgress(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

func TestRunCandyDropParticipations_RequiresAuth(t *testing.T) {
	root := newTestRoot(t)
	cmd := &cobra.Command{Use: "participations"}
	cmd.Flags().String("currency", "", "")
	cmd.Flags().String("status", "", "")
	cmd.Flags().Int64("start-time", 0, "")
	cmd.Flags().Int64("end-time", 0, "")
	cmd.Flags().Int32("page", 0, "")
	cmd.Flags().Int32("limit", 0, "")
	root.AddCommand(cmd)

	err := runCandyDropParticipations(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

func TestRunCandyDropAirdrops_RequiresAuth(t *testing.T) {
	root := newTestRoot(t)
	cmd := &cobra.Command{Use: "airdrops"}
	cmd.Flags().String("currency", "", "")
	cmd.Flags().Int64("start-time", 0, "")
	cmd.Flags().Int64("end-time", 0, "")
	cmd.Flags().Int32("page", 0, "")
	cmd.Flags().Int32("limit", 0, "")
	root.AddCommand(cmd)

	err := runCandyDropAirdrops(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

func TestCandyDropParticipationsAndAirdropsSymmetric(t *testing.T) {
	// Both participation and airdrop records share the same shape of optional filters.
	candy := findLaunchChild("candy-drop")
	require.NotNil(t, candy)

	for _, name := range []string{"participations", "airdrops"} {
		sub := findSubcommand(candy, name)
		require.NotNil(t, sub, "candy-drop %s missing", name)
		for _, flagName := range []string{"currency", "start-time", "end-time", "page", "limit"} {
			f := sub.Flag(flagName)
			require.NotNil(t, f, "%s should expose --%s", name, flagName)
			assert.Empty(t, f.Annotations[cobraBashCompOneRequiredFlag],
				"%s --%s must be optional", name, flagName)
		}
	}
}
