package intelcmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/toolschema"
)

type AliasBuildOptions struct {
	BackendPrefix   string
	BackendTitle    string
	ToolBaseline    []string
	SchemaSummaries map[string]toolschema.ToolSummary
	BusinessAliases map[string][]string
	MakeAlias       func(toolUse, toolName string) *cobra.Command
	ApplyBaseline   func(toolName string, cmd *cobra.Command)
	AfterAliasBuilt func(toolName string, cmd *cobra.Command)
}

// BuildGroupedAliases constructs grouped alias command trees for info/news style backends.
func BuildGroupedAliases(opts *AliasBuildOptions) []*cobra.Command {
	if opts == nil || opts.MakeAlias == nil {
		return nil
	}
	groups := map[string]*cobra.Command{}
	orderedGroupNames := []string{}
	for _, tool := range opts.ToolBaseline {
		parts := strings.Split(tool, "_")
		if len(parts) < 3 || parts[0] != opts.BackendPrefix {
			continue
		}
		groupName := parts[1]
		toolUse := strings.Join(parts[2:], "-")
		group, ok := groups[groupName]
		if !ok {
			group = &cobra.Command{Use: groupName, Short: opts.BackendTitle + " " + groupName + " shortcuts"}
			groups[groupName] = group
			orderedGroupNames = append(orderedGroupNames, groupName)
		}
		alias := opts.MakeAlias(toolUse, tool)
		if aliases, ok := opts.BusinessAliases[tool]; ok {
			alias.Aliases = aliases
		}
		if opts.ApplyBaseline != nil {
			opts.ApplyBaseline(tool, alias)
		}
		if schema, ok := opts.SchemaSummaries[tool]; ok && !toolschema.IsEmptyInputSchema(schema.InputSchema) {
			toolschema.ApplyInputSchemaFlags(alias, schema.InputSchema)
		}
		if opts.AfterAliasBuilt != nil {
			opts.AfterAliasBuilt(tool, alias)
		}
		group.AddCommand(alias)
	}
	out := make([]*cobra.Command, 0, len(orderedGroupNames))
	for _, name := range orderedGroupNames {
		out = append(out, groups[name])
	}
	return out
}
