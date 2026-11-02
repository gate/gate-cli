package client

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	gateapi "github.com/gate/gateapi-go/v7"
)

// SDK 176 owns classification. This tests presentation of an error it already
// returned; it must never infer success/failure from a successful order DTO.
func TestSDK176ErrorPresentation(t *testing.T) {
	for _, tc := range []struct {
		name           string
		err            error
		status         int
		label, message string
	}{
		{"typed business", gateapi.GateAPIError{Label: "AUTO_ORDER_NOT_FOUND", Message: "no orderID match"}, 205, "AUTO_ORDER_NOT_FOUND", "no orderID match"},
		{"unrecognized declared error", gateapi.GenericOpenAPIError{}, 205, "API_ERROR", "API returned an unrecognized error response"},
		{"wrapped declared error", fmt.Errorf("wrapped: %w", gateapi.GenericOpenAPIError{}), 205, "API_ERROR", "API returned an unrecognized error response"},
		{"ordinary decoding unchanged", gateapi.GenericOpenAPIError{}, 200, "INVALID_RESPONSE", "cannot decode API response:"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := &http.Response{StatusCode: tc.status, Header: http.Header{"X-Gate-Trace-Id": []string{"sdk176-trace"}}}
			got := ParseGateError(tc.err, response, "GET", "/api/v4/futures/usdt/price_orders/1", "")
			if got.Status != tc.status || got.Label != tc.label || !strings.HasPrefix(got.Message, tc.message) || got.TraceID != "sdk176-trace" {
				t.Fatalf("error context changed: %+v", got)
			}
		})
	}
}

// Formatting the newly declared 205 error is scoped to the SDK176 detail
// operation; unrelated endpoints and writes retain their existing decode errors.
func TestSDK176ErrorPresentationScope(t *testing.T) {
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v4/futures/usdt/contracts"},
		{http.MethodGet, "/api/v4/delivery/usdt/price_orders/1"},
		{http.MethodDelete, "/api/v4/futures/usdt/price_orders/1"},
		{http.MethodGet, "/unknown"},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			got := ParseGateError(gateapi.GenericOpenAPIError{}, &http.Response{StatusCode: 205}, tc.method, tc.path, "")
			if got.Label != "INVALID_RESPONSE" || !strings.HasPrefix(got.Message, "cannot decode API response:") {
				t.Fatalf("unrelated operation changed: %+v", got)
			}
		})
	}
}
