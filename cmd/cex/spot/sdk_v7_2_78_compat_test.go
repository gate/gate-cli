package spot

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gateapi "github.com/gate/gateapi-go/v7"
)

// SDK v7.2.78 marks SpotAccountBook.Type as deprecated and points users to
// the new authoritative `code` field. The CLI now renders both columns; this
// file pins:
//   1. The Code field exists on the SDK model.
//   2. Wire JSON binds `code` to that Go field.
//   3. The `cex spot account book` table includes a Code column.

func newSpotTestRoot(t *testing.T) *cobra.Command {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GATE_API_KEY", "fake-key")
	t.Setenv("GATE_API_SECRET", "fake-secret")
	root := &cobra.Command{Use: "gate-cli"}
	root.PersistentFlags().String("format", "text", "")
	root.PersistentFlags().String("profile", "default", "")
	root.PersistentFlags().Bool("debug", false, "")
	root.PersistentFlags().Bool("verbose", false, "")
	root.PersistentFlags().String("api-key", "", "")
	root.PersistentFlags().String("api-secret", "", "")
	return root
}

// (1) Compile-time + JSON-tag pin: SpotAccountBook must expose a Code
// field bound to the v7.2.78 `code` wire key. This is the field the
// table-rendering path now reads alongside the deprecated Type.
func TestSpotAccountBook_CodeFieldBindsToWireKey(t *testing.T) {
	payload := `{"id":"r1","time":1700000000000,"currency":"USDT","change":"1.23","balance":"100","type":"trade","code":"301"}`
	var rec gateapi.SpotAccountBook
	require.NoError(t, json.Unmarshal([]byte(payload), &rec))

	assert.Equal(t, "301", rec.Code,
		"v7.2.78 added Code as the authoritative account-change identifier; CLI table renders it")
	// Type must still bind so the CLI can keep rendering it for backward
	// visibility (deprecated, not removed).
	assert.Equal(t, "trade", rec.Type)
}

// (2) End-to-end: invoke runSpotAccountBook against a mock server and
// assert the rendered table includes both the legacy Type column and the
// new Code column. Captures stdout so the assertion is on what the user
// actually sees.
func TestRunSpotAccountBook_TableIncludesCodeColumn_v7_2_78(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"id":"r1","time":1700000000000,"currency":"USDT","change":"-0.01","balance":"99.99","type":"fee","code":"401"}]`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("GATE_BASE_URL", srv.URL)

	// Capture stdout. Using io.Copy + a goroutine prevents pipe-buffer
	// stalls when the CLI writes more than the default pipe size.
	r, w, _ := os.Pipe()
	oldOut := os.Stdout
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = oldOut })
	done := make(chan []byte, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.Bytes()
	}()

	root := newSpotTestRoot(t)
	cmd := &cobra.Command{Use: "book"}
	cmd.Flags().String("currency", "", "")
	cmd.Flags().Int64("from", 0, "")
	cmd.Flags().Int64("to", 0, "")
	cmd.Flags().Int32("limit", 0, "")
	root.AddCommand(cmd)

	err := runSpotAccountBook(cmd, nil)
	_ = w.Close()
	output := string(<-done)

	require.NoError(t, err)
	assert.Contains(t, output, "Type",
		"deprecated Type column must remain for backward compatibility")
	assert.Contains(t, output, "Code",
		"v7.2.78 added Code column; CLI must render the new authoritative field")
	assert.Contains(t, output, "401",
		"the row's Code value must be rendered alongside Type")
	assert.Contains(t, output, "fee",
		"the row's Type value must still be rendered for backward visibility")
}
