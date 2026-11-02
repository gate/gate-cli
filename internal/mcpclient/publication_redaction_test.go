package mcpclient

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

// Diagnostics retain request context but share the user-facing URL redaction
// boundary, including failures before an HTTP response exists.
func TestTransportDiagnosticsHideEndpointCredentials(t *testing.T) {
	for _, message := range []string{
		`Post "HTTPS://example.invalid/mcp?token=synthetic-query": refused`,
		`Post "http://user:synthetic-password@example.invalid/mcp?api_key=synthetic-query": refused`,
		`Post "https://example.invalid/mcp?token=synthetic-query": timeout`,
	} {
		var output bytes.Buffer
		c := &Client{transportDiag: true, errOut: &output, backend: "info"}
		c.logTransportFailure("initialize", "synthetic-request", time.Millisecond, errors.New(message))
		got := output.String()
		if strings.Contains(got, "synthetic-password") || strings.Contains(got, "synthetic-query") {
			t.Fatal("diagnostic exposed URL credentials")
		}
		if !strings.Contains(got, "synthetic-request") || !strings.Contains(got, "initialize") || !strings.Contains(got, "[endpoint]") {
			t.Fatal("safe request diagnostics were lost")
		}
	}
}
