//go:build agentic_e2e

package agentic_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gate/gate-cli/tests/contracts"
)

// fundsLedger is a zero-fee/zero-interest business fixture. It models balances
// and liabilities in integer 10^-8 USDT units across wallet, futures, earn and
// unified loan commands. The invariant is assets - debt == initial equity.
type fundsLedger struct {
	t                         *testing.T
	auth                      string
	mu                        sync.Mutex
	spot, futures, earn, debt int64
	writes                    int
}

func (l *fundsLedger) serve(w http.ResponseWriter, r *http.Request) {
	l.mu.Lock()
	defer l.mu.Unlock()
	raw, err := io.ReadAll(io.LimitReader(r.Body, 8193))
	if err != nil || len(raw) > 8192 {
		l.t.Error("invalid bounded money request")
		http.Error(w, "invalid body", 400)
		return
	}
	if err := contracts.CheckAuth(r, raw, l.auth); err != nil {
		l.t.Error(err)
	}
	w.Header().Set("Content-Type", "application/json")
	respond := func(value any) {
		if err := json.NewEncoder(w).Encode(value); err != nil {
			l.t.Error(err)
		}
	}
	reject := func() {
		w.WriteHeader(400)
		respond(map[string]string{"label": "INSUFFICIENT_FUNDS", "message": "synthetic balance/loan constraint"})
	}
	if r.Method == "GET" {
		switch r.URL.Path {
		case "/api/v4/spot/accounts":
			respond([]map[string]string{{"currency": "USDT", "available": spotDecimal(l.spot), "locked": "0"}})
		case "/api/v4/futures/usdt/accounts":
			respond(map[string]any{"currency": "USDT", "total": spotDecimal(l.futures), "available": spotDecimal(l.futures), "in_dual_mode": false})
		case "/api/v4/earn/uni/lends":
			respond([]map[string]string{{"currency": "USDT", "amount": spotDecimal(l.earn), "current_amount": spotDecimal(l.earn), "lent_amount": spotDecimal(l.earn), "min_rate": "0.0001"}})
		case "/api/v4/unified/loans":
			respond([]map[string]string{{"currency": "USDT", "amount": spotDecimal(l.debt), "type": "platform"}})
		default:
			l.t.Errorf("unexpected funds read %s", r.URL)
			http.Error(w, "unexpected route", 404)
		}
		return
	}
	if r.Method != "POST" {
		l.t.Errorf("unexpected funds method %s", r.Method)
		reject()
		return
	}
	var body struct{ Currency, Amount, From, To, Type, Settle string }
	if json.Unmarshal(raw, &body) != nil || body.Currency != "USDT" {
		reject()
		return
	}
	amount, err := spotUnits(body.Amount)
	if err != nil || amount <= 0 || amount > 1000*spotUnit {
		reject()
		return
	}
	switch r.URL.Path {
	case "/api/v4/wallet/transfers":
		if body.Settle != "usdt" {
			l.t.Error("futures transfer lost its settlement")
		}
		if body.From == "spot" && body.To == "futures" && amount <= l.spot {
			l.spot -= amount
			l.futures += amount
		} else if body.From == "futures" && body.To == "spot" && amount <= l.futures {
			l.futures -= amount
			l.spot += amount
		} else {
			reject()
			return
		}
		l.writes++
		respond(map[string]int64{"tx_id": 7})
	case "/api/v4/earn/uni/lends":
		if body.Type == "lend" && amount <= l.spot {
			l.spot -= amount
			l.earn += amount
		} else if body.Type == "redeem" && amount <= l.earn {
			l.earn -= amount
			l.spot += amount
		} else {
			reject()
			return
		}
		l.writes++
		respond(map[string]any{})
	case "/api/v4/unified/loans":
		if body.Type == "borrow" {
			l.debt += amount
			l.spot += amount
		} else if body.Type == "repay" && amount <= l.debt && amount <= l.spot {
			l.spot -= amount
			l.debt -= amount
		} else {
			reject()
			return
		}
		l.writes++
		respond(map[string]int64{"tran_id": 7})
	default:
		l.t.Errorf("unexpected funds write %s", r.URL)
		reject()
		return
	}
	if l.spot < 0 || l.futures < 0 || l.earn < 0 || l.debt < 0 || l.spot+l.futures+l.earn-l.debt != 100*spotUnit {
		l.t.Error("fixture violated equity conservation")
	}
}

func TestCrossProductFundsLifecycle(t *testing.T) {
	for _, auth := range []string{"hmac", "oauth"} {
		t.Run(auth, func(t *testing.T) {
			ledger := &fundsLedger{t: t, auth: auth, spot: 100 * spotUnit}
			peer := httptest.NewServer(http.HandlerFunc(ledger.serve))
			t.Cleanup(peer.Close)
			home := cexContractHome(t, peer.URL, auth)
			invoke := func(args ...string) []byte {
				t.Helper()
				out, stderr, err := futuresConfirmedInvoke(t, cliBinary, home, peer.URL, auth, append([]string{"--format", "json", "cex"}, args...)...)
				if err != nil || len(stderr) != 0 {
					t.Fatalf("%v: %v %s %s", args, err, out, stderr)
				}
				return out
			}
			read := func(args []string, key, want string) {
				t.Helper()
				raw := invoke(args...)
				var rows []map[string]json.RawMessage
				if json.Unmarshal(raw, &rows) == nil {
					if len(rows) != 1 {
						t.Fatalf("balance rows=%d", len(rows))
					}
					futuresField(t, rows[0], key, want)
				} else {
					futuresField(t, futuresObject(t, raw), key, want)
				}
			}
			spot := []string{"spot", "account", "list"}
			future := []string{"futures", "account", "get"}
			earn := []string{"earn", "uni", "lends"}
			loans := []string{"unified", "loan", "list"}
			read(spot, "available", "100.00000000")
			invoke("wallet", "transfer", "create", "--currency", "USDT", "--from", "spot", "--to", "futures", "--amount", "2.00000001")
			read(spot, "available", "97.99999999")
			read(future, "total", "2.00000001")
			invoke("wallet", "transfer", "create", "--currency", "USDT", "--from", "futures", "--to", "spot", "--amount", "2.00000001")
			read(spot, "available", "100.00000000")
			read(future, "total", "0.00000000")
			invoke("unified", "loan", "create", "--currency", "USDT", "--amount", "3.00000001", "--type", "borrow")
			read(loans, "amount", "3.00000001")
			read(spot, "available", "103.00000001")
			lend := func(kind, amount string) {
				t.Helper()
				invoke("earn", "uni", "lend", "--json", `{"currency":"USDT","amount":"`+amount+`","type":"`+kind+`","min_rate":"0.0001"}`)
			}
			lend("lend", "10.00000001")
			read(earn, "amount", "10.00000001")
			read(spot, "available", "93.00000000")
			lend("redeem", "4.00000001")
			read(earn, "amount", "6.00000000")
			read(spot, "available", "97.00000001")
			invoke("unified", "loan", "create", "--currency", "USDT", "--amount", "3.00000001", "--type", "repay")
			read(loans, "amount", "0.00000000")
			read(spot, "available", "94.00000000")
			lend("redeem", "6")
			read(earn, "amount", "0.00000000")
			read(spot, "available", "100.00000000")
			for _, args := range [][]string{
				{"wallet", "transfer", "create", "--currency", "USDT", "--from", "spot", "--to", "futures", "--amount", "101"},
				{"unified", "loan", "create", "--currency", "USDT", "--amount", "1", "--type", "repay"},
				{"earn", "uni", "lend", "--json", `{"currency":"USDT","amount":"1","type":"redeem"}`},
			} {
				out, stderr, err := futuresConfirmedInvoke(t, cliBinary, home, peer.URL, auth, append([]string{"--format", "json", "cex"}, args...)...)
				if err == nil || len(out) != 0 || !strings.Contains(string(stderr), "INSUFFICIENT_FUNDS") {
					t.Fatalf("invalid funds operation accepted: %v %s %s", err, out, stderr)
				}
			}
			read(spot, "available", "100.00000000")
			ledger.mu.Lock()
			defer ledger.mu.Unlock()
			if ledger.spot != 100*spotUnit || ledger.futures != 0 || ledger.earn != 0 || ledger.debt != 0 || ledger.writes != 7 {
				t.Fatalf("final funds ledger: %+v", ledger)
			}
		})
	}
}
