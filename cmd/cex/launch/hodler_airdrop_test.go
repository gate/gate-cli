package launch

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Public (no-auth) RunE coverage via httptest ---

// runHodlerProjects is public (no auth required); cover both "all-flags-set"
// and "all-flags-zero" branch matrices of the opts builder.
func TestRunHodlerProjects_MockServer_AllFlags(t *testing.T) {
	mockGateServer(t, `[]`)
	silenceStdout(t)
	root := newTestRoot(t)
	cmd := &cobra.Command{Use: "projects"}
	cmd.Flags().String("status", "active", "")
	cmd.Flags().String("keyword", "BTC", "")
	cmd.Flags().Int32("join", 1, "")
	cmd.Flags().Int32("page", 1, "")
	cmd.Flags().Int32("size", 10, "")
	root.AddCommand(cmd)

	err := runHodlerProjects(cmd, nil)
	assert.NoError(t, err)
}

func TestRunHodlerProjects_MockServer_NoFlags(t *testing.T) {
	mockGateServer(t, `[]`)
	silenceStdout(t)
	root := newTestRoot(t)
	cmd := &cobra.Command{Use: "projects"}
	cmd.Flags().String("status", "", "")
	cmd.Flags().String("keyword", "", "")
	cmd.Flags().Int32("join", 0, "")
	cmd.Flags().Int32("page", 0, "")
	cmd.Flags().Int32("size", 0, "")
	root.AddCommand(cmd)

	err := runHodlerProjects(cmd, nil)
	assert.NoError(t, err)
}

// --- RunE error-path coverage (auth-required subset) ---

func TestRunHodlerOrder_RequiresAuth(t *testing.T) {
	root := newTestRoot(t)
	cmd := &cobra.Command{Use: "order"}
	cmd.Flags().Int32("hodler-id", 42, "")
	root.AddCommand(cmd)

	err := runHodlerOrder(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

func TestRunHodlerOrderRecords_RequiresAuth(t *testing.T) {
	root := newTestRoot(t)
	cmd := &cobra.Command{Use: "order-records"}
	cmd.Flags().String("keyword", "", "")
	cmd.Flags().Int32("start-timest", 0, "")
	cmd.Flags().Int32("end-timest", 0, "")
	cmd.Flags().Int32("page", 0, "")
	cmd.Flags().Int32("size", 0, "")
	root.AddCommand(cmd)

	err := runHodlerOrderRecords(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

func TestRunHodlerAirdropRecords_RequiresAuth(t *testing.T) {
	root := newTestRoot(t)
	cmd := &cobra.Command{Use: "airdrop-records"}
	cmd.Flags().String("keyword", "", "")
	cmd.Flags().Int32("start-timest", 0, "")
	cmd.Flags().Int32("end-timest", 0, "")
	cmd.Flags().Int32("page", 0, "")
	cmd.Flags().Int32("size", 0, "")
	root.AddCommand(cmd)

	err := runHodlerAirdropRecords(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

func TestHodlerSubtreeRegistered(t *testing.T) {
	hodler := findLaunchChild("hodler")
	require.NotNil(t, hodler, "launch.Cmd should expose hodler subtree")

	want := map[string]bool{
		"projects":        false,
		"order":           false,
		"order-records":   false,
		"airdrop-records": false,
	}
	for _, sub := range hodler.Commands() {
		if _, ok := want[sub.Name()]; ok {
			want[sub.Name()] = true
		}
	}
	for name, found := range want {
		assert.True(t, found, "hodler should expose %q", name)
	}
}

func TestHodlerOrderRequiresHodlerID(t *testing.T) {
	hodler := findLaunchChild("hodler")
	require.NotNil(t, hodler)

	order := findSubcommand(hodler, "order")
	require.NotNil(t, order)

	hf := order.Flag("hodler-id")
	require.NotNil(t, hf, "hodler order should have --hodler-id flag")
	assert.Equal(t, "int32", hf.Value.Type(), "--hodler-id should be int32")
	assert.NotEmpty(t, hf.Annotations[cobraBashCompOneRequiredFlag],
		"--hodler-id should be required")
}

func TestHodlerProjectsOptionalFlags(t *testing.T) {
	hodler := findLaunchChild("hodler")
	require.NotNil(t, hodler)

	projects := findSubcommand(hodler, "projects")
	require.NotNil(t, projects)

	for _, name := range []string{"status", "keyword", "join", "page", "size"} {
		f := projects.Flag(name)
		require.NotNil(t, f, "hodler projects should expose --%s", name)
		assert.Empty(t, f.Annotations[cobraBashCompOneRequiredFlag],
			"--%s must be optional", name)
	}
}

func TestHodlerRecordsSymmetric(t *testing.T) {
	hodler := findLaunchChild("hodler")
	require.NotNil(t, hodler)

	// order-records and airdrop-records share identical pagination/filtering flags
	// (per SDK Opts: Keyword, StartTimest, EndTimest, Page, Size).
	for _, name := range []string{"order-records", "airdrop-records"} {
		sub := findSubcommand(hodler, name)
		require.NotNil(t, sub, "hodler %s missing", name)
		for _, flagName := range []string{"keyword", "start-timest", "end-timest", "page", "size"} {
			f := sub.Flag(flagName)
			require.NotNil(t, f, "%s should expose --%s", name, flagName)
			assert.Empty(t, f.Annotations[cobraBashCompOneRequiredFlag],
				"%s --%s must be optional", name, flagName)
		}
	}
}
