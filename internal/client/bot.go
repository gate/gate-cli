package client

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gate/gate-cli/internal/output"
	gateapi "github.com/gate/gateapi-go/v7"
)

// BotResultError applies AIHub's business code contract at the existing SDK boundary.
// HTTP 200 alone is not success; keep partial IDs/recovery facts without retrying writes.
func BotResultError(result any, callErr error, response *http.Response, method, path, body string) error {
	raw, encodeErr := json.Marshal(result)
	if callErr != nil {
		var withBody interface{ Body() []byte }
		var typed gateapi.GateAPIError
		if errors.As(callErr, &typed) {
			raw = typed.APIError.Body()
		} else if errors.As(callErr, &withBody) {
			raw = withBody.Body()
		} else {
			return CommandError(callErr, response, method, path, body)
		}
	}
	if encodeErr != nil {
		return CommandError(encodeErr, response, method, path, body)
	}
	var envelope struct {
		Code    int             `json:"code"`
		Label   string          `json:"label"`
		Message string          `json:"message"`
		TraceID string          `json:"trace_id"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return CommandError(err, response, method, path, body)
	}
	if callErr == nil && envelope.Code == 200 {
		if method != "POST" {
			return nil
		}
		var identity struct {
			ID     string `json:"strategy_id"`
			Status string `json:"status"`
		}
		if json.Unmarshal(envelope.Data, &identity) == nil && identity.ID != "" && identity.Status != "" {
			return nil
		}
		// A successful HTTP envelope without identity cannot prove creation/stop completion.
		envelope.Message = "write outcome unknown: AIHub response lacks a strategy ID or status"
	}
	detail := &output.GateError{Status: http.StatusOK, Label: envelope.Label, Message: envelope.Message, TraceID: envelope.TraceID, BusinessCode: &envelope.Code, Data: envelope.Data, Request: &output.RequestInfo{Method: method, URL: path, Body: body}}
	if response != nil {
		detail.Status = response.StatusCode
		if detail.TraceID == "" {
			detail.TraceID = response.Header.Get("X-Gate-Trace-Id")
		}
	}
	if detail.Label == "" {
		detail.Label = "BOT_BUSINESS_ERROR"
	}
	if detail.Message == "" {
		detail.Message = "AIHub returned an unsuccessful or incomplete business response"
	}
	var state struct {
		OperationStatus  string `json:"operation_status"`
		RecoveryRequired bool   `json:"recovery_required"`
	}
	if decodeErr := json.Unmarshal(envelope.Data, &state); decodeErr != nil {
		// Malformed/missing partial data leaves outcome unknown, so reconciliation is required.
		state.OperationStatus = ""
		state.RecoveryRequired = true
	}
	if method == "POST" && (state.OperationStatus != "rejected" || state.RecoveryRequired) {
		detail.SuggestedNextAction = "query the strategy ID and verify its state before submitting any further write"
	}
	return detail
}
