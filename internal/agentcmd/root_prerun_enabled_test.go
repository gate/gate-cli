//go:build agent

package agentcmd_test

import (
	"testing"

	"github.com/gate/gate-cli/cmd"
)

func TestRootPreRunAppliesAgentMaxOutputWithBuildTag(t *testing.T) {
	t.Setenv("GATE_CLI_AGENT", "1")
	t.Setenv("GATE_MAX_OUTPUT_BYTES", "")

	root := cmd.Root()
	if root.PersistentPreRun == nil {
		t.Fatal("expected root PersistentPreRun")
	}
	root.PersistentPreRun(root, nil)

	v, err := root.PersistentFlags().GetInt64("max-output-bytes")
	if err != nil {
		t.Fatal(err)
	}
	if v != 65536 {
		t.Fatalf("expected 65536 at pre-run with agent env, got %d", v)
	}
}
