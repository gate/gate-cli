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

// normalizeInsuranceResponse keeps the shared SDK model's string balance while
// accepting the API's numeric form. RawMessage preserves every decimal digit;
// authentication and HTTP execution remain in the existing client boundary.
func normalizeInsuranceResponse(response *http.Response, method, route string) {
	if response == nil || response.StatusCode < 200 || response.StatusCode >= 300 {
		return
	}
	operation, err := operations.Lookup(method, route)
	if err != nil || operation == nil {
		return
	}
	if operation.OperationID != "listFuturesInsuranceLedger" && operation.OperationID != "listDeliveryInsuranceLedger" {
		return
	}
	fail := func(reason string) {
		response.Body = &apiResponseReadError{err: &output.GateError{
			Status: response.StatusCode, Label: "INVALID_RESPONSE", ErrorType: "UNKNOWN",
			Message: "cannot decode insurance response: " + reason,
			TraceID: response.Header.Get("X-Gate-Trace-Id"), Request: &output.RequestInfo{Method: method, URL: route},
		}}
	}
	if response.Body == nil {
		fail("missing response body")
		return
	}
	// The transport wraps this body with its byte bound before any adapter reads it.
	raw, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	if readErr != nil {
		fail(readErr.Error())
		return
	}
	if closeErr != nil {
		fail("response body close failed")
		return
	}
	var records []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &records); err != nil || records == nil {
		fail("expected an insurance record array")
		return
	}
	for i, record := range records {
		balance := bytes.TrimSpace(record["b"])
		if len(balance) == 0 || bytes.Equal(balance, []byte("null")) {
			fail(fmt.Sprintf("record %d is missing balance b", i))
			return
		}
		if balance[0] == '"' {
			var text string
			if err := json.Unmarshal(balance, &text); err != nil || text == "" {
				fail(fmt.Sprintf("record %d has invalid string balance b", i))
				return
			}
			continue
		}
		// JSON syntax has already been validated. Only a number may be converted;
		// bool/object/array values must not become plausible successful balances.
		if balance[0] != '-' && (balance[0] < '0' || balance[0] > '9') {
			fail(fmt.Sprintf("record %d balance b must be a number or string", i))
			return
		}
		encoded, err := json.Marshal(string(balance))
		if err != nil {
			fail("cannot encode balance b")
			return
		}
		record["b"] = encoded
	}
	normalized, err := json.Marshal(records)
	if err != nil {
		fail("cannot encode insurance records")
		return
	}
	response.Body = io.NopCloser(bytes.NewReader(normalized))
	response.ContentLength = int64(len(normalized))
	if response.Header == nil {
		response.Header = make(http.Header)
	}
	response.Header.Set("Content-Length", fmt.Sprint(len(normalized)))
	response.TransferEncoding = nil
}
