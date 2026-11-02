//go:build agentic_e2e

package agentic_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// cfdCases 固定现有18个叶子契约，方向/价格类型使用公共SDK的真实定义。
func cfdCases() []commandCase {
	return []commandCase{
		{"categories", "GET", "/tradfi/symbols/categories", []string{"market", "categories"}, true},
		{"symbols", "GET", "/tradfi/symbols", []string{"market", "symbols"}, true},
		{"symbol", "GET", "/tradfi/symbols/detail", []string{"market", "symbol", "--symbol", "SYNTH"}, false},
		{"ticker", "GET", "/tradfi/symbols/SYNTH/tickers", []string{"market", "ticker", "--symbol", "SYNTH"}, true},
		{"kline", "GET", "/tradfi/symbols/SYNTH/klines", []string{"market", "kline", "--symbol", "SYNTH"}, true},
		{"info", "GET", "/tradfi/users/mt5-account", []string{"account", "info"}, false},
		{"assets", "GET", "/tradfi/users/assets", []string{"account", "assets"}, false},
		{"transactions", "GET", "/tradfi/transactions", []string{"account", "transactions"}, false},
		{"transfer", "POST", "/tradfi/transactions", []string{"account", "transfer", "--change", "0.00000001", "--type", "deposit"}, false},
		{"orders", "GET", "/tradfi/orders", []string{"order", "list"}, false},
		{"history", "GET", "/tradfi/orders/history", []string{"order", "history"}, false},
		{"create", "POST", "/tradfi/orders", []string{"order", "create", "--symbol", "SYNTH", "--side", "2", "--volume", "1", "--price-type", "market"}, false},
		{"update", "PUT", "/tradfi/orders/2147483648", []string{"order", "update", "--id", "2147483648", "--price", "10.00000001"}, false},
		{"cancel", "DELETE", "/tradfi/orders/9007199254740993", []string{"order", "cancel", "--id", "9007199254740993"}, false},
		{"positions", "GET", "/tradfi/positions", []string{"position", "list"}, false},
		{"position-history", "GET", "/tradfi/positions/history", []string{"position", "history"}, false},
		{"position-update", "PUT", "/tradfi/positions/2147483648", []string{"position", "update", "--id", "2147483648", "--tp", "10"}, false},
		{"close", "POST", "/tradfi/positions/9007199254740993/close", []string{"position", "close", "--id", "9007199254740993", "--close-type", "2"}, false},
	}
}
func TestCFDCommandWire(t *testing.T) {
	for _, tc := range cfdCases() {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != tc.method || r.URL.Path != "/api/v4"+tc.path {
					t.Errorf("wire %s %s", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"code":0,"data":{"list":[]}}`))
			}))
			defer server.Close()
			args := append([]string{"--format", "json", "cex", "tradfi"}, tc.args...)
			_, stderr, err := runCLI(t, server.URL, true, args...)
			if err != nil {
				t.Fatalf("CLI: %v %s", err, stderr)
			}
			if calls.Load() != 1 {
				t.Errorf("requests=%d want1", calls.Load())
			}
		})
	}
}
func TestCFDErrorIsFailure(t *testing.T) {
	for _, status := range []int{401, 403, 429, 503} {
		for _, tc := range cfdCases() {
			t.Run(fmt.Sprintf("%s/%d", tc.name, status), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(status)
					_, _ = w.Write([]byte(`{"label":"SYNTHETIC_FAILURE","message":"synthetic"}`))
				}))
				defer server.Close()
				args := append([]string{"--format", "json", "cex", "tradfi"}, tc.args...)
				stdout, stderr, err := runCLI(t, server.URL, true, args...)
				if err == nil || len(strings.TrimSpace(string(stdout))) != 0 {
					t.Errorf("failure status=%d exit=%v stdout=%s stderr=%s", status, err, stdout, stderr)
				}
			})
		}
	}
}
