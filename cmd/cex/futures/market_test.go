package futures

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockGateServer returns fixed JSON for every request and points GATE_BASE_URL
// at itself. Used for covering runXxx handlers that bypass RequireAuth.
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

// cobra uses this annotation key internally to mark MarkFlagRequired; tests
// probe via the Annotations map so we don't depend on cobra's internals.
const cobraBashCompOneRequiredFlag = "cobra_annotation_bash_completion_one_required_flag"

func TestFuturesMarketRiskLimitTableRegistered(t *testing.T) {
	found := false
	for _, c := range Cmd.Commands() {
		if c.Name() != "market" {
			continue
		}
		for _, sub := range c.Commands() {
			if sub.Name() == "risk-limit-table" {
				found = true
				// Verify it takes exactly one positional arg (the table-id).
				err := sub.Args(sub, []string{})
				assert.Error(t, err, "risk-limit-table should require 1 arg")
				err = sub.Args(sub, []string{"1", "2"})
				assert.Error(t, err, "risk-limit-table should reject 2 args")
				err = sub.Args(sub, []string{"42"})
				assert.NoError(t, err, "risk-limit-table should accept 1 arg")
				return
			}
		}
	}
	require.True(t, found, "market should expose risk-limit-table subcommand")
}

// runFuturesRiskLimitTable is public (no auth). Mock a minimal JSON response
// so the handler executes end-to-end against the httptest server.
func TestRunFuturesRiskLimitTable_MockServer(t *testing.T) {
	mockGateServer(t, `[]`)
	silenceStdout(t)
	root := newTestRoot(t)
	cmd := &cobra.Command{Use: "risk-limit-table"}
	cmd.Flags().String("settle", "usdt", "")
	root.AddCommand(cmd)

	err := runFuturesRiskLimitTable(cmd, []string{"42"})
	assert.NoError(t, err, "runFuturesRiskLimitTable should succeed against mock server")
}

func TestFuturesMarketExistingSubcommandsStillRegistered(t *testing.T) {
	// Guard against accidental regressions while adding new commands.
	want := map[string]bool{
		"ticker":              false,
		"contracts":           false,
		"risk-limit-tiers":    false,
		"risk-limit-table":    false,
		"candlesticks":        false,
		"orderbook":           false,
		"premium":             false,
		"funding-rate":        false,
		"index-constituents":  false,
		"batch-funding-rates": false,
	}
	for _, c := range Cmd.Commands() {
		if c.Name() != "market" {
			continue
		}
		for _, sub := range c.Commands() {
			if _, ok := want[sub.Name()]; ok {
				want[sub.Name()] = true
			}
		}
	}
	for name, found := range want {
		assert.True(t, found, "market should still expose %q", name)
	}
}
