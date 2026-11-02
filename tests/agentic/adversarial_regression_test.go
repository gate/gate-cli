//go:build agentic_e2e

package agentic_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gate/gate-cli/tests/contracts"
)

// A separate explicit acceptance gate preserves known failures instead of
// weakening their assertions or misreporting the historical suite as broken.
// Only remote HTTP and synthetic credentials are mocked; the CLI is real.
func TestAdversarialCEXUnexpectedArguments(t *testing.T) {
	for _, tc := range contracts.Cases(t) {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			auth := "hmac"
			if tc.Public {
				auth = "anonymous"
			}
			peer := contracts.NewPeer(t, tc, auth, "success")
			args := append(append([]string{"--format", "json"}, tc.Args...), "__unexpected_position__")
			out, _, err := runCLI(t, peer.URL, true, args...)
			if err == nil || peer.Calls() != 0 || len(out) != 0 {
				t.Errorf("extra positional argument reached business execution: err=%v requests=%d stdout_bytes=%d", err, peer.Calls(), len(out))
			}
		})
	}
}

func TestAdversarialTradingIntentWire(t *testing.T) {
	for _, tc := range contracts.UnsafeIntents(t) {
		for _, auth := range []string{"hmac", "oauth"} {
			t.Run(tc.ID+"/"+auth, func(t *testing.T) {
				t.Parallel()
				peer := contracts.NewPeer(t, tc.Case, auth, "success")
				home := cexContractHome(t, peer.URL, auth)
				args := append([]string{"--format", "json"}, tc.Case.Args...)
				out, _, err := futuresConfirmedInvoke(t, cliBinary, home, peer.URL, auth, args...)
				if err == nil || peer.Calls() >= len(tc.Case.Exchanges) || len(out) != 0 {
					t.Errorf("unsafe intent reached terminal write: err=%v calls=%d forbidden_body=%s", err, peer.Calls(), tc.Case.Exchanges[len(tc.Case.Exchanges)-1].Body)
				}
			})
		}
	}
}

// Read back through the real CLI after each attempted invalid operation. The
// existing fixed-size ledger is the exchange boundary, not a live venue proof.
func TestAdversarialFuturesStatePreserved(t *testing.T) {
	for _, auth := range []string{"hmac", "oauth"} {
		for _, scenario := range []string{"negative-long", "negative-add", "invalid-close-side"} {
			t.Run(scenario+"/"+auth, func(t *testing.T) {
				ledger, peer := newFuturesLedger(t, auth)
				home := futuresConfirmedHome(t, peer.URL, auth)
				invoke := func(args ...string) ([]byte, []byte, error) {
					return futuresConfirmedInvoke(t, cliBinary, home, peer.URL, auth, append([]string{"--format", "json", "cex", "futures"}, args...)...)
				}
				must := func(args ...string) []byte {
					t.Helper()
					out, stderr, err := invoke(args...)
					if err != nil || len(stderr) != 0 {
						t.Fatalf("setup/readback %v: %v %s", args, err, stderr)
					}
					return out
				}
				dual := scenario == "invalid-close-side"
				if dual {
					must("account", "dual-mode", "--enable=true")
				}
				wantLong, wantShort, wantWrites := int64(0), int64(0), 0
				if scenario != "negative-long" {
					must("order", "long", "--contract", "SYNTH_USDT", "--size", "10")
					wantLong, wantWrites = 10, 1
				}
				if dual {
					must("order", "short", "--contract", "SYNTH_USDT", "--size", "7")
					wantShort, wantWrites = 7, 2
				}
				args := []string{"order", "long", "--contract", "SYNTH_USDT", "--size=-2"}
				switch scenario {
				case "negative-add":
					args = []string{"order", "add", "--contract", "SYNTH_USDT", "--size=-14"}
				case "invalid-close-side":
					args = []string{"order", "close", "--contract", "SYNTH_USDT", "--side", "shorts"}
				}
				_, _, attempted := invoke(args...)
				group := "get"
				if dual {
					group = "get-dual"
				}
				readback := must("position", group, "--contract", "SYNTH_USDT")
				var rows []struct{ Mode, Size string }
				if dual {
					if err := json.Unmarshal(readback, &rows); err != nil {
						t.Fatal(err)
					}
				} else {
					var row struct{ Mode, Size string }
					if err := json.Unmarshal(readback, &row); err != nil {
						t.Fatal(err)
					}
					rows = append(rows, row)
				}
				wantRows := 1
				if dual {
					wantRows = 2
				}
				if len(rows) != wantRows {
					t.Fatalf("missing position readback: %s", readback)
				}
				ledger.mu.Lock()
				long, short, writes := ledger.long, ledger.short, ledger.writes
				ledger.mu.Unlock()
				// CLI readback must agree with the model, even when exposing a bug.
				for _, row := range rows {
					want := long - short
					if row.Mode == "dual_long" {
						want = long
					}
					if row.Mode == "dual_short" {
						want = -short
					}
					if row.Size != fmt.Sprint(want) {
						t.Fatalf("readback/model disagree: %+v long=%d short=%d", rows, long, short)
					}
				}
				if attempted == nil || long != wantLong || short != wantShort || writes != wantWrites {
					t.Errorf("invalid intent changed position: exit=%v long=%d short=%d writes=%d; want rejection and long=%d short=%d writes=%d; CLI_readback=%s", attempted, long, short, writes, wantLong, wantShort, wantWrites, readback)
				}
			})
		}
	}
}

// Common probes are deliberately local/read-only. Login and logout are not
// invoked with live credentials merely to check positional argument handling.
func TestAdversarialCommonUnexpectedArguments(t *testing.T) {
	for _, args := range [][]string{{"version"}, {"config", "list"}, {"config", "check"}, {"doctor", "--check", "config"}, {"cex", "auth", "status"}} {
		t.Run(strings.Join(args, "/"), func(t *testing.T) {
			home := cexContractHome(t, "http://127.0.0.1:1", "oauth")
			if _, stderr, err := futuresConfirmedInvoke(t, cliBinary, home, "http://127.0.0.1:1", "oauth", args...); err != nil {
				t.Fatalf("valid common-command control failed: %v %s", err, stderr)
			}
			out, _, err := futuresConfirmedInvoke(t, cliBinary, home, "http://127.0.0.1:1", "oauth", append(args, "__unexpected_position__")...)
			if err == nil || len(out) != 0 {
				t.Errorf("unexpected positional argument accepted: err=%v stdout_bytes=%d", err, len(out))
			}
		})
	}
}

// The remote fixture implements only the SDK-documented protection transition:
// null keeps protection, {} cancels it. Compare single and batch amendment on
// the same actually-created order, then query through the real CLI again.
func TestAdversarialSpotProtectionPreserved(t *testing.T) {
	for _, auth := range []string{"hmac", "oauth"} {
		for _, rawProtection := range []string{`{"trigger_price":null}`, `{"triger_price":"8"}`} {
			t.Run(auth+"/"+rawProtection, func(t *testing.T) {
				var mu sync.Mutex
				var stop json.RawMessage
				writes := 0
				peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					defer mu.Unlock()
					raw, err := io.ReadAll(io.LimitReader(r.Body, 8193))
					if err != nil || len(raw) > 8192 {
						t.Error("invalid bounded body")
						http.Error(w, "bad body", 400)
						return
					}
					if err := contracts.CheckAuth(r, raw, auth); err != nil {
						t.Error(err)
					}
					w.Header().Set("Content-Type", "application/json")
					respond := func(v any) {
						if err := json.NewEncoder(w).Encode(v); err != nil {
							t.Error(err)
						}
					}
					order := func() map[string]any {
						return map[string]any{"id": "9007199254740993", "currency_pair": "SYNTH_USDT", "side": "buy", "type": "limit", "price": "10", "amount": "1", "status": "open", "stop_loss": stop}
					}
					switch {
					case r.Method == "POST" && r.URL.Path == "/api/v4/spot/orders":
						var body map[string]json.RawMessage
						if json.Unmarshal(raw, &body) != nil {
							http.Error(w, "bad JSON", 400)
							return
						}
						stop = append(json.RawMessage{}, body["stop_loss"]...)
						writes++
						respond(order())
					case r.Method == "POST" && r.URL.Path == "/api/v4/spot/amend_batch_orders":
						var body []map[string]json.RawMessage
						if json.Unmarshal(raw, &body) != nil || len(body) != 1 {
							http.Error(w, "bad batch", 400)
							return
						}
						value := body[0]["stop_loss"]
						if len(value) != 0 && string(value) != "null" {
							if contracts.EqualJSON(value, []byte(`{}`)) {
								stop = json.RawMessage(`null`)
							} else {
								stop = append(json.RawMessage{}, value...)
							}
						}
						writes++
						respond([]map[string]any{{"id": "9007199254740993", "succeeded": true}})
					case r.Method == "GET" && r.URL.Path == "/api/v4/spot/orders/9007199254740993":
						respond(order())
					default:
						t.Errorf("unexpected protection route %s %s", r.Method, r.URL.Path)
						http.Error(w, "unexpected route", 404)
					}
				}))
				t.Cleanup(peer.Close)
				home := cexContractHome(t, peer.URL, auth)
				invoke := func(args ...string) ([]byte, []byte, error) {
					return futuresConfirmedInvoke(t, cliBinary, home, peer.URL, auth, append([]string{"--format", "json", "cex", "spot", "order"}, args...)...)
				}
				must := func(args ...string) []byte {
					t.Helper()
					out, stderr, err := invoke(args...)
					if err != nil || len(stderr) != 0 {
						t.Fatalf("setup/readback: %v %s", err, stderr)
					}
					return out
				}
				must("buy", "--pair", "SYNTH_USDT", "--amount", "1", "--price", "10", "--stop-loss-json", `{"trigger_price":"8"}`)
				before := futuresObject(t, must("get", "--pair", "SYNTH_USDT", "--id", "9007199254740993"))
				if !contracts.EqualJSON(before["stop_loss"], []byte(`{"trigger_price":"8"}`)) {
					t.Fatal("protected order precondition failed")
				}
				if _, _, err := invoke("amend", "--pair", "SYNTH_USDT", "--id", "9007199254740993", "--stop-loss-json", rawProtection); err == nil {
					t.Fatal("single amend control unexpectedly accepted malformed protection")
				}
				mu.Lock()
				controlWrites := writes
				mu.Unlock()
				if controlWrites != 1 {
					t.Fatal("single amend control reached remote write")
				}
				// A legitimate top-level null must preserve the existing stop. This
				// control distinguishes the invalid member from valid no-change input.
				must("batch-amend", "--orders-json", `[{"order_id":"9007199254740993","currency_pair":"SYNTH_USDT","amount":"2","stop_loss":null}]`)
				unchanged := futuresObject(t, must("get", "--pair", "SYNTH_USDT", "--id", "9007199254740993"))
				if !contracts.EqualJSON(unchanged["stop_loss"], before["stop_loss"]) {
					t.Fatal("valid null control changed protection")
				}
				_, _, err := invoke("batch-amend", "--orders-json", `[{"order_id":"9007199254740993","currency_pair":"SYNTH_USDT","amount":"2","stop_loss":`+rawProtection+`}]`)
				after := futuresObject(t, must("get", "--pair", "SYNTH_USDT", "--id", "9007199254740993"))
				mu.Lock()
				finalWrites := writes
				mu.Unlock()
				if err == nil || finalWrites != 2 || !contracts.EqualJSON(after["stop_loss"], before["stop_loss"]) {
					t.Errorf("malformed batch protection changed order: exit=%v writes=%d before=%s after=%s", err, finalWrites, before["stop_loss"], after["stop_loss"])
				}
			})
		}
	}
}
