//go:build agentic_e2e

package agentic_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gate/gate-cli/internal/output"
)

// Only the HTTP API peer is mocked. The real CLI process, SDK request, error
// parsing, printer and exit status remain in the business chain under test.
func TestPriceTriggerUpdateErrorCLI(t *testing.T) {
	for _, tc := range []struct {
		name, label, message, errorType, advice string
		status, exitCode                        int
		retryable                               bool
	}{
		{"unsupported", "APIOrderNotSupportUpdateTouchOrder", "Updating (including editing, adding and deleting) TP/SL conditions for API orders is not supported.", "UNSUPPORTED_OPERATION", "API orders do not support", 400, 1, false},
		{"localized unsupported", "APIOrderNotSupportUpdateTouchOrder", "不支持修改止盈止损", "UNSUPPORTED_OPERATION", "API orders do not support", 400, 1, false},
		{"correctable parameters", "INVALID_PARAM_VALUE", "invalid trigger price", "INVALID_ARGS", "corrected args", 400, 1, true},
		{"permission failure", "INSUFFICIENT_SCOPE", "missing futures_trade scope", "PERMISSION_DENIED", "verify granted scopes", 403, 4, false},
		{"unknown write outcome", "APIOrderNotSupportUpdateTouchOrder", "upstream failure", "NETWORK_ERROR", "query the original", 503, 1, false},
	} {
		for _, format := range []string{"json", "pretty", "table"} {
			t.Run(tc.name+"/"+format, func(t *testing.T) {
				var calls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.Method != http.MethodPut || r.URL.Path != "/api/v4/futures/usdt/price_orders/amend/9007199254740993" {
						t.Errorf("unexpected request: %s %s", r.Method, r.URL)
					}
					var body struct {
						OrderID      int64  `json:"order_id"`
						TriggerPrice string `json:"trigger_price"`
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Errorf("decode SDK request: %v", err)
					} else if body.OrderID != 9007199254740993 || body.TriggerPrice != "101000" {
						t.Errorf("unexpected SDK body: %+v", body)
					}
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("X-Gate-Trace-Id", "price-trigger-update-trace")
					w.WriteHeader(tc.status)
					if err := json.NewEncoder(w).Encode(map[string]string{"label": tc.label, "message": tc.message}); err != nil {
						t.Errorf("write API error: %v", err)
					}
				}))
				t.Cleanup(server.Close)
				stdout, stderr, err := runCLI(t, server.URL, true, "--format", format, "cex", "futures", "price-trigger", "update", "--id", "9007199254740993", "--body-order-id", "9007199254740993", "--trigger-price", "101000", "--settle", "usdt")
				var exit *exec.ExitError
				if !errors.As(err, &exit) || exit.ExitCode() != tc.exitCode || len(stdout) != 0 || calls.Load() != 1 {
					t.Fatalf("exit=%v stdout=%s calls=%d stderr=%s", err, stdout, calls.Load(), stderr)
				}
				if format == "json" {
					var envelope struct{ Error output.GateError }
					if err := json.Unmarshal(stderr, &envelope); err != nil {
						t.Fatalf("decode CLI error: %v, stderr=%s", err, stderr)
					}
					ge := envelope.Error
					if ge.Status != tc.status || ge.Label != tc.label || !strings.Contains(ge.Message, tc.message) || ge.TraceID != "price-trigger-update-trace" || ge.ErrorType != tc.errorType || ge.Retryable != tc.retryable || !strings.Contains(ge.SuggestedNextAction, tc.advice) {
						t.Fatalf("unexpected CLI error: %+v", ge)
					}
					if ge.Request == nil || ge.Request.Method != http.MethodPut || ge.Request.URL != "/api/v4/futures/usdt/price_orders/amend/9007199254740993" {
						t.Fatalf("lost request diagnostic: %+v", ge.Request)
					}
				} else {
					for _, want := range []string{tc.label, tc.message, tc.errorType, tc.advice, "price-trigger-update-trace"} {
						if !strings.Contains(string(stderr), want) {
							t.Errorf("missing %q in stderr=%s", want, stderr)
						}
					}
				}
			})
		}
	}
}

// A successful amend must retain its normal output and must not be locally
// blocked merely because API-created orders can receive a capability rejection.
func TestPriceTriggerUpdateSuccessCLI(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodPut || r.URL.Path != "/api/v4/futures/usdt/price_orders/amend/123" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]int64{"id": 123}); err != nil {
			t.Errorf("write API success: %v", err)
		}
	}))
	t.Cleanup(server.Close)
	stdout, stderr, err := runCLI(t, server.URL, true, "--format", "json", "cex", "futures", "price-trigger", "update", "--id", "123", "--body-order-id", "123", "--trigger-price", "101000")
	var result struct {
		ID int64 `json:"id"`
	}
	if err != nil || len(stderr) != 0 || calls.Load() != 1 || json.Unmarshal(stdout, &result) != nil || result.ID != 123 {
		t.Fatalf("exit=%v stdout=%s calls=%d stderr=%s", err, stdout, calls.Load(), stderr)
	}
}
