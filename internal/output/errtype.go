package output

import (
	"net/http"
	"strings"
)

// ClassifyGateAPIError maps HTTP status and Gate labels to agent-facing error_type values.
func ClassifyGateAPIError(status int, label, message string) string {
	switch status {
	case http.StatusUnauthorized:
		return "AUTH_ERROR"
	case http.StatusForbidden:
		return "PERMISSION_DENIED"
	case http.StatusNotFound:
		return "EMPTY_RESULT"
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
		return "EMPTY_RESULT"
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
