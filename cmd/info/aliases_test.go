package info

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/intelcmd"
	"github.com/gate/gate-cli/internal/intelfacade"
	"github.com/gate/gate-cli/internal/mcpspec"
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

func TestInfoGetCoinInfoHasStaticFlatFlagsWhenLoaderEmpty(t *testing.T) {
	oldLoader := infoSchemaLoader
	infoSchemaLoader = func() map[string]toolschema.ToolSummary { return map[string]toolschema.ToolSummary{} }
	t.Cleanup(func() { infoSchemaLoader = oldLoader })

	cmd := &cobra.Command{Use: "info"}
	orig := Cmd
	Cmd = cmd
	t.Cleanup(func() { Cmd = orig })

	buildInfoAliases()
	leafCmd, _, err := cmd.Find([]string{"coin", "get-coin-info"})
	if err != nil || leafCmd == nil {
		t.Fatalf("find get-coin-info: %v", err)
	}
	for _, name := range []string{"query", "query-type", "size", "symbol"} {
		if leafCmd.Flags().Lookup(name) == nil {
			t.Fatalf("missing --%s on get-coin-info", name)
		}
	}
}

func TestInfoIntelLeafToolAnnotation(t *testing.T) {
	for _, tool := range intelfacade.InfoToolBaseline {
		parts := strings.Split(tool, "_")
		if len(parts) < 3 {
			t.Fatalf("invalid tool %q", tool)
		}
		group := parts[1]
		leaf := strings.Join(parts[2:], "-")
		leafCmd, _, err := Cmd.Find([]string{group, leaf})
		if err != nil || leafCmd == nil {
			t.Fatalf("find %s/%s for %q: %v", group, leaf, tool, err)
		}
		if got := leafCmd.Annotations[intelcmd.AnnotationIntelToolName]; got != tool {
			t.Fatalf("%s/%s: annotation %q want %q", group, leaf, got, tool)
		}
	}
}

// TestInfoEachLeafRegistersAllBaselineFlags ensures every baseline JSON-schema property
// is wired as a cobra flag on the matching leaf (static wiring; empty schema tools skip).
func TestInfoEachLeafRegistersAllBaselineFlags(t *testing.T) {
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
		group := parts[1]
		leaf := strings.Join(parts[2:], "-")
		leafCmd, _, err := cmd.Find([]string{group, leaf})
		if err != nil || leafCmd == nil {
			t.Fatalf("find %s/%s for %q: %v", group, leaf, tool, err)
		}
		schema := intelfacade.InfoBaselineInputSchema(tool)
		if schema == nil {
			t.Fatalf("nil baseline schema for %q", tool)
		}
		props, ok := schema["properties"].(map[string]interface{})
		if !ok {
			t.Fatalf("%q: missing properties", tool)
		}
		for k := range props {
			flagName := strings.ReplaceAll(k, "_", "-")
			if leafCmd.Flags().Lookup(flagName) == nil {
				t.Errorf("tool %q missing flag --%s (baseline key %q)", tool, flagName, k)
			}
		}
	}
}

// TestInfoEachLeafRegistersAllSpecFields ensures embedded MCP spec fields are present as flags
// (belt-and-suspenders on top of intelfacade.TestInfoBaselineCoversSpecInputFields).
func TestInfoEachLeafRegistersAllSpecFields(t *testing.T) {
	doc, err := mcpspec.InfoInputsLogic()
	if err != nil {
		t.Fatal(err)
	}
	root := doc.(map[string]interface{})
	raw := root["tools"].([]interface{})
	specByTool := make(map[string][]string, len(raw))
	for _, item := range raw {
		tm := item.(map[string]interface{})
		name, _ := tm["tool_name"].(string)
		if name == "" {
			continue
		}
		fields, _ := tm["fields"].([]interface{})
		for _, f := range fields {
			fm := f.(map[string]interface{})
			if n, _ := fm["name"].(string); n != "" {
				specByTool[name] = append(specByTool[name], n)
			}
		}
	}

	oldLoader := infoSchemaLoader
	infoSchemaLoader = func() map[string]toolschema.ToolSummary { return map[string]toolschema.ToolSummary{} }
	t.Cleanup(func() { infoSchemaLoader = oldLoader })
	cmd := &cobra.Command{Use: "info"}
	orig := Cmd
	Cmd = cmd
	t.Cleanup(func() { Cmd = orig })
	buildInfoAliases()

	for _, tool := range intelfacade.InfoToolBaseline {
		names := specByTool[tool]
		if len(names) == 0 {
			continue
		}
		parts := strings.Split(tool, "_")
		group := parts[1]
		leaf := strings.Join(parts[2:], "-")
		leafCmd, _, err := cmd.Find([]string{group, leaf})
		if err != nil || leafCmd == nil {
			t.Fatalf("find %s/%s for %q: %v", group, leaf, tool, err)
		}
		for _, field := range names {
			flagName := strings.ReplaceAll(field, "_", "-")
			if leafCmd.Flags().Lookup(flagName) == nil {
				t.Errorf("tool %q missing flag for spec field %q (--%s)", tool, field, flagName)
			}
		}
	}
}
