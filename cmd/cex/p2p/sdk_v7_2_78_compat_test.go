package p2p

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	gateapi "github.com/gate/gateapi-go/v7"
)

// Tests in this file lock in the gateapi-go SDK upgrade from v7.2.71 to
// v7.2.78. They cover three layers:
//   1. Cobra command/flag wiring updated to match the new SDK surface.
//   2. SDK model JSON tags so the wire-level rename TradeId → Txid is
//      observable, not just a Go field rename.
//   3. RunE behavior end-to-end against a mock Gate server, exercising the
//      new return types and removed opts arguments.

const cobraRequiredFlagAnnotation = "cobra_annotation_bash_completion_one_required_flag"

func newP2pTestRoot(t *testing.T) *cobra.Command {
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

func authedP2pTestRoot(t *testing.T) *cobra.Command {
	root := newP2pTestRoot(t)
	t.Setenv("GATE_API_KEY", "fake-key")
	t.Setenv("GATE_API_SECRET", "fake-secret")
	return root
}

func mockP2pGateServer(t *testing.T, body string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("GATE_BASE_URL", srv.URL)
}

func silenceP2pStdout(t *testing.T) {
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

func findP2pSub(parent *cobra.Command, name string) *cobra.Command {
	for _, c := range parent.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Layer 1 — Command / flag wiring after SDK v7.2.78 upgrade.
// ---------------------------------------------------------------------------

// v7.2.78 dropped the optional TradeType param from
// P2pMerchantBooksAdsUpdateStatus, so the CLI must no longer expose it.
// Regression guard against accidentally re-introducing the flag.
func TestAdsUpdateStatus_NoTradeTypeFlag_AfterV7_2_78(t *testing.T) {
	updateStatus := findP2pSub(adsCmd, "update-status")
	require.NotNil(t, updateStatus, "ads update-status command must be registered")

	assert.Nil(t, updateStatus.Flag("trade-type"),
		"--trade-type was removed in v7.2.78 (P2pMerchantBooksAdsUpdateStatusOpts deleted) and must not reappear")
}

func TestAdsUpdateStatus_RequiredFlags(t *testing.T) {
	updateStatus := findP2pSub(adsCmd, "update-status")
	require.NotNil(t, updateStatus)

	for _, name := range []string{"adv-no", "adv-status"} {
		f := updateStatus.Flag(name)
		require.NotNil(t, f, "update-status must expose --%s", name)
		assert.NotEmpty(t, f.Annotations[cobraRequiredFlagAnnotation],
			"update-status --%s must be required", name)
	}
}

func TestTransactionConfirmPayment_RequiredFlags(t *testing.T) {
	confirm := findP2pSub(transactionCmd, "confirm-payment")
	require.NotNil(t, confirm, "confirm-payment must be registered")

	for _, name := range []string{"trade-id", "payment-method"} {
		f := confirm.Flag(name)
		require.NotNil(t, f, "confirm-payment must expose --%s", name)
		assert.NotEmpty(t, f.Annotations[cobraRequiredFlagAnnotation],
			"confirm-payment --%s must be required", name)
	}
}

func TestTransactionConfirmReceipt_RequiredFlags(t *testing.T) {
	confirm := findP2pSub(transactionCmd, "confirm-receipt")
	require.NotNil(t, confirm, "confirm-receipt must be registered")

	f := confirm.Flag("trade-id")
	require.NotNil(t, f)
	assert.NotEmpty(t, f.Annotations[cobraRequiredFlagAnnotation],
		"confirm-receipt --trade-id must be required")
}

func TestTransactionCancel_FlagShape(t *testing.T) {
	cancel := findP2pSub(transactionCmd, "cancel")
	require.NotNil(t, cancel)

	tradeID := cancel.Flag("trade-id")
	require.NotNil(t, tradeID)
	assert.NotEmpty(t, tradeID.Annotations[cobraRequiredFlagAnnotation],
		"cancel --trade-id must be required")

	for _, name := range []string{"reason-id", "reason-memo"} {
		f := cancel.Flag(name)
		require.NotNil(t, f, "cancel must expose --%s", name)
		assert.Empty(t, f.Annotations[cobraRequiredFlagAnnotation],
			"cancel --%s must remain optional", name)
	}
}

// ---------------------------------------------------------------------------
// Layer 2 — SDK model JSON tags. The on-the-wire rename TradeId → Txid is the
// breaking change of this upgrade; freeze it via JSON serialization so future
// regenerations cannot silently undo it.
// ---------------------------------------------------------------------------

func TestConfirmPayment_TxidJSONTag(t *testing.T) {
	body := gateapi.ConfirmPayment{
		Txid:          "12345",
		PaymentMethod: "bank",
	}
	raw, err := json.Marshal(body)
	require.NoError(t, err)

	s := string(raw)
	assert.Contains(t, s, `"txid":"12345"`,
		"ConfirmPayment must serialize order id under wire field `txid` (v7.2.78)")
	assert.NotContains(t, s, `"trade_id"`,
		"v7.2.78 renamed trade_id → txid; the legacy tag must be gone")
	assert.Contains(t, s, `"payment_method":"bank"`)
}

func TestConfirmPayment_PaymentMethodOmitemptyOnEmpty(t *testing.T) {
	// v7.2.78 made payment_method optional (omitempty); empty value should
	// be elided so the server applies its default behavior.
	raw, err := json.Marshal(gateapi.ConfirmPayment{Txid: "12345"})
	require.NoError(t, err)
	assert.NotContains(t, string(raw), `"payment_method"`,
		"empty payment_method should be omitted under v7.2.78 omitempty tag")
}

func TestConfirmReceipt_TxidJSONTag(t *testing.T) {
	raw, err := json.Marshal(gateapi.ConfirmReceipt{Txid: "67890"})
	require.NoError(t, err)

	s := string(raw)
	assert.Equal(t, `{"txid":"67890"}`, s,
		"ConfirmReceipt only carries txid under v7.2.78")
	assert.NotContains(t, s, `"trade_id"`)
}

func TestCancelOrder_TxidJSONTag(t *testing.T) {
	body := gateapi.CancelOrder{
		Txid:       "99999",
		ReasonId:   "9",
		ReasonMemo: "buyer no longer wants",
	}
	raw, err := json.Marshal(body)
	require.NoError(t, err)

	s := string(raw)
	assert.Contains(t, s, `"txid":"99999"`,
		"CancelOrder must serialize the order id as `txid` (v7.2.78)")
	assert.NotContains(t, s, `"trade_id"`)
	assert.Contains(t, s, `"reason_id":"9"`)
	assert.Contains(t, s, `"reason_memo":"buyer no longer wants"`)
}

func TestCancelOrder_ReasonFieldsOmitempty(t *testing.T) {
	raw, err := json.Marshal(gateapi.CancelOrder{Txid: "1"})
	require.NoError(t, err)
	s := string(raw)
	assert.NotContains(t, s, `"reason_id"`,
		"reason_id is optional; empty value must be omitted")
	assert.NotContains(t, s, `"reason_memo"`)
}

// ---------------------------------------------------------------------------
// Layer 3 — RunE end-to-end against a mock Gate server.
// Exercises the full handler including the SDK call, locking in:
//   - the new Txid wire field (handlers must not regress on the rename)
//   - the new return type for P2pMerchantBooksPlaceBizPushOrder
//   - the removed opts argument for P2pMerchantBooksAdsUpdateStatus
// ---------------------------------------------------------------------------

func TestRunConfirmPayment_Succeeds_AgainstMockServer(t *testing.T) {
	// Capture the request body to verify the wire payload uses `txid`.
	var captured string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		captured = string(buf)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"message":"ok"}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("GATE_BASE_URL", srv.URL)

	silenceP2pStdout(t)
	root := authedP2pTestRoot(t)
	cmd := &cobra.Command{Use: "confirm-payment"}
	cmd.Flags().String("trade-id", "55555", "")
	cmd.Flags().String("payment-method", "bank", "")
	root.AddCommand(cmd)

	err := runConfirmPayment(cmd, nil)
	require.NoError(t, err, "runConfirmPayment must succeed against a mock 200 response")

	// Wire-level proof that the v7.2.78 field rename reaches the server.
	assert.Contains(t, captured, `"txid":"55555"`,
		"runConfirmPayment must POST txid (not trade_id) under v7.2.78")
	assert.NotContains(t, captured, `"trade_id"`)
	assert.Contains(t, captured, `"payment_method":"bank"`)
}

func TestRunConfirmReceipt_Succeeds_AgainstMockServer(t *testing.T) {
	var captured string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		captured = string(buf)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"message":"ok"}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("GATE_BASE_URL", srv.URL)

	silenceP2pStdout(t)
	root := authedP2pTestRoot(t)
	cmd := &cobra.Command{Use: "confirm-receipt"}
	cmd.Flags().String("trade-id", "77777", "")
	root.AddCommand(cmd)

	err := runConfirmReceipt(cmd, nil)
	require.NoError(t, err)

	assert.Contains(t, captured, `"txid":"77777"`)
	assert.NotContains(t, captured, `"trade_id"`)
}

func TestRunTransactionCancel_Succeeds_AgainstMockServer(t *testing.T) {
	var captured string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		captured = string(buf)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"message":"ok"}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("GATE_BASE_URL", srv.URL)

	silenceP2pStdout(t)
	root := authedP2pTestRoot(t)
	cmd := &cobra.Command{Use: "cancel"}
	cmd.Flags().String("trade-id", "11111", "")
	cmd.Flags().String("reason-id", "9", "")
	cmd.Flags().String("reason-memo", "test reason", "")
	root.AddCommand(cmd)

	err := runTransactionCancel(cmd, nil)
	require.NoError(t, err)

	assert.Contains(t, captured, `"txid":"11111"`)
	assert.NotContains(t, captured, `"trade_id"`)
	assert.Contains(t, captured, `"reason_id":"9"`)
	assert.Contains(t, captured, `"reason_memo":"test reason"`)
}

// runAdsUpdateStatus dropped the third opts argument in v7.2.78. Locking the
// happy path proves the handler compiles against the new two-arg signature
// and exercises the body it sends.
func TestRunAdsUpdateStatus_Succeeds_AgainstMockServer(t *testing.T) {
	var capturedQuery, capturedBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.RawQuery
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		capturedBody = string(buf)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"status":1}}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("GATE_BASE_URL", srv.URL)

	silenceP2pStdout(t)
	root := authedP2pTestRoot(t)
	cmd := &cobra.Command{Use: "update-status"}
	cmd.Flags().Int32("adv-no", 12345, "")
	cmd.Flags().Int32("adv-status", 1, "")
	root.AddCommand(cmd)

	err := runAdsUpdateStatus(cmd, nil)
	require.NoError(t, err)

	// v7.2.78 removed the trade_type query param; verify it never gets sent.
	assert.NotContains(t, capturedQuery, "trade_type",
		"trade_type query param was removed in v7.2.78 and must not appear")
	assert.Contains(t, capturedBody, `"adv_no":12345`)
	assert.Contains(t, capturedBody, `"adv_status":1`)
}

// runPushOrder's underlying SDK method changed return type from
// map[string]interface{} to P2pMerchantBooksPlaceBizPushOrderResponse in
// v7.2.78. A successful round-trip proves the new struct type unmarshals.
func TestRunPushOrder_Succeeds_AgainstMockServer(t *testing.T) {
	mockP2pGateServer(t, `{"code":0,"message":"ok","timestamp":1700000000.5,"data":{}}`)
	silenceP2pStdout(t)
	root := authedP2pTestRoot(t)
	cmd := &cobra.Command{Use: "push-order"}
	cmd.Flags().String("json", `{"currencyType":"USDT","exchangeType":"CNY","type":"0","unitPrice":"7.0","number":"100","payType":"bank","minAmount":"100","maxAmount":"10000"}`, "")
	root.AddCommand(cmd)

	err := runPushOrder(cmd, nil)
	assert.NoError(t, err,
		"push-order must accept the new P2pMerchantBooksPlaceBizPushOrderResponse return type")
}

// ---------------------------------------------------------------------------
// Layer 4 — Auth gate. The RequireAuth wall must still trigger when no
// credentials are configured. Guards against accidental auth removal during
// future refactors of these handlers.
// ---------------------------------------------------------------------------

func TestRunConfirmPayment_RequiresAuth(t *testing.T) {
	root := newP2pTestRoot(t)
	cmd := &cobra.Command{Use: "confirm-payment"}
	cmd.Flags().String("trade-id", "1", "")
	cmd.Flags().String("payment-method", "bank", "")
	root.AddCommand(cmd)

	err := runConfirmPayment(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

func TestRunConfirmReceipt_RequiresAuth(t *testing.T) {
	root := newP2pTestRoot(t)
	cmd := &cobra.Command{Use: "confirm-receipt"}
	cmd.Flags().String("trade-id", "1", "")
	root.AddCommand(cmd)

	err := runConfirmReceipt(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

func TestRunTransactionCancel_RequiresAuth(t *testing.T) {
	root := newP2pTestRoot(t)
	cmd := &cobra.Command{Use: "cancel"}
	cmd.Flags().String("trade-id", "1", "")
	cmd.Flags().String("reason-id", "", "")
	cmd.Flags().String("reason-memo", "", "")
	root.AddCommand(cmd)

	err := runTransactionCancel(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

func TestRunAdsUpdateStatus_RequiresAuth(t *testing.T) {
	root := newP2pTestRoot(t)
	cmd := &cobra.Command{Use: "update-status"}
	cmd.Flags().Int32("adv-no", 1, "")
	cmd.Flags().Int32("adv-status", 1, "")
	root.AddCommand(cmd)

	err := runAdsUpdateStatus(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

func TestRunPushOrder_RequiresAuth(t *testing.T) {
	root := newP2pTestRoot(t)
	cmd := &cobra.Command{Use: "push-order"}
	cmd.Flags().String("json", `{}`, "")
	root.AddCommand(cmd)

	err := runPushOrder(cmd, nil)
	require.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "api key")
}

// ---------------------------------------------------------------------------
// Layer 5 — Adjacent v7.2.78 model surface used by --json input. Lock the
// wire-shape changes so they cannot regress without test failure.
// ---------------------------------------------------------------------------

// PlaceBizPushOrder dropped HidePayment and added TeamPaymentUid in v7.2.78.
func TestPlaceBizPushOrder_FieldShape(t *testing.T) {
	body := gateapi.PlaceBizPushOrder{
		CurrencyType:   "USDT",
		ExchangeType:   "CNY",
		Type:           "0",
		UnitPrice:      "7.0",
		Number:         "100",
		PayType:        "bank",
		MinAmount:      "100",
		MaxAmount:      "10000",
		TeamPaymentUid: "team-uid-001",
	}
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	s := string(raw)

	assert.Contains(t, s, `"team_payment_uid":"team-uid-001"`,
		"v7.2.78 added team_payment_uid")
	assert.NotContains(t, s, `"hide_payment"`,
		"v7.2.78 removed HidePayment from PlaceBizPushOrder")
}

// P2pTransactionActionResponse changed Timestamp from int32 → float32 in
// v7.2.78 and added Method/Data/Version. Verify the new fields parse so
// runConfirm* output is correctly decoded.
//
// The timestamp value below is deliberately small: float32 has only 23 mantissa
// bits, so 17-billion-class timestamps lose seconds-level precision and
// fractional comparison becomes flaky. The point of the test is the type and
// the new field set, not real-world clock values.
func TestP2pTransactionActionResponse_NewShape(t *testing.T) {
	payload := `{"timestamp":12345.5,"method":"confirm","code":0,"message":"ok","data":{},"version":"v4"}`
	var resp gateapi.P2pTransactionActionResponse
	require.NoError(t, json.Unmarshal([]byte(payload), &resp))

	assert.InDelta(t, 12345.5, resp.Timestamp, 0.001,
		"Timestamp is now float32 in v7.2.78")
	assert.Equal(t, "confirm", resp.Method)
	assert.Equal(t, int32(0), resp.Code)
	assert.Equal(t, "ok", resp.Message)
	assert.Equal(t, "v4", resp.Version)
}

// ---------------------------------------------------------------------------
// Layer 6 — v7.2.78 silent-behavior changes that gate-cli does not modify
// directly but inherits via the SDK upgrade.
//
// (b) GetChatsListRequest.Txid: int32 with omitempty.
//     v7.2.71 always serialized "txid":0; v7.2.78 omits it entirely. Per the
//     server contract, an omitted/zero txid means "return the latest order
//     with chat for the user". Locking this here protects users whose
//     scripts pass --txid 0 from a silent server-side semantics shift.
//
// (d) PlaceBizPushOrder.HidePayment was deleted in v7.2.78.
//     User --json blobs that still include "hide_payment" must be silently
//     dropped (Go's encoding/json ignores unknown fields by default), and
//     must NOT reach the server in the outbound POST body. The new
//     team_payment_uid field, conversely, must be passed through.
// ---------------------------------------------------------------------------

// (b1) Wire-shape check at the model layer: zero-valued Txid must not
// appear in the encoded payload under v7.2.78.
func TestGetChatsListRequest_TxidOmitemptyOnZero(t *testing.T) {
	body := gateapi.GetChatsListRequest{Txid: 0}
	raw, err := json.Marshal(body)
	require.NoError(t, err)

	assert.NotContains(t, string(raw), `"txid"`,
		"v7.2.78 added omitempty to GetChatsListRequest.Txid; zero value must not be sent so the server can apply its 'latest order' fallback")
}

// (b2) Non-zero Txid must still be sent, otherwise we just broke a user's
// ability to query a specific order.
func TestGetChatsListRequest_TxidPreservedWhenSet(t *testing.T) {
	body := gateapi.GetChatsListRequest{Txid: 42}
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"txid":42`,
		"non-zero txid must still serialize; omitempty only elides zero")
}

// (b3) Lastreceived/Firstreceived already had omitempty in v7.2.71 — pin
// here so the trio behaves consistently and a future regen does not flip
// one tag without the others.
func TestGetChatsListRequest_PaginationOmitemptyOnZero(t *testing.T) {
	raw, err := json.Marshal(gateapi.GetChatsListRequest{Txid: 1})
	require.NoError(t, err)
	s := string(raw)
	assert.NotContains(t, s, `"lastreceived"`)
	assert.NotContains(t, s, `"firstreceived"`)
}

// (b4) End-to-end: when the CLI is invoked with --txid=0 (the documented
// "give me the latest order" shortcut), the outbound POST body must not
// carry a `txid` key. Captures the actual wire payload.
//
// Note on `MarkFlagRequired`: required only checks whether the flag was set
// during parsing, not whether the value is non-zero, so `--txid 0` is a
// valid, accepted CLI invocation. The fact that v7.2.78 also turns this
// into a meaningful request (rather than a hard-coded txid=0 lookup) is
// what we lock down below.
func TestRunChatList_TxidZero_OmitsField_AgainstMockServer(t *testing.T) {
	var captured string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		captured = string(buf)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("GATE_BASE_URL", srv.URL)

	silenceP2pStdout(t)
	root := authedP2pTestRoot(t)
	cmd := &cobra.Command{Use: "list"}
	cmd.Flags().Int32("txid", 0, "")
	cmd.Flags().Int32("lastreceived", 0, "")
	cmd.Flags().Int32("firstreceived", 0, "")
	root.AddCommand(cmd)

	err := runChatList(cmd, nil)
	require.NoError(t, err)

	assert.NotContains(t, captured, `"txid"`,
		"runChatList with --txid 0 must POST a body without txid (v7.2.78 'latest order' contract)")
}

// (b5) End-to-end happy path: with a real txid, the field reaches the
// server. Guards against an over-eager omitempty regression on non-zero.
func TestRunChatList_TxidNonZero_IsSent_AgainstMockServer(t *testing.T) {
	var captured string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		captured = string(buf)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("GATE_BASE_URL", srv.URL)

	silenceP2pStdout(t)
	root := authedP2pTestRoot(t)
	cmd := &cobra.Command{Use: "list"}
	cmd.Flags().Int32("txid", 9876, "")
	cmd.Flags().Int32("lastreceived", 0, "")
	cmd.Flags().Int32("firstreceived", 0, "")
	root.AddCommand(cmd)

	err := runChatList(cmd, nil)
	require.NoError(t, err)
	assert.Contains(t, captured, `"txid":9876`)
}

// (d1) Reverse direction: a legacy --json blob carrying the now-removed
// `hide_payment` key must unmarshal cleanly (Go's encoding/json silently
// drops unknown keys), and the deserialized struct must not contain that
// data — so it cannot leak back onto the wire.
//
// This is the user-script protection: someone may have CI scripts built
// against the v7.2.71 wire shape; we want them to get a graceful no-op on
// hide_payment, not a parse error.
func TestPlaceBizPushOrder_LegacyHidePaymentSilentlyDropped(t *testing.T) {
	legacyJSON := `{
		"currencyType": "USDT",
		"exchangeType": "CNY",
		"type":         "0",
		"unitPrice":    "7.0",
		"number":       "100",
		"payType":      "bank",
		"minAmount":    "100",
		"maxAmount":    "10000",
		"hide_payment": "1"
	}`
	var body gateapi.PlaceBizPushOrder
	require.NoError(t, json.Unmarshal([]byte(legacyJSON), &body),
		"legacy blob with hide_payment must still unmarshal — Go drops unknown keys")

	// Round-trip: re-encode and prove hide_payment did not survive the
	// trip into the struct (it has no field to bind to).
	reencoded, err := json.Marshal(body)
	require.NoError(t, err)
	assert.NotContains(t, string(reencoded), `"hide_payment"`,
		"hide_payment was dropped by the v7.2.78 model and must not reappear after re-encoding")

	// Sanity: the supported fields survived.
	assert.Equal(t, "USDT", body.CurrencyType)
	assert.Equal(t, "bank", body.PayType)
}

// (d2) End-to-end: users posting a v7.2.71-style JSON containing
// `hide_payment` get a silent drop, not a hard failure. The request still
// goes out, just without that field. This is the CLI-level user contract.
func TestRunPushOrder_LegacyHidePayment_DroppedOnWire(t *testing.T) {
	var captured string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		captured = string(buf)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{}}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("GATE_BASE_URL", srv.URL)

	silenceP2pStdout(t)
	root := authedP2pTestRoot(t)
	cmd := &cobra.Command{Use: "push-order"}
	cmd.Flags().String("json", `{"currencyType":"USDT","exchangeType":"CNY","type":"0","unitPrice":"7.0","number":"100","payType":"bank","minAmount":"100","maxAmount":"10000","hide_payment":"1"}`, "")
	root.AddCommand(cmd)

	err := runPushOrder(cmd, nil)
	require.NoError(t, err,
		"push-order with a legacy hide_payment field must still succeed (graceful migration for user scripts)")

	assert.NotContains(t, captured, `"hide_payment"`,
		"v7.2.78 dropped hide_payment; it must not be relayed to the server even if the user supplies it")
	// The other flags must still reach the server — proves the legacy
	// field is the only thing that disappears.
	assert.Contains(t, captured, `"currencyType":"USDT"`)
	assert.Contains(t, captured, `"payType":"bank"`)
}

// (d3) Forward direction: users passing the v7.2.78-only `team_payment_uid`
// field must see it pass through to the server. Pairs with (d1) to lock
// "drops the deprecated field, forwards the new one" behavior.
func TestRunPushOrder_TeamPaymentUid_ForwardedOnWire(t *testing.T) {
	var captured string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		captured = string(buf)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{}}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("GATE_BASE_URL", srv.URL)

	silenceP2pStdout(t)
	root := authedP2pTestRoot(t)
	cmd := &cobra.Command{Use: "push-order"}
	cmd.Flags().String("json", `{"currencyType":"USDT","exchangeType":"CNY","type":"0","unitPrice":"7.0","number":"100","payType":"bank","minAmount":"100","maxAmount":"10000","team_payment_uid":"team-001"}`, "")
	root.AddCommand(cmd)

	err := runPushOrder(cmd, nil)
	require.NoError(t, err)

	assert.Contains(t, captured, `"team_payment_uid":"team-001"`,
		"v7.2.78 added team_payment_uid; user-supplied value must round-trip into the request")
}
