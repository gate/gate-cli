package earn

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

// mockGateServer spins up an httptest server returning fixed JSON for every
// request, redirects GATE_BASE_URL at it, and registers cleanup.
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

// silenceStdout swaps os.Stdout for /dev/null to keep `go test` output clean.
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

// newTestRoot builds a minimal root with persistent flags matching cmd/root.go
// so runXxx handlers can call cmdutil.GetClient/GetPrinter without panicking.
// env is isolated so no real API key leaks in.
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

func TestDualSubcommands(t *testing.T) {
	want := map[string]bool{
		"plans":           false,
		"orders":          false,
		"place":           false,
		"balance":         false,
		"refund-preview":  false,
		"refund":          false,
		"modify-reinvest": false,
		"recommend":       false,
	}

	dualFound := false
	for _, c := range Cmd.Commands() {
		if c.Name() == "dual" {
			dualFound = true
			for _, sub := range c.Commands() {
				if _, ok := want[sub.Name()]; ok {
					want[sub.Name()] = true
				}
			}
			break
		}
	}
	require.True(t, dualFound, "dual command should be registered on earn.Cmd")

	for name, found := range want {
		assert.True(t, found, "dual should expose subcommand %q", name)
	}
}

func TestDualRefundPreviewTakesPositionalArg(t *testing.T) {
	for _, c := range Cmd.Commands() {
		if c.Name() != "dual" {
			continue
		}
		for _, sub := range c.Commands() {
			if sub.Name() != "refund-preview" {
				continue
			}
			// Args must be set to ExactArgs(1); invoke it with wrong arity to prove.
			err := sub.Args(sub, []string{})
			assert.Error(t, err, "refund-preview requires exactly 1 arg")
			err = sub.Args(sub, []string{"id1", "id2"})
			assert.Error(t, err, "refund-preview should reject 2 args")
			err = sub.Args(sub, []string{"42"})
			assert.NoError(t, err, "refund-preview should accept 1 arg")
			return
		}
	}
	t.Fatal("dual refund-preview subcommand not found")
}

func TestDualRefundRequiredFlags(t *testing.T) {
	for _, c := range Cmd.Commands() {
		if c.Name() != "dual" {
			continue
		}
		for _, sub := range c.Commands() {
			if sub.Name() != "refund" {
				continue
			}
			for _, name := range []string{"order-id", "req-id"} {
				f := sub.Flag(name)
				require.NotNil(t, f, "refund should have --%s flag", name)
				ann := f.Annotations[cobraBashCompOneRequiredFlag]
				assert.NotEmpty(t, ann, "--%s should be marked required", name)
			}
			return
		}
	}
	t.Fatal("dual refund subcommand not found")
}

func TestDualModifyReinvestRequiredFlags(t *testing.T) {
	for _, c := range Cmd.Commands() {
		if c.Name() != "dual" {
			continue
		}
		for _, sub := range c.Commands() {
			if sub.Name() != "modify-reinvest" {
				continue
			}
			for _, name := range []string{"order-id", "status"} {
				f := sub.Flag(name)
				require.NotNil(t, f, "modify-reinvest should have --%s flag", name)
				ann := f.Annotations[cobraBashCompOneRequiredFlag]
				assert.NotEmpty(t, ann, "--%s should be marked required", name)
			}
			// duration is optional
			f := sub.Flag("duration")
			require.NotNil(t, f, "modify-reinvest should have --duration flag")
			assert.Empty(t, f.Annotations[cobraBashCompOneRequiredFlag], "--duration must not be required")
			return
		}
	}
	t.Fatal("dual modify-reinvest subcommand not found")
}

func TestDualRecommendOptionalFlags(t *testing.T) {
	for _, c := range Cmd.Commands() {
		if c.Name() != "dual" {
			continue
		}
		for _, sub := range c.Commands() {
			if sub.Name() != "recommend" {
				continue
			}
			for _, name := range []string{"mode", "coin", "type", "history-pids"} {
				f := sub.Flag(name)
				require.NotNil(t, f, "recommend should have --%s flag", name)
				assert.Empty(t, f.Annotations[cobraBashCompOneRequiredFlag], "--%s must not be required", name)
			}
			return
		}
	}
	t.Fatal("dual recommend subcommand not found")
}

// cobra uses this annotation key internally to mark MarkFlagRequired; we probe
// via the Annotations map so we don't have to import cobra's internals.
const cobraBashCompOneRequiredFlag = "cobra_annotation_bash_completion_one_required_flag"

// runDualRecommend does not call RequireAuth, so covering it needs a mock
// server. The optional flags feed four `if != ""` branches that we light up
// by passing non-empty values.
func TestRunDualRecommend_MockServer_AllFlags(t *testing.T) {
	mockGateServer(t, `[]`)
	silenceStdout(t)
	root := newTestRoot(t)
	cmd := &cobra.Command{Use: "recommend"}
	cmd.Flags().String("mode", "normal", "")
	cmd.Flags().String("coin", "BTC", "")
	cmd.Flags().String("type", "call", "")
	cmd.Flags().String("history-pids", "1,2,3", "")
	root.AddCommand(cmd)

	err := runDualRecommend(cmd, nil)
	assert.NoError(t, err, "runDualRecommend should succeed against mock server")
}

// Same handler but all optional flags empty: exercises the non-set branches
// of the opts builder.
func TestRunDualRecommend_MockServer_NoFlags(t *testing.T) {
	mockGateServer(t, `[]`)
	silenceStdout(t)
	root := newTestRoot(t)
	cmd := &cobra.Command{Use: "recommend"}
	cmd.Flags().String("mode", "", "")
	cmd.Flags().String("coin", "", "")
	cmd.Flags().String("type", "", "")
	cmd.Flags().String("history-pids", "", "")
	root.AddCommand(cmd)

	err := runDualRecommend(cmd, nil)
	assert.NoError(t, err)
}

// --- RunE error-path coverage ---
//
// The four dual runXxx helpers all require auth before touching the SDK. With
// no credentials configured, they return an error that mentions "API key".
// Driving the function directly keeps coverage counters accurate without
// starting an HTTP server.

func TestRunDualRefundPreview_RequiresAuth(t *testing.T) {
	root := newTestRoot(t)
	cmd := &cobra.Command{Use: "refund-preview"}
	root.AddCommand(cmd)

	err := runDualRefundPreview(cmd, []string{"order-42"})
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

func TestRunDualRefund_RequiresAuth(t *testing.T) {
	root := newTestRoot(t)
	cmd := &cobra.Command{Use: "refund"}
	cmd.Flags().String("order-id", "42", "")
	cmd.Flags().String("req-id", "req-abc", "")
	root.AddCommand(cmd)

	err := runDualRefund(cmd, []string{})
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

func TestRunDualModifyReinvest_RequiresAuth(t *testing.T) {
	root := newTestRoot(t)
	cmd := &cobra.Command{Use: "modify-reinvest"}
	cmd.Flags().Int64("order-id", 42, "")
	cmd.Flags().Int32("status", 1, "")
	cmd.Flags().Int64("duration", 86400, "")
	root.AddCommand(cmd)

	err := runDualModifyReinvest(cmd, []string{})
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}
