package intelcmd

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gate/gate-cli/internal/exitcode"
	"github.com/gate/gate-cli/internal/output"
)

// Auth exit code 4 must remain visible even when successful data uses tables.
// Keep the existing silence marker and already-typed API error behavior intact.
func TestExecuteMissingAuthAllFormats(t *testing.T) {
	const message = "OAuth2 authorization or API key required"
	for _, format := range []string{"json", "pretty", "table"} {
		t.Run(format, func(t *testing.T) {
			failure := exitcode.New(4, errors.New(message))
			var stderr bytes.Buffer
			EmitExecuteErrorEnvelope(&stderr, nil, []string{"--format", format}, failure, message)
			if !strings.Contains(stderr.String(), message) {
				t.Fatalf("auth failure was silent: %s", stderr.String())
			}
			// The new human-readable path must classify the failure as auth.
			if format == "table" && (!strings.Contains(stderr.String(), "AUTH_ERROR") || !strings.Contains(stderr.String(), "gate-cli cex auth login") || strings.Contains(stderr.String(), "fix flags")) {
				t.Fatalf("misleading auth guidance: %s", stderr.String())
			}
			var coded *exitcode.Error
			if !errors.As(failure, &coded) || coded.Code != 4 {
				t.Fatal("auth exit code changed")
			}
		})
	}
	for _, failure := range []error{ErrSilenced, fmt.Errorf("wrapped: %w", ErrSilenced)} {
		var stderr bytes.Buffer
		EmitExecuteErrorEnvelope(&stderr, nil, []string{"--format", "table"}, failure, message)
		if stderr.Len() != 0 {
			t.Fatalf("silenced error printed twice: %s", stderr.String())
		}
	}
	var stderr bytes.Buffer
	failure := exitcode.New(4, &output.GateError{Status: 401, Label: "AUTH_REQUIRED", Message: message})
	EmitExecuteErrorEnvelope(&stderr, nil, []string{"--format", "table"}, failure, message)
	if strings.Count(stderr.String(), message) != 1 {
		t.Fatalf("API auth error printed more than once: %s", stderr.String())
	}
}
