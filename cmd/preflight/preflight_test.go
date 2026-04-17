package preflight

import "testing"

func TestPreflightFlags(t *testing.T) {
	if Cmd.Flags().Lookup("fallback-enabled") == nil {
		t.Fatalf("missing --fallback-enabled")
	}
}
