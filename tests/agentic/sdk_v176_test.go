//go:build agentic_e2e

package agentic_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// Real CLI subprocess -> production transport -> published SDK -> mock backend.
// Error status/content must reach stderr and a nonzero exit, with no replay.
func TestSDK176CLIPriceOrder(t *testing.T) {
	for _, tc := range []struct {
		name    string
		status  int
		body    string
		failure bool
	}{
		{"normal", 200, `{"id":9007199254740993,"initial":{"contract":"SYNTH_USDT","size":1,"price":"0"},"trigger":{"price":"1","rule":1}}`, false},
		{"205 not found", 205, `{"code":"1034","label":"AUTO_ORDER_NOT_FOUND","message":"no orderID match"}`, true},
		{"205 empty", 205, ``, true}, {"205 null", 205, `null`, true}, {"205 malformed", 205, `not-json`, true},
		{"205 private body", 205, `{"private_debug":"sdk176-private-canary"}`, true},
		{"400", 400, `{"label":"INVALID_PARAM_VALUE","message":"invalid order"}`, true},
		{"503", 503, `{"label":"INTERNAL","message":"unavailable"}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != "GET" || r.URL.Path != "/api/v4/futures/usdt/price_orders/9007199254740993" {
					t.Errorf("route: %s %s", r.Method, r.URL.Path)
				}
				if r.Header.Get("KEY") != "synthetic-key" || r.Header.Get("SIGN") == "" {
					t.Error("production HMAC signing bypassed")
				}
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Gate-Trace-Id", "sdk176-trace")
				w.WriteHeader(tc.status)
				if _, err := io.WriteString(w, tc.body); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(peer.Close)
			out, stderr, err := runCLI(t, peer.URL, true, "--format", "json", "cex", "futures", "price-trigger", "get", "--id", "9007199254740993")
			if calls.Load() != 1 {
				t.Fatalf("request count=%d", calls.Load())
			}
			if !tc.failure {
				if err != nil || len(stderr) != 0 || !bytes.Contains(out, []byte("9007199254740993")) {
					t.Fatalf("normal result changed: %v %s %s", err, out, stderr)
				}
				return
			}
			if err == nil || len(out) != 0 {
				t.Fatalf("HTTP%d became success: %v stdout=%s stderr=%s", tc.status, err, out, stderr)
			}
			if bytes.Contains(stderr, []byte("sdk176-private-canary")) {
				t.Fatalf("private backend response leaked: %s", stderr)
			}
			if !json.Valid(stderr) {
				t.Fatalf("not a structured CLI error: %s", stderr)
			}
			if tc.name == "205 not found" && (!bytes.Contains(stderr, []byte("AUTO_ORDER_NOT_FOUND")) || !bytes.Contains(stderr, []byte("no orderID match")) || !bytes.Contains(stderr, []byte("sdk176-trace"))) {
				t.Fatalf("business error context lost: %s", stderr)
			}
		})
	}
}

// The normal SDK envelope, empty list, precision, filters and signing all survive
// the CLI path; the test does not unwrap data on behalf of production code.
func TestSDK176CLITrailList(t *testing.T) {
	for _, orders := range []string{`[]`, `[{"id":"9007199254740993","contract":"SYNTH_USDT","amount":"1","status":"open"}]`} {
		t.Run(orders, func(t *testing.T) {
			var calls atomic.Int32
			peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != "GET" || r.URL.Path != "/api/v4/futures/usdt/autoorder/v1/trail/list" {
					t.Errorf("route: %s %s", r.Method, r.URL.Path)
				}
				if r.URL.Query().Get("contract") != "SYNTH_USDT" || r.URL.Query().Get("page_size") != "2" {
					t.Errorf("filters changed: %s", r.URL.RawQuery)
				}
				if r.Header.Get("KEY") != "synthetic-key" || r.Header.Get("SIGN") == "" {
					t.Error("signing bypassed")
				}
				w.Header().Set("Content-Type", "application/json")
				if _, err := fmt.Fprintf(w, `{"code":0,"message":"ok","data":{"orders":%s},"timestamp":1793582374966743}`, orders); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(peer.Close)
			out, stderr, err := runCLI(t, peer.URL, true, "--format", "json", "cex", "futures", "trail", "list", "--contract", "SYNTH_USDT", "--limit", "2")
			if err != nil || len(stderr) != 0 || calls.Load() != 1 {
				t.Fatalf("normal list failed: %v %s %s calls=%d", err, out, stderr, calls.Load())
			}
			var value struct {
				Message string `json:"message"`
				Data    *struct {
					Orders []struct {
						ID                       json.Number `json:"id"`
						Contract, Amount, Status string
					} `json:"orders"`
				} `json:"data"`
				Timestamp json.Number `json:"timestamp"`
			}
			if err := json.Unmarshal(out, &value); err != nil {
				t.Fatal(err)
			}
			if value.Data == nil || value.Message != "ok" || value.Timestamp.String() != "1793582374966743" {
				t.Fatalf("SDK envelope lost: %s", out)
			}
			if orders != "[]" && (len(value.Data.Orders) != 1 || value.Data.Orders[0].ID.String() != "9007199254740993" || value.Data.Orders[0].Amount != "1") {
				t.Fatalf("orders lost: %s", out)
			}
			if orders == "[]" && len(value.Data.Orders) != 0 {
				t.Fatalf("empty list changed: %s", out)
			}
		})
	}
}

// An unrelated operation still reports its original decoding error; the new
// SDK176 detail-error presentation must not change all HTTP205 responses.
func TestSDK176CLI205Scope(t *testing.T) {
	peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v4/futures/usdt/contracts" {
			t.Errorf("unexpected control route: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(205)
		if _, err := io.WriteString(w, "not-json"); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(peer.Close)
	out, stderr, err := runCLI(t, peer.URL, false, "--format", "json", "cex", "futures", "market", "contracts")
	if err == nil || len(out) != 0 || !bytes.Contains(stderr, []byte("INVALID_RESPONSE")) {
		t.Fatalf("unrelated205 decoding changed: %v %s %s", err, out, stderr)
	}
}
