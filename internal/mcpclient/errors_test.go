package mcpclient

import (
	"errors"
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
