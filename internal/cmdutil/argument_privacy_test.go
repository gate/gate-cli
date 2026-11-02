package cmdutil

import (
	"errors"
	"strings"
	"testing"

	"github.com/gate/gate-cli/internal/output"
	"github.com/spf13/cobra"
)

// Validation must classify only the input problem, without turning an extra
// word into unknown-command diagnostics that can include credential argv.
func TestArgumentPolicyPrivateError(t *testing.T) {
	root := &cobra.Command{Use: "gate-cli"}
	leaf := &cobra.Command{Use: "version", Run: func(*cobra.Command, []string) {}}
	root.AddCommand(leaf)
	NewArgumentPolicy(root)
	if err := leaf.Args(leaf, nil); err != nil {
		t.Fatalf("normal invocation: %v", err)
	}
	err := leaf.Args(leaf, []string{"secret-canary-cli"})
	var ge *output.GateError
	if !errors.As(err, &ge) || strings.Contains(err.Error(), "secret-canary-cli") || strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("unsafe positional rejection: %v", err)
	}
}
