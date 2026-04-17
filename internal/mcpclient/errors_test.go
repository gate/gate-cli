package mcpclient

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestSanitizeUserErrorMessage_RemovesURLAndMCPWord(t *testing.T) {
	msg := sanitizeUserErrorMessage(errors.New(`Post "https://api.gatemcp.ai/mcp/info": Forbidden`))
	if msg == "" {
		t.Fatal("expected non-empty message")
	}
	if strings.Contains(strings.ToLower(msg), "mcp") {
		t.Fatalf("expected no mcp in message, got: %s", msg)
	}
	if strings.Contains(strings.ToLower(msg), "https://") {
		t.Fatalf("expected no raw url in message, got: %s", msg)
	}
}

func TestSanitizeUserErrorMessage_ResponseTooLargeHint(t *testing.T) {
	wrapped := fmt.Errorf("response body exceeded 16777216 bytes: %w", errIntelHTTPBodyTooLarge)
	msg := sanitizeUserErrorMessage(wrapped)
	if !strings.Contains(msg, "GATE_INTEL_MAX_RESPONSE_BYTES") {
		t.Fatalf("expected response-too-large hint, got: %s", msg)
	}
	if strings.Contains(strings.ToLower(msg), "--max-output-bytes") && !strings.Contains(msg, "transport read cap") {
		t.Fatalf("expected transport vs CLI output distinction, got: %s", msg)
	}
}

func TestParseError_ResponseTooLargeLabel(t *testing.T) {
	err := &Error{Kind: ErrorKindTransport, Err: errors.New("response body exceeded 16777216 bytes")}
	ge := ParseError(err, &http.Response{StatusCode: 502}, "POST", "news/invoke", "x")
	if ge.Label != "INTEL_RESPONSE_TOO_LARGE" {
		t.Fatalf("expected INTEL_RESPONSE_TOO_LARGE, got: %s", ge.Label)
	}
}
