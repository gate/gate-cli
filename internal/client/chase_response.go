package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gate/gate-cli/internal/operations"
	"github.com/gate/gate-cli/internal/output"
)

// apiResponseReadError defers a checked response failure to the SDK's body-read boundary.
// Keeping an ordinary HTTP response preserves its real status/trace; net/http would
// discard that response if RoundTrip returned both a response and an error.
type apiResponseReadError struct{ err error }

func (body *apiResponseReadError) Read([]byte) (int, error) { return 0, body.err }
func (body *apiResponseReadError) Close() error             { return nil }

// normalizeChaseResponse adapts the deployed {code,message,data} protocol to the
// existing SDK models. Flat API-schema responses remain supported. Only the five
// catalogued chase operations participate; authentication, signing and requests
// continue through the same client and SDK. No write is retried here.
func normalizeChaseResponse(response *http.Response, method, route string) {
	if response == nil || response.StatusCode < 200 || response.StatusCode >= 300 {
		return
	}
	operation, err := operations.Lookup(method, route)
	if err != nil || operation == nil {
		return
	}
	switch operation.OperationID {
	case "createChaseOrder", "getChaseOrderDetail", "getChaseOrders", "stopChaseOrder", "stopAllChaseOrders":
	default:
		return
	}
	fail := func(message string, code *int, label string, data json.RawMessage) {
		detail := &output.GateError{Status: response.StatusCode, Message: message, Label: label, BusinessCode: code, Data: data, TraceID: response.Header.Get("X-Gate-Trace-Id"), Request: &output.RequestInfo{Method: method, URL: route}}
		if method != "GET" {
			detail.SuggestedNextAction = "query the chase order and any child orders before submitting further writes"
		}
		response.Body = &apiResponseReadError{err: detail}
	}
	if response.Body == nil {
		fail("chase response body is missing; reconcile any write outcome", nil, "", nil)
		return
	}
	raw, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	if readErr != nil {
		fail("chase response could not be read; reconcile any write outcome", nil, "", nil)
		return
	}
	if closeErr != nil {
		fail("chase response could not be closed; reconcile any write outcome", nil, "", nil)
		return
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		fail("invalid chase response JSON; reconcile any write outcome", nil, "", nil)
		return
	}
	payload := raw
	if codeRaw, wrapped := fields["code"]; wrapped {
		var code int
		if err := json.Unmarshal(codeRaw, &code); err != nil || bytes.Equal(bytes.TrimSpace(codeRaw), []byte("null")) {
			fail("invalid chase response business code; reconcile any write outcome", nil, "", nil)
			return
		}
		var message, label string
		if messageRaw, ok := fields["message"]; ok {
			if err := json.Unmarshal(messageRaw, &message); err != nil {
				fail("invalid chase response message; reconcile any write outcome", &code, "", nil)
				return
			}
		}
		if labelRaw, ok := fields["label"]; ok {
			if err := json.Unmarshal(labelRaw, &label); err != nil {
				fail("invalid chase response label; reconcile any write outcome", &code, "", nil)
				return
			}
		}
		if code != 0 {
			fail(message, &code, label, fields["data"])
			return
		}
		payload = fields["data"]
	}
	// Success must contain the SDK's result object, never an ignored error envelope.
	var data map[string]json.RawMessage
	if err := json.Unmarshal(payload, &data); err != nil || data == nil {
		fail("chase response data is missing or invalid; reconcile any write outcome", nil, "", nil)
		return
	}
	required := "order"
	switch operation.OperationID {
	case "createChaseOrder":
		required = "id"
	case "getChaseOrders", "stopAllChaseOrders":
		required = "orders"
	}
	value, ok := data[required]
	if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		fail(fmt.Sprintf("chase response lacks %s; reconcile any write outcome", required), nil, "", nil)
		return
	}
	switch required {
	case "id":
		var id string
		if err := json.Unmarshal(value, &id); err != nil || id == "" {
			fail("chase response lacks a valid order ID; reconcile any write outcome", nil, "", nil)
			return
		}
	case "orders":
		var orders []struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(value, &orders); err != nil {
			fail("chase response orders must be an array; reconcile any write outcome", nil, "", nil)
			return
		}
		for _, order := range orders {
			if order.ID == "" {
				fail("chase response contains an order without an ID; reconcile any write outcome", nil, "", nil)
				return
			}
		}
	default:
		var order struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(value, &order); err != nil || order.ID == "" {
			fail("chase response lacks a valid order object; reconcile any write outcome", nil, "", nil)
			return
		}
	}
	response.Body = io.NopCloser(bytes.NewReader(payload))
	response.ContentLength = int64(len(payload))
	if response.Header == nil {
		response.Header = make(http.Header)
	}
	response.Header.Set("Content-Length", fmt.Sprint(len(payload)))
	response.TransferEncoding = nil
}
