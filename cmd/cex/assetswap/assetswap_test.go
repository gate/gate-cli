package assetswap

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestRoot provides a minimal cobra root so runXxx handlers can call
// cmdutil.GetClient/GetPrinter during tests. Env is isolated.
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

// authedTestRoot mirrors newTestRoot but seeds fake credentials so handlers
// pass the RequireAuth gate and exercise downstream branches (e.g. JSON parse).
func authedTestRoot(t *testing.T) *cobra.Command {
	root := newTestRoot(t)
	t.Setenv("GATE_API_KEY", "fake-key")
	t.Setenv("GATE_API_SECRET", "fake-secret")
	return root
}

// mockGateServer stands up an httptest server returning fixed JSON for every
// request, points GATE_BASE_URL at it, and registers cleanup. Used for tests
// covering runXxx handlers that skip RequireAuth and go straight to the SDK.
func mockGateServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("GATE_BASE_URL", srv.URL)
	return srv
}

// silenceStdout swaps os.Stdout for /dev/null during the test so printer
// output does not pollute `go test` output. Tests using this must not run in
// parallel (t.Parallel() not called, which is the default).
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

var _ = io.Discard // keep the io import useful for future drain helpers

const cobraBashCompOneRequiredFlag = "cobra_annotation_bash_completion_one_required_flag"

func findSub(parent *cobra.Command, name string) *cobra.Command {
	for _, c := range parent.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

func TestAssetswapRootCommand(t *testing.T) {
	assert.Equal(t, "assetswap", Cmd.Name(), "module root command should be named assetswap")
	assert.NotEmpty(t, Cmd.Short, "Cmd.Short should not be empty")
}

func TestAssetswapFirstLevelSubcommands(t *testing.T) {
	want := map[string]bool{
		"assets":   false,
		"config":   false,
		"evaluate": false,
		"order":    false,
	}
	for _, c := range Cmd.Commands() {
		if _, ok := want[c.Name()]; ok {
			want[c.Name()] = true
		}
	}
	for name, found := range want {
		assert.True(t, found, "assetswap should expose first-level %q", name)
	}
}

func TestAssetswapOrderSecondLevelSubcommands(t *testing.T) {
	orderCmd := findSub(Cmd, "order")
	require.NotNil(t, orderCmd, "assetswap.order should be registered")

	want := map[string]bool{
		"create":  false,
		"preview": false,
		"list":    false,
		"get":     false,
	}
	for _, c := range orderCmd.Commands() {
		if _, ok := want[c.Name()]; ok {
			want[c.Name()] = true
		}
	}
	for name, found := range want {
		assert.True(t, found, "assetswap order should expose %q", name)
	}
}

func TestAssetswapOrderCreateRequiresJSON(t *testing.T) {
	orderCmd := findSub(Cmd, "order")
	require.NotNil(t, orderCmd)
	create := findSub(orderCmd, "create")
	require.NotNil(t, create)

	j := create.Flag("json")
	require.NotNil(t, j, "order create should have --json flag")
	assert.NotEmpty(t, j.Annotations[cobraBashCompOneRequiredFlag],
		"order create --json should be required")
}

func TestAssetswapOrderPreviewRequiresJSON(t *testing.T) {
	orderCmd := findSub(Cmd, "order")
	require.NotNil(t, orderCmd)
	preview := findSub(orderCmd, "preview")
	require.NotNil(t, preview)

	j := preview.Flag("json")
	require.NotNil(t, j, "order preview should have --json flag")
	assert.NotEmpty(t, j.Annotations[cobraBashCompOneRequiredFlag],
		"order preview --json should be required")
}

func TestAssetswapOrderGetTakesPositionalArg(t *testing.T) {
	orderCmd := findSub(Cmd, "order")
	require.NotNil(t, orderCmd)
	get := findSub(orderCmd, "get")
	require.NotNil(t, get)

	err := get.Args(get, []string{})
	assert.Error(t, err, "order get should require 1 arg")
	err = get.Args(get, []string{"id1", "id2"})
	assert.Error(t, err, "order get should reject 2 args")
	err = get.Args(get, []string{"42"})
	assert.NoError(t, err, "order get should accept 1 arg")
}

func TestAssetswapEvaluateOptionalFlags(t *testing.T) {
	ev := findSub(Cmd, "evaluate")
	require.NotNil(t, ev)

	for _, name := range []string{"max-value", "cursor", "size"} {
		f := ev.Flag(name)
		require.NotNil(t, f, "evaluate should expose --%s", name)
		assert.Empty(t, f.Annotations[cobraBashCompOneRequiredFlag],
			"evaluate --%s must be optional", name)
	}
}

// --- RunE error-path coverage ---

func TestRunEvaluate_RequiresAuth(t *testing.T) {
	root := newTestRoot(t)
	cmd := &cobra.Command{Use: "evaluate"}
	cmd.Flags().Int32("max-value", 0, "")
	cmd.Flags().String("cursor", "", "")
	cmd.Flags().Int32("size", 0, "")
	root.AddCommand(cmd)

	err := runEvaluate(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

func TestRunOrderCreate_RequiresAuth(t *testing.T) {
	root := newTestRoot(t)
	cmd := &cobra.Command{Use: "create"}
	cmd.Flags().String("json", `{"from":[{"asset":"BTC","amount":"0.1"}],"to":[{"asset":"USDT","amount":"10000"}]}`, "")
	root.AddCommand(cmd)

	err := runOrderCreate(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

// With credentials seeded, an invalid --json body is reported by
// json.Unmarshal — covers the dedicated error-wrap branch.
func TestRunOrderCreate_InvalidJSON(t *testing.T) {
	root := authedTestRoot(t)
	cmd := &cobra.Command{Use: "create"}
	cmd.Flags().String("json", "not-a-valid-json", "")
	root.AddCommand(cmd)

	err := runOrderCreate(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --json body",
		"error should be wrapped with invalid --json body prefix")
}

func TestRunOrderPreview_RequiresAuth(t *testing.T) {
	root := newTestRoot(t)
	cmd := &cobra.Command{Use: "preview"}
	cmd.Flags().String("json", `{"from":[],"to":[]}`, "")
	root.AddCommand(cmd)

	err := runOrderPreview(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

func TestRunOrderPreview_InvalidJSON(t *testing.T) {
	root := authedTestRoot(t)
	cmd := &cobra.Command{Use: "preview"}
	cmd.Flags().String("json", "][not-json[", "")
	root.AddCommand(cmd)

	err := runOrderPreview(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --json body")
}

func TestRunOrderList_RequiresAuth(t *testing.T) {
	root := newTestRoot(t)
	cmd := &cobra.Command{Use: "list"}
	for _, name := range []string{"from", "to", "status", "offset", "size", "sort-mode", "order-by"} {
		cmd.Flags().Int32(name, 0, "")
	}
	root.AddCommand(cmd)

	err := runOrderList(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

func TestRunOrderGet_RequiresAuth(t *testing.T) {
	root := newTestRoot(t)
	cmd := &cobra.Command{Use: "get"}
	root.AddCommand(cmd)

	err := runOrderGet(cmd, []string{"order-123"})
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

// --- Public (no-auth) RunE coverage via httptest ---
//
// runAssets / runConfig do not call RequireAuth, so covering them requires a
// mock HTTP server. We return an empty JSON object which the SDK happily
// unmarshals into a zero-valued response struct.

func TestRunAssets_Succeeds(t *testing.T) {
	mockGateServer(t, `{}`)
	silenceStdout(t)
	root := newTestRoot(t)
	cmd := &cobra.Command{Use: "assets"}
	root.AddCommand(cmd)

	err := runAssets(cmd, nil)
	assert.NoError(t, err, "runAssets should succeed against mock server")
}

func TestRunConfig_Succeeds(t *testing.T) {
	mockGateServer(t, `{}`)
	silenceStdout(t)
	root := newTestRoot(t)
	cmd := &cobra.Command{Use: "config"}
	root.AddCommand(cmd)

	err := runConfig(cmd, nil)
	assert.NoError(t, err, "runConfig should succeed against mock server")
}

// Server returning an HTTP error path exercises the PrintError branch.
// runAssets returns nil (error swallowed by PrintError) but still covers lines.
func TestRunAssets_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"label":"INTERNAL","message":"boom"}`))
	}))
	defer srv.Close()
	t.Setenv("GATE_BASE_URL", srv.URL)
	// printer writes error JSON to stderr; silence stderr for cleanliness.
	oldErr := os.Stderr
	devNull, _ := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	os.Stderr = devNull
	t.Cleanup(func() { os.Stderr = oldErr; _ = devNull.Close() })

	root := newTestRoot(t)
	cmd := &cobra.Command{Use: "assets"}
	root.AddCommand(cmd)

	err := runAssets(cmd, nil)
	assert.NoError(t, err, "runAssets swallows SDK errors via PrintError and returns nil")
}

func TestAssetswapOrderListOptionalFlags(t *testing.T) {
	orderCmd := findSub(Cmd, "order")
	require.NotNil(t, orderCmd)
	listCmd := findSub(orderCmd, "list")
	require.NotNil(t, listCmd)

	for _, name := range []string{"from", "to", "status", "offset", "size", "sort-mode", "order-by"} {
		f := listCmd.Flag(name)
		require.NotNil(t, f, "order list should expose --%s", name)
		assert.Empty(t, f.Annotations[cobraBashCompOneRequiredFlag],
			"order list --%s must be optional", name)
	}
}
