package news

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/intelcmd"
	"github.com/gate/gate-cli/internal/intelfacade"
	"github.com/gate/gate-cli/internal/mcpspec"
	"github.com/gate/gate-cli/internal/toolschema"
)

func TestNewsAliasesCoverBaselineTools(t *testing.T) {
	oldLoader := newsSchemaLoader
	newsSchemaLoader = func() map[string]toolschema.ToolSummary { return map[string]toolschema.ToolSummary{} }
	t.Cleanup(func() { newsSchemaLoader = oldLoader })

	cmd := &cobra.Command{Use: "news"}
	orig := Cmd
	Cmd = cmd
	t.Cleanup(func() { Cmd = orig })

	buildNewsAliases()

	for _, tool := range intelfacade.NewsToolBaseline {
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

func TestNewsSearchNewsHasStaticFlatFlagsWhenLoaderEmpty(t *testing.T) {
	oldLoader := newsSchemaLoader
	newsSchemaLoader = func() map[string]toolschema.ToolSummary { return map[string]toolschema.ToolSummary{} }
	t.Cleanup(func() { newsSchemaLoader = oldLoader })

	cmd := &cobra.Command{Use: "news"}
	orig := Cmd
	Cmd = cmd
	t.Cleanup(func() { Cmd = orig })

	buildNewsAliases()
	leafCmd, _, err := cmd.Find([]string{"feed", "search-news"})
	if err != nil || leafCmd == nil {
		t.Fatalf("find search-news: %v", err)
	}
	for _, name := range []string{
		"query",
		"coin",
		"platform",
		"platform-type",
		"lang",
		"time-range",
		"start-time",
		"end-time",
		"sort-by",
		"top-total-score",
		"limit",
		"page",
		"similarity-score",
	} {
		if leafCmd.Flags().Lookup(name) == nil {
			t.Fatalf("missing --%s on search-news", name)
		}
	}
}

func TestNewsIntelLeafToolAnnotation(t *testing.T) {
	for _, tool := range intelfacade.NewsToolBaseline {
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

func TestNewsEachLeafRegistersAllBaselineFlags(t *testing.T) {
	oldLoader := newsSchemaLoader
	newsSchemaLoader = func() map[string]toolschema.ToolSummary { return map[string]toolschema.ToolSummary{} }
	t.Cleanup(func() { newsSchemaLoader = oldLoader })

	cmd := &cobra.Command{Use: "news"}
	orig := Cmd
	Cmd = cmd
	t.Cleanup(func() { Cmd = orig })
	buildNewsAliases()

	for _, tool := range intelfacade.NewsToolBaseline {
		parts := strings.Split(tool, "_")
		group := parts[1]
		leaf := strings.Join(parts[2:], "-")
		leafCmd, _, err := cmd.Find([]string{group, leaf})
		if err != nil || leafCmd == nil {
			t.Fatalf("find %s/%s for %q: %v", group, leaf, tool, err)
		}
		schema := intelfacade.NewsBaselineInputSchema(tool)
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

func TestNewsEachLeafRegistersAllSpecParams(t *testing.T) {
	doc, err := mcpspec.NewsToolsArgs()
	if err != nil {
		t.Fatal(err)
	}
	root := doc.(map[string]interface{})
	raw := root["tools"].([]interface{})
	specParamsByTool := make(map[string][]string, len(raw))
	for _, item := range raw {
		tm := item.(map[string]interface{})
		name, _ := tm["name"].(string)
		if name == "" {
			continue
		}
		ir, ok := tm["input_rules"].(map[string]interface{})
		if !ok {
			continue
		}
		params, ok := ir["params"].([]interface{})
		if !ok {
			continue
		}
		for _, p := range params {
			pm := p.(map[string]interface{})
			if pn, _ := pm["name"].(string); pn != "" {
				specParamsByTool[name] = append(specParamsByTool[name], pn)
			}
		}
	}

	oldLoader := newsSchemaLoader
	newsSchemaLoader = func() map[string]toolschema.ToolSummary { return map[string]toolschema.ToolSummary{} }
	t.Cleanup(func() { newsSchemaLoader = oldLoader })
	cmd := &cobra.Command{Use: "news"}
	orig := Cmd
	Cmd = cmd
	t.Cleanup(func() { Cmd = orig })
	buildNewsAliases()

	for _, tool := range intelfacade.NewsToolBaseline {
		names := specParamsByTool[tool]
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
		for _, param := range names {
			flagName := strings.ReplaceAll(param, "_", "-")
			if leafCmd.Flags().Lookup(flagName) == nil {
				t.Errorf("tool %q missing flag for spec param %q (--%s)", tool, param, flagName)
			}
		}
	}
}
