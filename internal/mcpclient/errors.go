package mcpclient

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gate/gate-cli/internal/output"
)

var (
	urlPattern = regexp.MustCompile(`https?://[^\s"]+`)
)

// ErrorKind indicates which layer failed.
type ErrorKind string

const (
	ErrorKindTransport ErrorKind = "transport"
	ErrorKindProtocol  ErrorKind = "protocol"
)

// Error represents MCP transport/protocol errors.
type Error struct {
	Kind        ErrorKind
	Err         error
	RequestID   string
	ToolName    string
	JSONRPCCode *int
}

func (e *Error) Error() string {
	if e == nil || e.Err == nil {
		return "unknown intel transport error"
	}
	return e.Err.Error()
}

func (e *Error) Unwrap() error { return e.Err }

// ParseError maps low-level errors into output.GateError for CLI printing.
func ParseError(err error, httpResp *http.Response, method, url, toolName string) *output.GateError {
	out := &output.GateError{
		Status:  http.StatusInternalServerError,
		Label:   "INTEL_ERROR",
		Message: sanitizeUserErrorMessage(err),
		Request: &output.RequestInfo{
			Method: method,
			URL:    url,
		},
		ToolName: toolName,
	}

	if httpResp != nil {
		out.Status = httpResp.StatusCode
		out.TraceID = httpResp.Header.Get("x-gate-trace-id")
	}

	var mcpErr *Error
	if errors.As(err, &mcpErr) {
		out.RequestID = mcpErr.RequestID
		out.ToolName = mcpErr.ToolName
		out.JSONRPCCode = mcpErr.JSONRPCCode
		switch mcpErr.Kind {
		case ErrorKindTransport:
			if strings.Contains(strings.ToLower(mcpErr.Error()), "response body exceeded") {
				out.Label = "INTEL_RESPONSE_TOO_LARGE"
			} else {
				out.Label = "INTEL_TRANSPORT_ERROR"
			}
		case ErrorKindProtocol:
			out.Label = "INTEL_PROTOCOL_ERROR"
		}
	}

	return out
}

func sanitizeUserErrorMessage(err error) string {
	if err == nil {
		return "intel request failed"
	}
	// Prefer stable sentinel over substring checks (CR-705 / CR-1013).
	if errors.Is(err, errIntelHTTPBodyTooLarge) {
		return "intel HTTP response exceeded the configured maximum read size (raise GATE_INTEL_MAX_RESPONSE_BYTES). " +
			"--max-output-bytes only limits how much tool output is printed locally; it does not raise this transport read cap."
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return "intel request failed"
	}
	msg = urlPattern.ReplaceAllString(msg, "[endpoint]")
	msg = strings.ReplaceAll(msg, "/mcp/", "/")
	msg = strings.ReplaceAll(msg, "/mcp", "")
	msg = strings.ReplaceAll(msg, "MCP", "intel")
	msg = strings.ReplaceAll(msg, "mcp", "intel")
	if strings.Contains(strings.ToLower(msg), "response body exceeded") {
		return fmt.Sprintf("%s; raise GATE_INTEL_MAX_RESPONSE_BYTES for the transport read cap (distinct from --max-output-bytes for printed output)", msg)
	}
	return msg
}
