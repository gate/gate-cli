package info

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/intelfacade"
	"github.com/gate/gate-cli/internal/toolschema"
)

func TestInfoAliasesCoverBaselineTools(t *testing.T) {
	oldLoader := infoSchemaLoader
	infoSchemaLoader = func() map[string]toolschema.ToolSummary { return map[string]toolschema.ToolSummary{} }
	t.Cleanup(func() { infoSchemaLoader = oldLoader })

	cmd := &cobra.Command{Use: "info"}
	orig := Cmd
	Cmd = cmd
	t.Cleanup(func() { Cmd = orig })

	buildInfoAliases()

	for _, tool := range intelfacade.InfoToolBaseline {
		parts := strings.Split(tool, "_")
		if len(parts) < 3 {
			t.Fatalf("invalid baseline tool: %s", tool)
		}
		group := parts[1]
		leaf := strings.Join(parts[2:], "-")

		groupCmd, _, err := cmd.Find([]string{group})
		if err != nil || groupCmd == nil || groupCmd.Name() != group {
			t.Fatalf("missing group command %q for tool %q", group, tool)
		}
		leafCmd, _, err := cmd.Find([]string{group, leaf})
		if err != nil || leafCmd == nil || leafCmd.Name() != leaf {
			t.Fatalf("missing leaf command %q for tool %q", leaf, tool)
		}
		if leafCmd.Flags().Lookup("params") == nil || leafCmd.Flags().Lookup("args-json") == nil || leafCmd.Flags().Lookup("args-file") == nil {
			t.Fatalf("missing fallback flags for tool %q", tool)
		}
	}
}
