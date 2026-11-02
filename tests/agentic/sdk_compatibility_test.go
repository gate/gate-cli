//go:build agentic_e2e

package agentic_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// TestSDK161Compatibility 覆盖升级时改变的三类wire契约，路径ID与body ID不得混用。
func TestSDK161Compatibility(t *testing.T) {
	for _, tc := range []struct {
		name, path, response string
		args                 []string
	}{
		{"price-trigger", "/api/v4/futures/usdt/price_orders/amend/pending-2147483648", `{"id":9007199254740993,"id_string":"9007199254740993"}`, []string{"futures", "price-trigger", "update", "--id", "pending-2147483648", "--body-order-id", "9007199254740993", "--price", "10"}},
		{"saved-address", "/api/v4/wallet/saved_address", `[]`, []string{"wallet", "deposit", "saved", "--currency", "USDT", "--chain", "ETH", "--limit", "20"}},
		{"adl", "/api/v4/crossex/adl_rank", `{"symbol":"SYNTH","crossex_adl_rank":"1","exchange_adl_rank":"2"}`, []string{"cross-ex", "position", "adl-rank"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Path != tc.path {
					t.Errorf("path=%s want%s", r.URL.Path, tc.path)
				}
				if tc.name == "price-trigger" {
					body, err := io.ReadAll(io.LimitReader(r.Body, 8192))
					if err != nil {
						t.Error(err)
					}
					var fields map[string]json.RawMessage
					if json.Unmarshal(body, &fields) != nil || string(fields["order_id"]) != "9007199254740993" {
						t.Error("body order ID truncated or replaced with path ID")
					}
				}
				if tc.name == "saved-address" && (r.URL.Query().Get("currency") != "USDT" || r.URL.Query().Get("limit") != "20") {
					t.Error("saved-address opts lost")
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tc.response))
			}))
			defer server.Close()
			args := append([]string{"--format", "json", "cex"}, tc.args...)
			stdout, stderr, err := runCLI(t, server.URL, true, args...)
			if tc.name == "adl" {
				if err == nil || calls.Load() != 0 || !strings.Contains(string(stderr), "unknown command") {
					t.Fatalf("retired command reached network or succeeded: %v %s", err, stderr)
				}
				return
			}
			if err != nil {
				t.Fatalf("CLI: %v %s", err, stderr)
			}
			// Keep the original wire/large-ID assertions and also prove a real
			// success identity survived, rather than treating an empty DTO as proof.
			if tc.name == "price-trigger" && !strings.Contains(string(stdout), `"id_string":"9007199254740993"`) {
				t.Fatalf("successful trigger identity lost: %s", stdout)
			}
			if calls.Load() != 1 {
				t.Errorf("requests=%d want1", calls.Load())
			}
		})
	}
}
