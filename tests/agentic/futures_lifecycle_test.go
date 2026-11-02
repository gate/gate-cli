//go:build agentic_e2e

package agentic_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/gate/gate-cli/tests/contracts"
)

// futuresLedger models only the exchange business boundary. It owns positions,
// pending/finished orders and fills across independent CLI processes. Explicit
// transitions make a read-back assertion depend on the preceding actual write.
// It is not a matching-engine, exchange accounting or live integration proof.
type futuresLedger struct {
	mu                 sync.Mutex
	dual               bool
	long, short        int64
	next               int64
	orders             map[int64]map[string]any
	trades             []map[string]any
	closed             []map[string]any
	writes             int
	failNext, loseNext bool
	auth               string
	t                  *testing.T
}

func newFuturesLedger(t *testing.T, auth string) (*futuresLedger, *httptest.Server) {
	t.Helper()
	l := &futuresLedger{t: t, auth: auth, next: 9007199254740993, orders: map[int64]map[string]any{}}
	peer := httptest.NewServer(http.HandlerFunc(l.serve))
	t.Cleanup(peer.Close)
	return l, peer
}

func (l *futuresLedger) position(mode string) map[string]any {
	size := l.long - l.short
	if mode == "dual_long" {
		size = l.long
	}
	if mode == "dual_short" {
		size = -l.short
	}
	return map[string]any{"contract": "SYNTH_USDT", "size": strconv.FormatInt(size, 10), "mode": mode, "entry_price": "10.00000001", "mark_price": "10.00000001", "leverage": "2"}
}

func (l *futuresLedger) serve(w http.ResponseWriter, r *http.Request) {
	l.mu.Lock()
	defer l.mu.Unlock()
	raw, err := io.ReadAll(io.LimitReader(r.Body, 8193))
	if err != nil || len(raw) > 8192 {
		l.t.Error("unbounded or unreadable scenario body")
		http.Error(w, "bad body", 400)
		return
	}
	wireAuth := l.auth
	if r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/contracts") {
		wireAuth = "anonymous"
	}
	if err := contracts.CheckAuth(r, raw, wireAuth); err != nil {
		l.t.Error(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Gate-Trace-Id", "lifecycle-trace")
	respond := func(v any) {
		if err := json.NewEncoder(w).Encode(v); err != nil {
			l.t.Error(err)
		}
	}
	reject := func(status int, label string) {
		w.WriteHeader(status)
		respond(map[string]string{"label": label, "message": "synthetic lifecycle rejection"})
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v4/futures/"), "/")
	if len(parts) < 2 || (parts[0] != "usdt" && parts[0] != "btc") {
		reject(404, "UNKNOWN_SETTLEMENT")
		return
	}
	path := "/" + strings.Join(parts[1:], "/")
	if r.Method == "POST" && l.failNext {
		l.failNext = false
		reject(503, "REJECTED_BEFORE_APPLY")
		return
	}
	switch {
	case r.Method == "GET" && path == "/accounts":
		respond(map[string]any{"currency": strings.ToUpper(parts[0]), "total": "1000.00000001", "available": "900.00000001", "in_dual_mode": l.dual})
	case r.Method == "POST" && path == "/dual_mode":
		if l.long != 0 || l.short != 0 {
			reject(400, "POSITIONS_NOT_FLAT")
			return
		}
		l.dual = r.URL.Query().Get("dual_mode") == "true"
		respond(map[string]any{"in_dual_mode": l.dual})
	case r.Method == "GET" && path == "/contracts":
		respond([]map[string]any{{"name": "SYNTH_USDT", "order_size_min": "1", "order_size_max": "1000", "quanto_multiplier": "0.0001"}})
	case r.Method == "GET" && path == "/positions/SYNTH_USDT":
		respond(l.position("single"))
	case r.Method == "GET" && path == "/dual_comp/positions/SYNTH_USDT":
		respond([]map[string]any{l.position("dual_long"), l.position("dual_short")})
	case r.Method == "GET" && path == "/positions":
		out := []map[string]any{}
		if l.long != 0 || l.short != 0 {
			if l.dual {
				out = append(out, l.position("dual_long"), l.position("dual_short"))
			} else {
				out = append(out, l.position("single"))
			}
		}
		respond(out)
	case r.Method == "POST" && path == "/positions/SYNTH_USDT/leverage":
		respond(l.position("single"))
	case r.Method == "GET" && path == "/position_close":
		respond(l.closed)
	case r.Method == "GET" && (path == "/my_trades" || path == "/my_trades_timerange"):
		respond(l.trades)
	case path == "/orders" && r.Method == "POST":
		var order struct {
			Contract, Size, Price, Tif, Text string
			Close                            bool
			ReduceOnly                       bool   `json:"reduce_only"`
			AutoSize                         string `json:"auto_size"`
		}
		if json.Unmarshal(raw, &order) != nil || order.Contract != "SYNTH_USDT" {
			reject(400, "INVALID_ORDER")
			return
		}
		size, err := strconv.ParseInt(order.Size, 10, 64)
		if err != nil || size > 1000 || size < -1000 {
			reject(400, "INVALID_SIZE")
			return
		}
		market := order.Price == "0" && order.Tif == "ioc"
		beforeLong, beforeShort := l.long, l.short
		if market {
			switch {
			case order.Close:
				l.long, l.short = 0, 0
			case order.AutoSize == "close_long":
				l.long = 0
			case order.AutoSize == "close_short":
				l.short = 0
			case order.ReduceOnly:
				if size < 0 && l.long >= -size {
					l.long += size
				} else if size > 0 && l.short >= size {
					l.short -= size
				} else {
					reject(400, "REDUCE_ONLY_REJECT")
					return
				}
			case size > 0:
				if l.dual {
					l.long += size
				} else {
					net := l.long - l.short + size
					l.long, l.short = 0, 0
					if net >= 0 {
						l.long = net
					} else {
						l.short = -net
					}
				}
			case size < 0:
				if l.dual {
					l.short -= size
				} else {
					net := l.long - l.short + size
					l.long, l.short = 0, 0
					if net >= 0 {
						l.long = net
					} else {
						l.short = -net
					}
				}
			default:
				reject(400, "ZERO_SIZE")
				return
			}
		}
		id := l.next
		l.next++
		l.writes++
		status, finish := "open", ""
		if market {
			status, finish = "finished", "filled"
			l.trades = append(l.trades, map[string]any{"id": id, "order_id": strconv.FormatInt(id, 10), "contract": order.Contract, "size": order.Size, "price": "10.00000001"})
		}
		if beforeLong+beforeShort > 0 && l.long+l.short == 0 {
			l.closed = append(l.closed, map[string]any{"contract": order.Contract, "pnl": "0.00000001", "side": "long"})
		}
		result := map[string]any{"id": id, "contract": order.Contract, "size": order.Size, "price": order.Price, "tif": order.Tif, "text": order.Text, "status": status, "finish_as": finish, "reduce_only": order.ReduceOnly}
		l.orders[id] = result
		if l.loseNext {
			l.loseNext = false
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				l.t.Error(err)
				return
			}
			if err := conn.Close(); err != nil {
				l.t.Error(err)
			}
			return
		}
		respond(result)
	case path == "/orders" && r.Method == "GET":
		out := []map[string]any{}
		ids := make([]int64, 0, len(l.orders))
		for id := range l.orders {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		for _, id := range ids {
			o := l.orders[id]
			if o["status"] == r.URL.Query().Get("status") {
				out = append(out, o)
			}
		}
		respond(out)
	case path == "/orders" && r.Method == "DELETE":
		out := []map[string]any{}
		for _, o := range l.orders {
			if o["status"] == "open" {
				o["status"], o["finish_as"] = "finished", "cancelled"
				out = append(out, o)
			}
		}
		respond(out)
	case strings.HasPrefix(path, "/orders/"):
		identity := strings.TrimPrefix(path, "/orders/")
		id, _ := strconv.ParseInt(identity, 10, 64)
		order, ok := l.orders[id]
		if strings.HasPrefix(identity, "t-") {
			for _, candidate := range l.orders {
				if candidate["text"] == identity {
					order, ok = candidate, true
					break
				}
			}
		}
		if !ok {
			reject(404, "ORDER_NOT_FOUND")
			return
		}
		switch r.Method {
		case "GET":
			respond(order)
		case "PUT":
			if order["status"] != "open" {
				reject(400, "ORDER_FINISHED")
				return
			}
			var body map[string]json.RawMessage
			if json.Unmarshal(raw, &body) != nil {
				reject(400, "INVALID_AMEND")
				return
			}
			for _, key := range []string{"size", "price"} {
				if v, ok := body[key]; ok {
					var text string
					if json.Unmarshal(v, &text) != nil {
						reject(400, "INVALID_AMEND")
						return
					}
					order[key] = text
				}
			}
			respond(order)
		case "DELETE":
			if order["status"] != "open" {
				reject(400, "ORDER_FINISHED")
				return
			}
			order["status"], order["finish_as"] = "finished", "cancelled"
			respond(order)
		default:
			reject(405, "UNEXPECTED_METHOD")
		}
	default:
		l.t.Errorf("unexpected lifecycle request: %s %s", r.Method, r.URL)
		reject(404, "UNEXPECTED_ROUTE")
	}
}

func futuresObject(t *testing.T, raw []byte) map[string]json.RawMessage {
	t.Helper()
	var out map[string]json.RawMessage
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("expected object: %v %s", err, raw)
	}
	return out
}

func futuresField(t *testing.T, object map[string]json.RawMessage, key, want string) {
	t.Helper()
	got := strings.Trim(string(object[key]), `"`)
	if got != want {
		t.Fatalf("%s=%s, want %s", key, got, want)
	}
}

// Both directions and both settlement routes exercise a full order/position
// lifecycle. Decimal price and IDs above 2^53 must survive every command hop.
func TestFuturesTradingLifecycle(t *testing.T) {
	for _, auth := range []string{"hmac", "oauth"} {
		for _, settle := range []string{"usdt", "btc"} {
			for _, dual := range []bool{false, true} {
				for _, side := range []string{"long", "short"} {
					t.Run(fmt.Sprintf("%s/%s/dual=%t/%s", auth, settle, dual, side), func(t *testing.T) {
						ledger, peer := newFuturesLedger(t, auth)
						home := futuresConfirmedHome(t, peer.URL, auth)
						invoke := func(args ...string) []byte {
							t.Helper()
							argv := append([]string{"--format", "json", "cex", "futures"}, args...)
							argv = append(argv, "--settle", settle)
							out, stderr, err := futuresConfirmedInvoke(t, cliBinary, home, peer.URL, auth, argv...)
							if err != nil || len(stderr) != 0 {
								t.Fatalf("%v: %v %s %s", args, err, out, stderr)
							}
							return out
						}
						assertSize := func(want int64) {
							t.Helper()
							group, mode := "get", "single"
							if dual {
								group, mode = "get-dual", "dual_"+side
							}
							raw := invoke("position", group, "--contract", "SYNTH_USDT")
							if dual {
								var rows []map[string]json.RawMessage
								if err := json.Unmarshal(raw, &rows); err != nil {
									t.Fatal(err)
								}
								for _, row := range rows {
									if strings.Trim(string(row["mode"]), `"`) == mode {
										futuresField(t, row, "size", fmt.Sprint(want))
										return
									}
								}
								t.Fatal("requested dual side missing")
							} else {
								futuresField(t, futuresObject(t, raw), "size", fmt.Sprint(want))
							}
						}
						futuresField(t, futuresObject(t, invoke("account", "get")), "total", "1000.00000001")
						if dual {
							invoke("account", "dual-mode", "--enable=true")
						}
						if !bytes.Contains(invoke("market", "contracts"), []byte("SYNTH_USDT")) {
							t.Fatal("contract discovery lost")
						}
						invoke("position", "update-leverage", "--contract", "SYNTH_USDT", "--leverage", "2")
						opened := futuresObject(t, invoke("order", side, "--contract", "SYNTH_USDT", "--size", "10", "--text", "t-open"))
						futuresField(t, opened, "id", "9007199254740993")
						futuresField(t, opened, "status", "finished")
						sign := int64(1)
						if side == "short" {
							sign = -1
						}
						assertSize(10 * sign)
						futuresField(t, futuresObject(t, invoke("order", "get-by-text", "--text", "t-open")), "id", "9007199254740993")
						invoke("order", "add", "--contract", "SYNTH_USDT", "--size", "4")
						assertSize(14 * sign)
						reduced := futuresObject(t, invoke("order", "remove", "--contract", "SYNTH_USDT", "--size", "3"))
						futuresField(t, reduced, "reduce_only", "true")
						assertSize(11 * sign)
						invoke("order", "close", "--contract", "SYNTH_USDT", "--side", side, "--size", "2")
						assertSize(9 * sign)
						pending := futuresObject(t, invoke("order", side, "--contract", "SYNTH_USDT", "--size", "1", "--price", "10.00000001", "--text", "t-pending"))
						id := string(pending["id"])
						futuresField(t, pending, "status", "open")
						assertSize(9 * sign)
						invoke("order", "amend", "--id", id, "--price", "9.00000001")
						futuresField(t, futuresObject(t, invoke("order", "get", "--id", id)), "price", "9.00000001")
						var open []json.RawMessage
						if err := json.Unmarshal(invoke("order", "list", "--contract", "SYNTH_USDT", "--status", "open"), &open); err != nil || len(open) != 1 {
							t.Fatalf("pending inventory: %v count=%d", err, len(open))
						}
						invoke("order", "cancel", "--id", id)
						futuresField(t, futuresObject(t, invoke("order", "get", "--id", id)), "finish_as", "cancelled")
						// Rejected writes do not mutate; an accepted write with a lost response
						// does mutate once and must be reconciled by its correlation text.
						ledger.mu.Lock()
						ledger.failNext = true
						before := ledger.writes
						ledger.mu.Unlock()
						argv := []string{"--format", "json", "cex", "futures", "order", side, "--contract", "SYNTH_USDT", "--size", "1", "--settle", settle, "--text", "t-uncertain"}
						out, stderr, err := futuresConfirmedInvoke(t, cliBinary, home, peer.URL, auth, argv...)
						if err == nil || len(out) != 0 || !bytes.Contains(stderr, []byte("REJECTED_BEFORE_APPLY")) {
							t.Fatalf("rejection lost: %v %s %s", err, out, stderr)
						}
						assertSize(9 * sign)
						ledger.mu.Lock()
						if ledger.writes != before {
							t.Error("rejected write changed ledger")
						}
						ledger.loseNext = true
						ledger.mu.Unlock()
						out, stderr, err = futuresConfirmedInvoke(t, cliBinary, home, peer.URL, auth, argv...)
						if err == nil || len(out) != 0 || !bytes.Contains(stderr, []byte("outcome unknown")) {
							t.Fatalf("unknown write lost: %v %s %s", err, out, stderr)
						}
						assertSize(10 * sign)
						futuresField(t, futuresObject(t, invoke("order", "get-by-text", "--text", "t-uncertain")), "status", "finished")
						ledger.mu.Lock()
						if ledger.writes != before+1 {
							t.Error("unknown write was replayed")
						}
						ledger.mu.Unlock()
						invoke("order", side, "--contract", "SYNTH_USDT", "--size", "1", "--price", "8")
						invoke("order", "cancel", "--all", "--contract", "SYNTH_USDT")
						invoke("order", "close", "--contract", "SYNTH_USDT", "--side", side)
						assertSize(0)
						if !contracts.EqualJSON(invoke("position", "list"), []byte(`[]`)) {
							t.Fatal("positions not flat")
						}
						if !bytes.Contains(invoke("position", "close-history", "--contract", "SYNTH_USDT"), []byte("0.00000001")) {
							t.Fatal("close history missing")
						}
						var trades []json.RawMessage
						if err := json.Unmarshal(invoke("order", "my-trades", "--contract", "SYNTH_USDT"), &trades); err != nil || len(trades) != 6 {
							t.Fatalf("fills=%d want 6: %v", len(trades), err)
						}
						for _, direction := range []string{"add", "remove"} {
							_, stderr, err := futuresConfirmedInvoke(t, cliBinary, home, peer.URL, auth, "--format", "json", "cex", "futures", "order", direction, "--contract", "SYNTH_USDT", "--size", "1", "--settle", settle)
							if err == nil || !bytes.Contains(stderr, []byte("no open position")) {
								t.Fatalf("flat %s accepted: %v %s", direction, err, stderr)
							}
						}
						ledger.mu.Lock()
						defer ledger.mu.Unlock()
						if ledger.long != 0 || ledger.short != 0 || ledger.writes != 8 {
							t.Fatalf("final ledger: long=%d short=%d writes=%d", ledger.long, ledger.short, ledger.writes)
						}
					})
				}
			}
		}
	}
}

// Closing one side of a hedged account must leave the other side untouched.
// Unlike single-sided dual scenarios, both positions are nonzero concurrently.
func TestFuturesDualSideIsolation(t *testing.T) {
	for _, auth := range []string{"hmac", "oauth"} {
		t.Run(auth, func(t *testing.T) {
			ledger, peer := newFuturesLedger(t, auth)
			home := futuresConfirmedHome(t, peer.URL, auth)
			invoke := func(args ...string) []byte {
				t.Helper()
				out, stderr, err := futuresConfirmedInvoke(t, cliBinary, home, peer.URL, auth, append([]string{"--format", "json", "cex", "futures"}, args...)...)
				if err != nil {
					t.Fatalf("%v: %v %s %s", args, err, out, stderr)
				}
				return out
			}
			invoke("account", "dual-mode", "--enable=true")
			invoke("order", "long", "--contract", "SYNTH_USDT", "--size", "5")
			invoke("order", "short", "--contract", "SYNTH_USDT", "--size", "7")
			check := func(long, short string) {
				t.Helper()
				var rows []map[string]json.RawMessage
				if err := json.Unmarshal(invoke("position", "get-dual", "--contract", "SYNTH_USDT"), &rows); err != nil || len(rows) != 2 {
					t.Fatalf("dual snapshot: %v %s", err, rows)
				}
				futuresField(t, rows[0], "size", long)
				futuresField(t, rows[1], "size", short)
			}
			check("5", "-7")
			invoke("order", "close", "--contract", "SYNTH_USDT", "--side", "short", "--size", "2")
			check("5", "-5")
			invoke("order", "close", "--contract", "SYNTH_USDT", "--side", "short")
			check("5", "0")
			invoke("order", "close", "--contract", "SYNTH_USDT", "--side", "long")
			check("0", "0")
			ledger.mu.Lock()
			defer ledger.mu.Unlock()
			if ledger.long != 0 || ledger.short != 0 || ledger.writes != 5 {
				t.Fatalf("hedge isolation failed: %+v", ledger)
			}
		})
	}
}
