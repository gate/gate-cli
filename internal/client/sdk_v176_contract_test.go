package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	gateapi "github.com/gate/gateapi-go/v7"
)

// Exercise the published SDK before any CLI/MCP adapter can mask its decoder.
// Only the remote HTTP peer is mocked; these assertions must turn red on v173.
func TestSDK176PriceOrderDetail(t *testing.T) {
	for _, tc := range []struct {
		name    string
		status  int
		body    string
		failure bool
	}{
		{"normal", 200, `{"id":9007199254740993,"initial":{"contract":"SYNTH_USDT","size":1,"price":"0"},"trigger":{"price":"1","rule":1}}`, false},
		{"not found", 205, `{"code":"1034","label":"AUTO_ORDER_NOT_FOUND","message":"no orderID match"}`, true},
		{"empty error", 205, ``, true}, {"null error", 205, `null`, true}, {"unknown error", 205, `{}`, true}, {"broken error", 205, `not-json`, true},
		{"bad request", 400, `{"label":"INVALID_PARAM_VALUE","message":"invalid order"}`, true},
		{"unavailable", 503, `{"label":"INTERNAL","message":"unavailable"}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method != "GET" || r.URL.Path != "/api/v4/futures/usdt/price_orders/9007199254740993" {
					t.Errorf("unexpected route: %s %s", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				if _, err := fmt.Fprint(w, tc.body); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(peer.Close)
			cfg := gateapi.NewConfiguration()
			cfg.BasePath = peer.URL + "/api/v4"
			cfg.HTTPClient = &http.Client{Timeout: 2 * time.Second}
			value, response, err := gateapi.NewAPIClient(cfg).FuturesApi.GetPriceTriggeredOrder(t.Context(), "usdt", 9007199254740993)
			if response == nil || response.StatusCode != tc.status || calls.Load() != 1 {
				t.Fatalf("lost status or retried: response=%v calls=%d", response, calls.Load())
			}
			if tc.failure {
				if err == nil {
					t.Fatalf("business failure returned successful order: %#v", value)
				}
				return
			}
			if err != nil || value.Id != 9007199254740993 {
				t.Fatalf("normal order changed: %#v %v", value, err)
			}
		})
	}
}

// The SDK must retain the envelope and exact large identifiers, including an
// explicitly empty array, without a transport-side response reconstruction.
func TestSDK176TrailEnvelope(t *testing.T) {
	for _, orders := range []string{`[]`, `[{"id":"9007199254740993","contract":"SYNTH_USDT","amount":"1","status":"open"}]`} {
		t.Run(orders, func(t *testing.T) {
			peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || r.URL.Path != "/api/v4/futures/usdt/autoorder/v1/trail/list" {
					t.Errorf("wrong route: %s %s", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				if _, err := fmt.Fprintf(w, `{"code":0,"message":"ok","data":{"orders":%s},"timestamp":1793582374966743,"unknown":true}`, orders); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(peer.Close)
			cfg := gateapi.NewConfiguration()
			cfg.BasePath = peer.URL + "/api/v4"
			cfg.HTTPClient = &http.Client{Timeout: 2 * time.Second}
			value, _, err := gateapi.NewAPIClient(cfg).FuturesApi.GetTrailOrders(t.Context(), "usdt", nil)
			if err != nil {
				t.Fatal(err)
			}
			model := reflect.ValueOf(value)
			code := model.FieldByName("Code")
			data := model.FieldByName("Data")
			timestamp := model.FieldByName("Timestamp")
			if !code.IsValid() || !data.IsValid() || !timestamp.IsValid() {
				t.Fatalf("SDK discarded Trail envelope: %#v", value)
			}
			if code.Int() != 0 || timestamp.Int() != 1793582374966743 {
				t.Fatal("lost envelope metadata")
			}
			list := data.FieldByName("Orders")
			if !list.IsValid() || list.IsNil() {
				t.Fatal("orders array lost")
			}
			expected := 0
			if orders != "[]" {
				expected = 1
			}
			if list.Len() != expected {
				t.Fatalf("orders=%d want %d", list.Len(), expected)
			}
			raw, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			if expected == 1 && !bytes.Contains(raw, []byte("9007199254740993")) {
				t.Fatalf("large order id lost: %s", raw)
			}
		})
	}
}

// HTTP 205 is an error for the detail operation alone, not every SDK endpoint.
func TestSDK176Unmarked205(t *testing.T) {
	peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(205)
		if _, err := fmt.Fprint(w, "[]"); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(peer.Close)
	cfg := gateapi.NewConfiguration()
	cfg.BasePath = peer.URL + "/api/v4"
	cfg.HTTPClient = &http.Client{Timeout: 2 * time.Second}
	value, response, err := gateapi.NewAPIClient(cfg).FuturesApi.ListFuturesContracts(t.Context(), "usdt", nil)
	if err != nil || response == nil || response.StatusCode != 205 || len(value) != 0 {
		t.Fatalf("unmarked endpoint changed: %#v %v %v", value, response, err)
	}
}
