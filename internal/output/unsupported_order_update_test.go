package output

import (
	"net/http"
	"reflect"
	"strings"
	"testing"
)

// Only the explicit business rejection denotes an unsupported operation; status
// failures and similar labels must retain their existing recovery contracts.
func TestUnsupportedOrderUpdateClassification(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, label, message, want string
		status                     int
	}{
		{"unsupported", "APIOrderNotSupportUpdateTouchOrder", "TP/SL cannot be updated", "UNSUPPORTED_OPERATION", http.StatusBadRequest},
		{"localized", "APIOrderNotSupportUpdateTouchOrder", "不支持修改止盈止损", "UNSUPPORTED_OPERATION", http.StatusBadRequest},
		{"ordinary validation", "INVALID_PARAM_VALUE", "invalid trigger price", "INVALID_ARGS", http.StatusBadRequest},
		{"message alone", "", "TP/SL cannot be updated", "INVALID_ARGS", http.StatusBadRequest},
		{"similar label", "APIOrderNotSupportUpdateTouchOrderOther", "rejected", "INVALID_ARGS", http.StatusBadRequest},
		{"unauthorized", "APIOrderNotSupportUpdateTouchOrder", "rejected", "AUTH_ERROR", http.StatusUnauthorized},
		{"forbidden", "APIOrderNotSupportUpdateTouchOrder", "rejected", "PERMISSION_DENIED", http.StatusForbidden},
		{"missing route", "APIOrderNotSupportUpdateTouchOrder", "rejected", "API_NOT_FOUND", http.StatusNotFound},
		{"server failure", "APIOrderNotSupportUpdateTouchOrder", "rejected", "NETWORK_ERROR", http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClassifyCLIError(tc.status, tc.label, tc.message); got != tc.want {
				t.Errorf("ClassifyCLIError()=%q, want %q", got, tc.want)
			}
		})
	}
}

// Convergence is called again by the printer: repeated formatting must never
// restore retry advice for a request the backend explicitly cannot support.
func TestUnsupportedOrderUpdateConvergence(t *testing.T) {
	t.Parallel()
	ge := &GateError{
		Status:    http.StatusBadRequest,
		Label:     "APIOrderNotSupportUpdateTouchOrder",
		Message:   "Updating (including editing, adding and deleting) TP/SL conditions for API orders is not supported.",
		Request:   &RequestInfo{Method: http.MethodPut, URL: "/api/v4/futures/usdt/price_orders/amend/123"},
		Retryable: true,
	}
	FillAgentErrorConvergence(ge)
	if ge.ErrorType != "UNSUPPORTED_OPERATION" || ge.Retryable {
		t.Fatalf("unsupported operation must be non-retryable: %+v", ge)
	}
	if !strings.Contains(ge.SuggestedNextAction, "API orders do not support") ||
		!strings.Contains(ge.SuggestedNextAction, "TP/SL") ||
		!strings.Contains(ge.SuggestedNextAction, "stop retrying") ||
		strings.Contains(ge.SuggestedNextAction, "corrected args") {
		t.Fatalf("misleading recovery advice: %q", ge.SuggestedNextAction)
	}
	before := *ge
	FillAgentErrorConvergence(ge)
	if !reflect.DeepEqual(*ge, before) {
		t.Fatalf("repeated convergence changed error: before=%+v, after=%+v", before, ge)
	}
}
