package news

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/intelfacade"
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
	for _, name := range []string{"query", "coin", "limit", "sort-by"} {
		if leafCmd.Flags().Lookup(name) == nil {
			t.Fatalf("missing --%s on search-news", name)
		}
	}
}
