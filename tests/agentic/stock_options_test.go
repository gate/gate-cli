//go:build agentic_e2e

package agentic_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gate/gate-cli/internal/output"
)

// TestStockOptionalWire 对每个可选参数检查实际query；false、0与省略具有不同语义。
func TestStockOptionalWire(t *testing.T) {
	cases := []struct {
		args  []string
		query url.Values
	}{
		{[]string{"market", "symbols", "--symbols", "SYNTH,OTHER", "--exchange", "hk", "--page", "2", "--page-size", "500", "--with-desc-i18n=false"}, url.Values{"symbols": {"SYNTH,OTHER"}, "exchange": {"hk"}, "page": {"2"}, "page_size": {"500"}, "with_desc_i18n": {"false"}}},
		{[]string{"market", "symbol", "--symbols", "SYNTH", "--exchange", "us"}, url.Values{"symbols": {"SYNTH"}, "exchange": {"us"}, "page": {"1"}, "page_size": {"10"}}},
		{[]string{"order", "history", "--symbol", "SYNTH", "--order-ids", "9007199254740993", "--begin", "0", "--end", "2147483647", "--side", "1"}, url.Values{"symbol": {"SYNTH"}, "order_ids": {"9007199254740993"}, "begin_time": {"0"}, "end_time": {"2147483647"}, "side": {"1"}, "page": {"1"}, "page_size": {"10"}}},
		{[]string{"position", "list", "--symbol", "SYNTH", "--exchange", "jp", "--pnl-calc-type", "1", "--pnl-calc-price", "2"}, url.Values{"symbol": {"SYNTH"}, "exchange": {"jp"}, "pnl_calc_type": {"1"}, "pnl_calc_price": {"2"}}},
		{[]string{"account", "assets", "--pnl-calc-type", "2", "--pnl-calc-price", "1"}, url.Values{"pnl_calc_type": {"2"}, "pnl_calc_price": {"1"}}},
		{[]string{"account", "transactions", "--begin", "0", "--end", "9007199254740993", "--type", "deposit"}, url.Values{"begin_time": {"0"}, "end_time": {"9007199254740993"}, "type": {"deposit"}, "page": {"1"}, "page_size": {"10"}}},
		{[]string{"account", "transactions", "--ref-id", "synthetic-ref"}, url.Values{"ref_id": {"synthetic-ref"}}},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if !reflect.DeepEqual(r.URL.Query(), tc.query) {
					t.Errorf("query=%v want%v", r.URL.Query(), tc.query)
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"list": []interface{}{}}}); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			args := append([]string{"cex", "stock"}, tc.args...)
			if _, stderr, err := runCLI(t, server.URL, true, args...); err != nil {
				t.Fatalf("CLI %v %s", err, stderr)
			}
			if calls.Load() != 1 {
				t.Fatal("query missing or repeated")
			}
		})
	}
}

// TestStockLostWriteResponse 后端已受理后断开连接，CLI必须提示核实结果，不能补发。
func TestStockLostWriteResponse(t *testing.T) {
	for _, tc := range stockCases() {
		if tc.method == "GET" {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				connection, _, err := w.(http.Hijacker).Hijack()
				if err != nil {
					t.Error(err)
					return
				}
				if err := connection.Close(); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			args := append([]string{"cex", "stock"}, tc.args...)
			stdout, stderr, err := runCLI(t, server.URL, true, args...)
			if err == nil || len(bytes.TrimSpace(stdout)) > 0 || !strings.Contains(string(stderr), "outcome unknown") {
				t.Fatalf("lost response not reported: %v %s", err, stderr)
			}
			if calls.Load() != 1 {
				t.Fatal("write resent after lost response")
			}
		})
	}
}

// TestStockOutputPrecision 真实SDK DTO和table都不能把大ID/十进制经过float64。
func TestStockOutputPrecision(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"timestamp":1,"data":{"list":[{"order_id":"9007199254740993","time_setup":9007199254740993,"volume":"0.00000001","price":"10.00000001"}]}}`)); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	for _, format := range []string{"json", "table"} {
		stdout, stderr, err := runCLI(t, server.URL, true, "--format", format, "cex", "stock", "order", "list")
		if err != nil {
			t.Fatalf("format %s: %v %s", format, err, stderr)
		}
		for _, value := range []string{"9007199254740993", "0.00000001", "10.00000001"} {
			if !bytes.Contains(stdout, []byte(value)) {
				t.Errorf("format %s lost %s", format, value)
			}
		}
	}
}

// TestStockUnusableWriteResponse 成功状态却无法解码或服务端5xx时，结果未知不能诱导直接再次下单。
func TestStockUnusableWriteResponse(t *testing.T) {
	for _, status := range []int{200, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				if _, err := w.Write([]byte(`not-json`)); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			stdout, stderr, err := runCLI(t, server.URL, true, "--format", "json", "cex", "stock", "account", "transfer", "--change", "1", "--type", "deposit", "--ref-id", "synthetic-ref")
			if err == nil || !strings.Contains(string(stderr), "outcome unknown") {
				t.Errorf("unusable result lost uncertainty: %v %s", err, stderr)
			}
			if calls.Load() != 1 {
				t.Fatal("unusable write retried")
			}
			var envelope struct{ Error output.GateError }
			if decodeErr := json.Unmarshal(stderr, &envelope); decodeErr != nil || envelope.Error.Retryable || len(stdout) != 0 || !strings.Contains(envelope.Error.SuggestedNextAction, "query the original") {
				t.Fatalf("unsafe write diagnostic: decode=%v stdout=%s stderr=%s", decodeErr, stdout, stderr)
			}
		})
	}
}
