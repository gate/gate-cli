package client

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gate/gate-cli/internal/operations"
	"github.com/gate/gate-cli/internal/output"
)

// normalizeFuturesResponse recognizes only confirmed error contracts before
// successful SDK DTO decoding can discard them. It never unwraps success data,
// validates order identity, changes a request or retries a write.
func normalizeFuturesResponse(response *http.Response, method, route string) {
	if response == nil || response.StatusCode < 200 || response.StatusCode >= 300 {
		return
	}
	op, err := operations.Lookup(method, route)
	if err != nil || op == nil {
		return
	}
	switch op.OperationID {
	case "createTrailOrder", "getTrailOrders", "updateTrailOrder", "stopTrailOrder", "getTrailOrderChangeLog", "createPriceTriggeredOrder", "updatePriceTriggeredOrder":
	default:
		return
	}
	fail := func(label, message string, code *int, data json.RawMessage) {
		detail := &output.GateError{Status: response.StatusCode, Label: label, Message: message, BusinessCode: code, Data: data, TraceID: response.Header.Get("X-Gate-Trace-Id"), Request: &output.RequestInfo{Method: method, URL: route}}
		if method != http.MethodGet && method != http.MethodHead {
			detail.SuggestedNextAction = "query the original order or business reference before submitting any further write"
		}
		response.Body = &apiResponseReadError{err: detail}
	}
	if response.Body == nil {
		fail("INVALID_RESPONSE", "futures response body is missing", nil, nil)
		return
	}
	// The transport has already installed the shared 8MiB read bound. Close the
	// original exactly once on every path, then give the SDK an owned memory body.
	raw, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	if readErr != nil || closeErr != nil {
		fail("INVALID_RESPONSE", "futures response could not be read or closed; reconcile any write outcome", nil, nil)
		return
	}
	response.Body = io.NopCloser(bytes.NewReader(raw))
	response.ContentLength = int64(len(raw))
	if response.Header == nil {
		response.Header = make(http.Header)
	}
	response.Header.Set("Content-Length", strconv.Itoa(len(raw)))
	response.TransferEncoding = nil

	fields, complete := scanFuturesResponse(raw)
	if !complete {
		return
	} // Keep malformed/unknown JSON on the existing SDK path.
	// SDK v178 now declares these envelopes. Preserve native success decoding,
	// but do not turn a backend business rejection into a successful CLI exit.
	declaredEnvelope := op.OperationID == "createTrailOrder" || op.OperationID == "updateTrailOrder" || op.OperationID == "stopTrailOrder" || op.OperationID == "getTrailOrderChangeLog"
	declaredCode := declaredEnvelope && fields.count[0] > 0
	knownListError := op.OperationID == "getTrailOrders" && fields.minusOne
	labelError := fields.hasLabel && fields.hasMessage
	if !declaredCode && !knownListError && !labelError {
		return
	}
	var code *int
	var value int32
	validCode := fields.count[0] == 0 || (json.Unmarshal(fields.code, &value) == nil && !bytes.Equal(bytes.TrimSpace(fields.code), []byte("null")))
	if !validCode && labelError {
		// Complete business errors also use decimal string codes. Accept only
		// canonical nonzero int32 values; never broaden success/unknown DTOs or
		// let a malformed code hide behind otherwise valid Label/Message fields.
		text, ok := futuresResponseString(fields.code)
		if ok {
			parsed, err := strconv.ParseInt(text, 10, 32)
			if err == nil && parsed != 0 && strconv.FormatInt(parsed, 10) == text {
				value, validCode = int32(parsed), true
			}
		}
	}
	if fields.count[0] == 1 && validCode {
		n := int(value)
		code = &n
	}
	protocolError := func() { fail("INVALID_RESPONSE", "invalid futures response protocol", code, nil) }
	if !validCode || fields.count[0] > 1 || fields.count[1] > 1 || fields.count[2] > 1 {
		protocolError()
		return
	}
	if declaredCode && value == 0 {
		return
	} // These SDK envelopes declare zero as success.
	if !declaredCode && code != nil && value == 0 {
		protocolError()
		return
	}
	label, labelOK := futuresResponseString(fields.label)
	message, messageOK := futuresResponseString(fields.message)
	if !labelOK || !messageOK {
		protocolError()
		return
	}
	if fields.count[0] == 0 {
		// Reuse the existing complete Label/Message contract without changing its
		// parser or accepting arbitrary partial error-looking DTO extensions.
		var ok bool
		label, message, ok = decodeBusinessError(raw)
		if !ok {
			return
		}
	}
	if strings.TrimSpace(message) == "" {
		message = "API returned a business rejection"
	}
	var data json.RawMessage
	if op.OperationID == "createTrailOrder" && fields.count[3] == 1 {
		// Preserve the SDK's already-public partial identity, never raw debug,
		// token or account fields hidden inside an otherwise unknown envelope.
		var identity struct {
			ID string `json:"id,omitempty"`
		}
		if json.Unmarshal(fields.data, &identity) == nil && identity.ID != "" {
			var encodeErr error
			data, encodeErr = json.Marshal(identity)
			if encodeErr != nil {
				protocolError()
				return
			}
		}
	}
	fail(label, message, code, data)
}

type futuresResponseFields struct {
	code, label, message, data     json.RawMessage
	count                          [4]int
	minusOne, hasLabel, hasMessage bool
}

// Scan only four protocol fields with bounded temporary storage. Keep evidence
// from every occurrence so a duplicate final zero cannot erase an earlier error.
func scanFuturesResponse(raw []byte) (fields futuresResponseFields, complete bool) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	start, err := decoder.Token()
	if err != nil || start != json.Delim('{') {
		return fields, false
	}
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return fields, false
		}
		name, ok := key.(string)
		if !ok {
			return fields, false
		}
		var value json.RawMessage
		if decoder.Decode(&value) != nil {
			return fields, false
		}
		switch strings.ToLower(name) {
		case "code":
			fields.code = value
			fields.count[0]++
			var n int32
			if json.Unmarshal(value, &n) == nil && n == -1 {
				fields.minusOne = true
			}
		case "label":
			fields.label = value
			fields.count[1]++
			s, ok := futuresResponseString(value)
			fields.hasLabel = fields.hasLabel || ok && strings.TrimSpace(s) != ""
		case "message":
			fields.message = value
			fields.count[2]++
			s, ok := futuresResponseString(value)
			fields.hasMessage = fields.hasMessage || ok && strings.TrimSpace(s) != ""
		case "data":
			fields.data = value
			fields.count[3]++
		}
	}
	end, err := decoder.Token()
	if err != nil || end != json.Delim('}') {
		return fields, false
	}
	var trailing json.RawMessage
	return fields, decoder.Decode(&trailing) == io.EOF
}

func futuresResponseString(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 {
		return "", true
	}
	var value string
	err := json.Unmarshal(raw, &value)
	return value, err == nil && !bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}
