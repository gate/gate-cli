package client_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	gateapi "github.com/gate/gateapi-go/v7"

	"github.com/gate/gate-cli/internal/client"
)

// makeHTTPResp builds a minimal *http.Response with the given status and headers.
func makeHTTPResp(status int, headers map[string]string) *http.Response {
	h := make(http.Header)
	for k, v := range headers {
		h.Set(k, v)
	}
	return &http.Response{StatusCode: status, Header: h}
}

// TestParseGateError_NilHTTPResp verifies that a nil httpResp still returns a
// usable GateError with status 500 and no trace ID.
func TestParseGateError_NilHTTPResp(t *testing.T) {
	err := fmt.Errorf("network error")
	gateErr := client.ParseGateError(err, nil, "GET", "/test", "")

	assert.Equal(t, 500, gateErr.Status)
	assert.Empty(t, gateErr.TraceID)
	assert.Equal(t, "network error", gateErr.Message)
}

// TestParseGateError_TraceIDFromHeader verifies trace ID is read from the
// x-gate-trace-id response header.
func TestParseGateError_TraceIDFromHeader(t *testing.T) {
	resp := makeHTTPResp(429, map[string]string{
		"x-gate-trace-id": "abc-123",
	})
	err := fmt.Errorf("rate limited")
	gateErr := client.ParseGateError(err, resp, "GET", "/test", "")

	assert.Equal(t, 429, gateErr.Status)
	assert.Equal(t, "abc-123", gateErr.TraceID)
}

// TestParseGateError_TraceIDHeaderCaseInsensitive verifies that header name
// comparison is case-insensitive (Go http.Header canonical form).
func TestParseGateError_TraceIDHeaderCaseInsensitive(t *testing.T) {
	// Simulate a server that returns the header with mixed case.
	h := make(http.Header)
	h["X-Gate-Trace-Id"] = []string{"canonical-456"} // stored in canonical form by Go HTTP
	resp := &http.Response{StatusCode: 400, Header: h}

	gateErr := client.ParseGateError(fmt.Errorf("bad request"), resp, "POST", "/test", "")
	assert.Equal(t, "canonical-456", gateErr.TraceID)
}

// TestParseGateError_NoTraceIDHeader verifies that an empty TraceID is returned
// when the response header is absent.
func TestParseGateError_NoTraceIDHeader(t *testing.T) {
	resp := makeHTTPResp(400, nil)
	err := fmt.Errorf("bad request")
	gateErr := client.ParseGateError(err, resp, "POST", "/test", "{}")

	assert.Equal(t, 400, gateErr.Status)
	assert.Empty(t, gateErr.TraceID)
}

// TestParseGateError_GateAPIError verifies label, message, status, and trace ID
// are all populated correctly for a structured GateAPIError.
func TestParseGateError_GateAPIError(t *testing.T) {
	resp := makeHTTPResp(422, map[string]string{
		"x-gate-trace-id": "trace-xyz",
	})
	apiErr := gateapi.GateAPIError{
		Label:   "INVALID_PARAM",
		Message: "amount is required",
	}
	gateErr := client.ParseGateError(apiErr, resp, "POST", "/spot/orders", `{"pair":"BTC_USDT"}`)

	assert.Equal(t, 422, gateErr.Status)
	assert.Equal(t, "INVALID_PARAM", gateErr.Label)
	assert.Equal(t, "amount is required", gateErr.Message)
	assert.Equal(t, "trace-xyz", gateErr.TraceID)
	assert.Equal(t, "POST", gateErr.Request.Method)
}

// TestParseGateError_GateAPIErrorNoTraceID verifies that a GateAPIError without
// the header results in an empty TraceID (Gate API does not always send it).
func TestParseGateError_GateAPIErrorNoTraceID(t *testing.T) {
	resp := makeHTTPResp(422, nil) // no x-gate-trace-id header
	apiErr := gateapi.GateAPIError{
		Label:   "INVALID_PARAM",
		Message: "amount is required",
	}
	gateErr := client.ParseGateError(apiErr, resp, "POST", "/spot/orders", "")
	assert.Empty(t, gateErr.TraceID)
}

// TestParseGateError_GateAPIErrorDetailOverridesMessage verifies that Detail
// takes precedence over Message (as per SDK GetMessage() semantics).
func TestParseGateError_GateAPIErrorDetailOverridesMessage(t *testing.T) {
	resp := makeHTTPResp(400, nil)
	apiErr := gateapi.GateAPIError{
		Label:   "INVALID_PARAM_VALUE",
		Message: "generic message",
		Detail:  "price must be positive",
	}
	gateErr := client.ParseGateError(apiErr, resp, "POST", "/test", "")

	assert.Equal(t, "price must be positive", gateErr.Message)
}

// TestParseGateError_RequestInfo verifies that request method, URL, and body
// are captured in the GateError.
func TestParseGateError_RequestInfo(t *testing.T) {
	resp := makeHTTPResp(500, nil)
	err := fmt.Errorf("server error")
	gateErr := client.ParseGateError(err, resp, "DELETE", "/futures/usdt/orders/123", "")

	assert.Equal(t, "DELETE", gateErr.Request.Method)
	assert.Equal(t, "/futures/usdt/orders/123", gateErr.Request.URL)
}
