package output

import (
	"net/http"
	"strings"
)

// ClassifyGateAPIError maps HTTP status and Gate labels to agent-facing error_type values.
func ClassifyGateAPIError(status int, label, message string) string {
	switch status {
	case http.StatusBadRequest:
		// This explicit capability rejection cannot be fixed by changing flags.
		// Match the stable label, not translated text; other HTTP failures keep
		// their authentication, routing and uncertain-outcome semantics.
		if label == "APIOrderNotSupportUpdateTouchOrder" {
			return "UNSUPPORTED_OPERATION"
		}
	case http.StatusUnauthorized:
		return "AUTH_ERROR"
	case http.StatusForbidden:
		return "PERMISSION_DENIED"
	case http.StatusNotFound:
		// A missing HTTP route is an integration failure, not proof that the
		// authenticated account has no records. Only typed business labels
		// establish that an individual resource was not found.
		switch label {
		case "ORDER_NOT_FOUND", "POSITION_NOT_FOUND", "RECORD_NOT_FOUND":
			return "EMPTY_RESULT"
		default:
			return "API_NOT_FOUND"
		}
	case http.StatusRequestTimeout, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return "NETWORK_ERROR"
	case http.StatusTooManyRequests:
		return "NETWORK_ERROR"
	}
	blob := strings.ToLower(label + " " + message)
	switch {
	case strings.Contains(blob, "invalid key"), strings.Contains(blob, "api key"), strings.Contains(blob, "authentication"):
		return "AUTH_ERROR"
	case strings.Contains(blob, "permission"), strings.Contains(blob, "forbidden"):
		return "PERMISSION_DENIED"
	case strings.Contains(blob, "not found"), strings.Contains(blob, "no record"),
		strings.Contains(blob, "no results"), strings.Contains(blob, "no matching"),
		strings.Contains(blob, "empty result"), strings.Contains(blob, "zero results"):
		return "EMPTY_RESULT"
	}
	if status >= 500 {
		return "NETWORK_ERROR"
	}
	return ""
}

// ClassifyCLIError extends gate API mapping with CLI/Intel labels for stderr JSON.
func ClassifyCLIError(status int, label, message string) string {
	// Explicit validation labels outrank weak message matches such as "not found"
	// for business rejections carried by successful HTTP responses.
	if status >= 200 && status < 300 {
		switch label {
		case "BAD_REQUEST", "INVALID_ARGUMENT", "INVALID_ARGUMENTS", "INVALID_PARAM", "INVALID_PARAM_VALUE":
			return "INVALID_ARGS"
		}
	}
	if t := ClassifyGateAPIError(status, label, message); t != "" {
		return t
	}
	blob := strings.ToLower(label + " " + message)
	switch {
	case strings.Contains(blob, "invalid_argument"), strings.Contains(blob, "unsupported_format"):
		return "INVALID_ARGS"
	case strings.Contains(blob, "intel_transport"), strings.Contains(blob, "timeout"), strings.Contains(blob, "network"):
		return "NETWORK_ERROR"
	case strings.Contains(blob, "intel_protocol"), strings.Contains(blob, "intel_result"):
		return "UNKNOWN"
	}
	switch status {
	case http.StatusBadRequest:
		return "INVALID_ARGS"
	case http.StatusUnauthorized:
		return "AUTH_ERROR"
	case http.StatusForbidden:
		return "PERMISSION_DENIED"
	case http.StatusNotFound:
		return "API_NOT_FOUND"
	}
	if status >= 500 {
		return "NETWORK_ERROR"
	}
	return "UNKNOWN"
}

// InvalidArgsError is a caller-fixable validation error (HTTP 400).
func InvalidArgsError(message string) *GateError {
	ge := &GateError{
		Status:    http.StatusBadRequest,
		Label:     "INVALID_ARGUMENTS",
		Message:   message,
		ErrorType: "INVALID_ARGS",
	}
	FillAgentErrorConvergence(ge)
	return ge
}
