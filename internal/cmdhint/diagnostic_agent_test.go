//go:build agent

package cmdhint

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintDiagnosticJSONLineWithBuildTag(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	PrintDiagnostic(&buf, &Diagnostic{
		ErrorType: "COMMAND_NOT_FOUND",
		Suggested: "gate-cli cex earn uni records",
	})
	out := buf.String()
	if !strings.Contains(out, "gate_cli_diagnostic=") {
		t.Fatalf("missing diagnostic line with -tags agent: %q", out)
	}
	if !strings.Contains(out, "Hint:") {
		t.Fatalf("missing human hint: %q", out)
	}
}
