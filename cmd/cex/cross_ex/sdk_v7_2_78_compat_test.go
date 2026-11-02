package crossex

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gateapi "github.com/gate/gateapi-go/v7"
)

// Tests in this file lock in the gateapi-go SDK upgrade from v7.2.71 to
// v7.2.78 for the cross_ex module. Coverage:
//   1. The new --statement-type flag on `account book`.
//   2. JSON wire-tag for CrossexAccountBookRecord (Type → StatementType).
//   3. Retained RunE handlers reject requests under release retirement.
//   4. RequireAuth gate still trips when no credentials are configured.

const cobraRequiredFlagAnnotation = "cobra_annotation_bash_completion_one_required_flag"

func newCrossexTestRoot(t *testing.T) *cobra.Command {
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

func authedCrossexTestRoot(t *testing.T) *cobra.Command {
	root := newCrossexTestRoot(t)
	t.Setenv("GATE_API_KEY", "fake-key")
	t.Setenv("GATE_API_SECRET", "fake-secret")
	return root
}

func silenceCrossexStdout(t *testing.T) {
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

func findCrossexSub(parent *cobra.Command, name string) *cobra.Command {
	for _, c := range parent.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Layer 1 — Flag wiring on `account book`.
// v7.2.78 added an optional StatementType filter. The CLI exposes it as
// --statement-type. Below pins both the existence and the optional-ness.
// ---------------------------------------------------------------------------

func TestAccountBook_HasStatementTypeFlag_AfterV7_2_78(t *testing.T) {
	book := findCrossexSub(accountCmd, "book")
	require.NotNil(t, book, "account book command must be registered")

	f := book.Flag("statement-type")
	require.NotNil(t, f,
		"--statement-type was added in v7.2.78 to expose the new ListCrossexAccountBookOpts.StatementType field")
	assert.Empty(t, f.Annotations[cobraRequiredFlagAnnotation],
		"--statement-type must remain optional")
}

func TestAccountBook_AllFlagsOptional(t *testing.T) {
	book := findCrossexSub(accountCmd, "book")
	require.NotNil(t, book)

	for _, name := range []string{"page", "limit", "coin", "statement-type", "from", "to"} {
		f := book.Flag(name)
		require.NotNil(t, f, "account book must expose --%s", name)
		assert.Empty(t, f.Annotations[cobraRequiredFlagAnnotation],
			"account book --%s must be optional", name)
	}
}

// ---------------------------------------------------------------------------
// Layer 2 — SDK model field rename (CrossexAccountBookRecord.Type →
// .StatementType). Lock JSON wire shape to catch silent regressions.
// ---------------------------------------------------------------------------

func TestCrossexAccountBookRecord_StatementTypeJSONTag(t *testing.T) {
	payload := `{
		"id": "rec-1",
		"user_id": "uid-1",
		"business_id": "biz-1",
		"statement_type": "TRADING_FEE",
		"exchange_type": "BINANCE",
		"coin": "USDT",
		"change": "1.23",
		"balance": "100.00",
		"create_time": "2026-01-01T00:00:00Z"
	}`
	var rec gateapi.CrossexAccountBookRecord
	require.NoError(t, json.Unmarshal([]byte(payload), &rec))

	assert.Equal(t, "TRADING_FEE", rec.StatementType,
		"v7.2.78 renamed Type → StatementType (json: statement_type)")
	assert.Equal(t, "rec-1", rec.Id)
	assert.Equal(t, "BINANCE", rec.ExchangeType)
}

// The legacy `type` JSON key must no longer bind to any field on the
// struct, otherwise the rename was incomplete.
func TestCrossexAccountBookRecord_LegacyTypeKeyDoesNotBind(t *testing.T) {
	payload := `{"id": "rec-2", "type": "LEGACY_VALUE"}`
	var rec gateapi.CrossexAccountBookRecord
	require.NoError(t, json.Unmarshal([]byte(payload), &rec))

	assert.Equal(t, "rec-2", rec.Id)
	assert.Empty(t, rec.StatementType,
		"legacy `type` key must not populate StatementType under v7.2.78")
}

// ListCrossexAccountBookOpts must accept StatementType and serialize it as
// the `statement_type` query-param. We can only directly inspect the struct,
// not the URL it produces. CLI execution below independently checks retirement.
func TestListCrossexAccountBookOpts_StatementTypeField(t *testing.T) {
	opts := gateapi.ListCrossexAccountBookOpts{
		StatementType: optional.NewString("TRANSACTION"),
	}
	require.True(t, opts.StatementType.IsSet(),
		"StatementType must be a settable optional.String on v7.2.78")
	assert.Equal(t, "TRANSACTION", opts.StatementType.Value())
}

// ---------------------------------------------------------------------------
// Layer 3 — Retained RunE handlers are blocked with full, empty and partial flags.
// ---------------------------------------------------------------------------

// TestRunAccountBook_AllFlags_RetiredBeforeSend verifies that the retained handler rejects
// valid flags and synthetic credentials before any API request.
func TestRunAccountBook_AllFlags_RetiredBeforeSend(t *testing.T) {
	var capturedURL *url.URL
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("retired CLI handler reached the API")
		capturedURL = r.URL
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"id":"rec-1","statement_type":"TRADING_FEE","coin":"USDT","exchange_type":"BINANCE","change":"1","balance":"100","create_time":"now"}]`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("GATE_BASE_URL", srv.URL)

	silenceCrossexStdout(t)
	root := authedCrossexTestRoot(t)
	cmd := &cobra.Command{Use: "book"}
	cmd.Flags().Int32("page", 2, "")
	cmd.Flags().Int32("limit", 50, "")
	cmd.Flags().String("coin", "USDT", "")
	cmd.Flags().String("statement-type", "TRADING_FEE", "")
	cmd.Flags().Int32("from", 1700000000, "")
	cmd.Flags().Int32("to", 1700001000, "")
	root.AddCommand(cmd)

	err := runAccountBook(cmd, nil)
	require.ErrorContains(t, err, "operation unavailable in this CLI release")
	assert.Nil(t, capturedURL, "retired endpoint must not receive a request")
}

// TestRunAccountBook_NoFlags_RetiredBeforeSend verifies that the retained handler rejects
// valid flags and synthetic credentials before any API request.
func TestRunAccountBook_NoFlags_RetiredBeforeSend(t *testing.T) {
	var capturedURL *url.URL
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("retired CLI handler reached the API")
		capturedURL = r.URL
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("GATE_BASE_URL", srv.URL)

	silenceCrossexStdout(t)
	root := authedCrossexTestRoot(t)
	cmd := &cobra.Command{Use: "book"}
	cmd.Flags().Int32("page", 0, "")
	cmd.Flags().Int32("limit", 0, "")
	cmd.Flags().String("coin", "", "")
	cmd.Flags().String("statement-type", "", "")
	cmd.Flags().Int32("from", 0, "")
	cmd.Flags().Int32("to", 0, "")
	root.AddCommand(cmd)

	err := runAccountBook(cmd, nil)
	require.ErrorContains(t, err, "operation unavailable in this CLI release")
	assert.Nil(t, capturedURL, "retired endpoint must not receive a request")
}

// TestRunAccountBook_StatementTypeOnly_RetiredBeforeSend verifies that the retained handler rejects
// valid flags and synthetic credentials before any API request.
func TestRunAccountBook_StatementTypeOnly_RetiredBeforeSend(t *testing.T) {
	var capturedURL *url.URL
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("retired CLI handler reached the API")
		capturedURL = r.URL
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("GATE_BASE_URL", srv.URL)

	silenceCrossexStdout(t)
	root := authedCrossexTestRoot(t)
	cmd := &cobra.Command{Use: "book"}
	cmd.Flags().Int32("page", 0, "")
	cmd.Flags().Int32("limit", 0, "")
	cmd.Flags().String("coin", "", "")
	cmd.Flags().String("statement-type", "FUNDING_FEE", "")
	cmd.Flags().Int32("from", 0, "")
	cmd.Flags().Int32("to", 0, "")
	root.AddCommand(cmd)

	err := runAccountBook(cmd, nil)
	require.ErrorContains(t, err, "operation unavailable in this CLI release")
	assert.Nil(t, capturedURL, "retired endpoint must not receive a request")
}

// TestRunAccountBook_TableRendering_RetiredBeforeSend verifies that the retained handler rejects
// valid flags and synthetic credentials before any API request.
func TestRunAccountBook_TableRendering_RetiredBeforeSend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("retired CLI handler reached the API")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"id":"r1","statement_type":"AUTO_REPAY","coin":"USDT","exchange_type":"GATE","change":"-0.01","balance":"42","create_time":"now"}]`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("GATE_BASE_URL", srv.URL)

	root := authedCrossexTestRoot(t)
	// Output format must not affect the release-level execution denial.
	require.NoError(t, root.PersistentFlags().Set("format", "text"))

	cmd := &cobra.Command{Use: "book"}
	cmd.Flags().Int32("page", 0, "")
	cmd.Flags().Int32("limit", 0, "")
	cmd.Flags().String("coin", "", "")
	cmd.Flags().String("statement-type", "", "")
	cmd.Flags().Int32("from", 0, "")
	cmd.Flags().Int32("to", 0, "")
	root.AddCommand(cmd)

	err := runAccountBook(cmd, nil)
	require.ErrorContains(t, err, "operation unavailable in this CLI release")
}

// ---------------------------------------------------------------------------
// Layer 4 — Auth gate. Account book reads private data; RequireAuth must
// continue to fence unauthenticated callers.
// ---------------------------------------------------------------------------

func TestRunAccountBook_RequiresAuth(t *testing.T) {
	root := newCrossexTestRoot(t)
	cmd := &cobra.Command{Use: "book"}
	cmd.Flags().Int32("page", 0, "")
	cmd.Flags().Int32("limit", 0, "")
	cmd.Flags().String("coin", "", "")
	cmd.Flags().String("statement-type", "", "")
	cmd.Flags().Int32("from", 0, "")
	cmd.Flags().Int32("to", 0, "")
	root.AddCommand(cmd)

	err := runAccountBook(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}
