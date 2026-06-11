//go:build !agent

package cmdhint

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintDiagnosticOmitsMachineLinesWithoutBuildTag(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	PrintDiagnostic(&buf, &Diagnostic{
		ErrorType:           "COMMAND_NOT_FOUND",
		SuggestedNextAction: "run gate-cli info -h",
	})
	out := buf.String()
	if strings.Contains(out, "gate_cli_diagnostic=") {
		t.Fatalf("machine diagnostic must not print without -tags agent: %q", out)
	}
	if strings.Contains(out, "gate_cli_agent_resolve_hint=") {
		t.Fatalf("resolve hint must not print without -tags agent: %q", out)
	}
	if !strings.Contains(out, "Hint:") {
		t.Fatalf("human hint should still print: %q", out)
	}
}
