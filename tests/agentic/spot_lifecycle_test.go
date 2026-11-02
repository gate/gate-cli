//go:build agentic_e2e

package agentic_test

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/gate/gate-cli/tests/contracts"
)

// The fixture uses integer 10^-8 units and zero fees. It checks CLI decimal
// preservation and reservations, not live venue rounding/fee/matching rules.
const spotUnit int64 = 100000000

func spotUnits(raw string) (int64, error) {
	r, ok := new(big.Rat).SetString(raw)
	if !ok {
		return 0, fmt.Errorf("invalid decimal")
	}
	r.Mul(r, new(big.Rat).SetInt64(spotUnit))
	if !r.IsInt() || !r.Num().IsInt64() || r.Sign() < 0 {
		return 0, fmt.Errorf("amount outside fixture precision/range")
	}
	return r.Num().Int64(), nil
}
func spotDecimal(n int64) string { return fmt.Sprintf("%d.%08d", n/spotUnit, n%spotUnit) }

type spotLedger struct {
	t                   *testing.T
	auth                string
	mu                  sync.Mutex
	base, quote, locked int64
	next                int64
	orders              map[string]map[string]string
	trades              []map[string]string
}

func (l *spotLedger) serve(w http.ResponseWriter, r *http.Request) {
	l.mu.Lock()
	defer l.mu.Unlock()
	raw, err := io.ReadAll(io.LimitReader(r.Body, 8193))
	if err != nil || len(raw) > 8192 {
		l.t.Error("invalid scenario request")
		http.Error(w, "bad body", 400)
		return
	}
	if err := contracts.CheckAuth(r, raw, l.auth); err != nil {
		l.t.Error(err)
	}
	w.Header().Set("Content-Type", "application/json")
	respond := func(v any) {
		if err := json.NewEncoder(w).Encode(v); err != nil {
			l.t.Error(err)
		}
	}
	reject := func(label string) {
		w.WriteHeader(400)
		respond(map[string]string{"label": label, "message": "synthetic spot rejection"})
	}
	switch {
	case r.Method == "GET" && r.URL.Path == "/api/v4/spot/accounts":
		out := []map[string]string{{"currency": "SYNTH", "available": spotDecimal(l.base), "locked": "0.00000000"}, {"currency": "USDT", "available": spotDecimal(l.quote), "locked": spotDecimal(l.locked)}}
		if c := r.URL.Query().Get("currency"); c != "" {
			if c == "USDT" {
				out = out[1:]
			} else {
				out = out[:1]
			}
		}
		respond(out)
	case r.Method == "GET" && r.URL.Path == "/api/v4/spot/my_trades":
		respond(l.trades)
	case r.Method == "GET" && r.URL.Path == "/api/v4/spot/orders":
		out := []map[string]string{}
		for _, o := range l.orders {
			if o["status"] == r.URL.Query().Get("status") {
				out = append(out, o)
			}
		}
		respond(out)
	case r.Method == "POST" && r.URL.Path == "/api/v4/spot/orders":
		var input map[string]json.RawMessage
		if json.Unmarshal(raw, &input) != nil {
			reject("INVALID_ORDER")
			return
		}
		// Only trading scalars belong to this small ledger. SDK protection
		// objects/nulls must not be converted into strings in the response.
		o := map[string]string{}
		for _, key := range []string{"currency_pair", "side", "type", "amount", "price", "text", "time_in_force"} {
			if value, ok := input[key]; ok {
				var text string
				if err := json.Unmarshal(value, &text); err != nil {
					reject("INVALID_ORDER")
					return
				}
				o[key] = text
			}
		}
		if o["currency_pair"] != "SYNTH_USDT" {
			reject("INVALID_ORDER")
			return
		}
		amount, err := spotUnits(o["amount"])
		if err != nil || amount == 0 {
			reject("INVALID_AMOUNT")
			return
		}
		if o["side"] != "buy" && o["side"] != "sell" {
			reject("INVALID_SIDE")
			return
		}
		if o["type"] == "market" {
			if o["side"] == "buy" {
				if amount > l.quote || amount%10 != 0 {
					reject("INSUFFICIENT_QUOTE")
					return
				}
				l.quote -= amount
				l.base += amount / 10
			} else {
				if amount > l.base {
					reject("INSUFFICIENT_BASE")
					return
				}
				l.base -= amount
				l.quote += amount * 10
			}
			o["status"], o["left"] = "closed", "0"
		} else {
			price, err := spotUnits(o["price"])
			if err != nil || price <= 0 || amount > 10*spotUnit || price > 50*spotUnit {
				reject("INVALID_PRICE")
				return
			}
			// Bounded fixture amounts make this exact multiplication fit int64.
			reserve := amount * price / spotUnit
			if o["side"] != "buy" || reserve > l.quote {
				reject("INSUFFICIENT_QUOTE")
				return
			}
			l.quote -= reserve
			l.locked += reserve
			o["status"], o["left"] = "open", o["amount"]
		}
		o["id"] = strconv.FormatInt(l.next, 10)
		l.next++
		l.orders[o["id"]] = o
		if o["status"] == "closed" {
			l.trades = append(l.trades, map[string]string{"id": o["id"], "order_id": o["id"], "currency_pair": o["currency_pair"], "side": o["side"], "amount": o["amount"], "price": "10.00000000"})
		}
		respond(o)
	case strings.HasPrefix(r.URL.Path, "/api/v4/spot/orders/"):
		id := strings.TrimPrefix(r.URL.Path, "/api/v4/spot/orders/")
		o, ok := l.orders[id]
		if !ok {
			reject("ORDER_NOT_FOUND")
			return
		}
		switch r.Method {
		case "GET":
			respond(o)
		case "DELETE":
			if o["status"] != "open" {
				reject("ORDER_FINISHED")
				return
			}
			amount, _ := spotUnits(o["amount"])
			price, _ := spotUnits(o["price"])
			reserve := amount * price / spotUnit
			l.locked -= reserve
			l.quote += reserve
			o["status"] = "cancelled"
			respond(o)
		case "PATCH":
			if o["status"] != "open" {
				reject("ORDER_FINISHED")
				return
			}
			var update map[string]any
			if json.Unmarshal(raw, &update) != nil {
				reject("INVALID_AMEND")
				return
			}
			oldAmount, _ := spotUnits(o["amount"])
			oldPrice, _ := spotUnits(o["price"])
			amount, price := oldAmount, oldPrice
			if v, ok := update["amount"].(string); ok {
				amount, err = spotUnits(v)
				if err != nil {
					reject("INVALID_AMOUNT")
					return
				}
			}
			if v, ok := update["price"].(string); ok {
				price, err = spotUnits(v)
				if err != nil {
					reject("INVALID_PRICE")
					return
				}
			}
			if amount <= 0 || price <= 0 || amount > 10*spotUnit || price > 50*spotUnit {
				reject("INVALID_AMEND")
				return
			}
			oldReserve, newReserve := oldAmount*oldPrice/spotUnit, amount*price/spotUnit
			if newReserve-oldReserve > l.quote {
				reject("INSUFFICIENT_QUOTE")
				return
			}
			l.quote += oldReserve - newReserve
			l.locked += newReserve - oldReserve
			o["amount"], o["price"] = spotDecimal(amount), spotDecimal(price)
			respond(o)
		default:
			l.t.Errorf("unexpected method %s", r.Method)
			reject("UNEXPECTED_METHOD")
		}
	default:
		l.t.Errorf("unexpected spot lifecycle route: %s %s", r.Method, r.URL)
		reject("UNEXPECTED_ROUTE")
	}
}

func TestSpotTradingLifecycle(t *testing.T) {
	for _, auth := range []string{"hmac", "oauth"} {
		t.Run(auth, func(t *testing.T) {
			ledger := &spotLedger{t: t, auth: auth, base: spotUnit, quote: 100 * spotUnit, next: 9007199254740993, orders: map[string]map[string]string{}}
			peer := httptest.NewServer(http.HandlerFunc(ledger.serve))
			t.Cleanup(peer.Close)
			home := futuresConfirmedHome(t, peer.URL, auth)
			invoke := func(args ...string) []byte {
				t.Helper()
				out, stderr, err := futuresConfirmedInvoke(t, cliBinary, home, peer.URL, auth, append([]string{"--format", "json", "cex", "spot"}, args...)...)
				if err != nil || len(stderr) != 0 {
					t.Fatalf("%v: %v %s %s", args, err, out, stderr)
				}
				return out
			}
			balances := func(base, quote, locked string) {
				t.Helper()
				var rows []struct{ Currency, Available, Locked string }
				if err := json.Unmarshal(invoke("account", "list"), &rows); err != nil {
					t.Fatal(err)
				}
				if len(rows) != 2 || rows[0].Available != base || rows[1].Available != quote || rows[1].Locked != locked {
					t.Fatalf("balances=%+v want %s/%s locked=%s", rows, base, quote, locked)
				}
			}
			balances("1.00000000", "100.00000000", "0.00000000")
			pending := futuresObject(t, invoke("order", "buy", "--pair", "SYNTH_USDT", "--amount", "2.00000001", "--price", "10.00000000"))
			futuresField(t, pending, "id", "9007199254740993")
			balances("1.00000000", "79.99999990", "20.00000010")
			invoke("order", "amend", "--pair", "SYNTH_USDT", "--id", "9007199254740993", "--amount", "2", "--price", "9.00000001")
			balances("1.00000000", "81.99999998", "18.00000002")
			futuresField(t, futuresObject(t, invoke("order", "get", "--pair", "SYNTH_USDT", "--id", "9007199254740993")), "price", "9.00000001")
			invoke("order", "cancel", "--pair", "SYNTH_USDT", "--id", "9007199254740993")
			balances("1.00000000", "100.00000000", "0.00000000")
			invoke("order", "buy", "--pair", "SYNTH_USDT", "--quote", "5.00000010")
			balances("1.50000001", "94.99999990", "0.00000000")
			invoke("order", "sell", "--pair", "SYNTH_USDT", "--amount", "0.50000001")
			balances("1.00000000", "100.00000000", "0.00000000")
			if !contracts.EqualJSON(invoke("order", "list", "--pair", "SYNTH_USDT", "--status", "open"), []byte(`[]`)) {
				t.Fatal("pending order leaked")
			}
			var trades []json.RawMessage
			if err := json.Unmarshal(invoke("order", "my-trades", "--pair", "SYNTH_USDT"), &trades); err != nil || len(trades) != 2 {
				t.Fatalf("fills=%d want 2: %v", len(trades), err)
			}
			// A rejected sell must leave balances and fill count unchanged.
			out, stderr, err := futuresConfirmedInvoke(t, cliBinary, home, peer.URL, auth, "--format", "json", "cex", "spot", "order", "sell", "--pair", "SYNTH_USDT", "--amount", "2")
			if err == nil || len(out) != 0 || !strings.Contains(string(stderr), "INSUFFICIENT_BASE") {
				t.Fatalf("oversell accepted: %v %s %s", err, out, stderr)
			}
			balances("1.00000000", "100.00000000", "0.00000000")
		})
	}
}
