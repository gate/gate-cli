//go:build agent

package cmdhint

import (
	"strings"

	"github.com/gate/gate-cli/internal/intelfacade"
)

// BaselineMCPCatalog returns one Leaf per info/news MCP tool in the shipped baseline (50 tools).
// Intent uses CLI path tokens (e.g. coin-get-coin-info), not MCP wire names (info_coin_get_coin_info).
func BaselineMCPCatalog() []Leaf {
	names := baselineToolNames()
	out := make([]Leaf, 0, len(names))
	for _, name := range names {
		out = append(out, BaselineToolLeaf(name))
	}
	return out
}

func baselineToolNames() []string {
	names := make([]string, 0, intelfacade.BaselineToolCount())
	names = append(names, intelfacade.InfoToolBaseline...)
	names = append(names, intelfacade.NewsToolBaseline...)
	return names
}

// BaselineToolLeaf builds a catalog Leaf entry for one MCP tool in the info/news baseline.
func BaselineToolLeaf(toolName string) Leaf {
	toolName = strings.TrimSpace(toolName)
	backend := "info"
	if strings.HasPrefix(toolName, "news_") {
		backend = "news"
	}
	cmd := MCPToolToCLICommand(toolName)
	path := catalogCLIPath(cmd)
	return Leaf{
		Intent:         cliPathIntent(path, backend),
		Command:        cmd,
		RequiredPrefix: backend,
		OutputType:     "json",
		Risk:           "public_read",
	}
}

func cliPathIntent(path, backend string) string {
	path = strings.TrimSpace(path)
	if backend != "" && strings.HasPrefix(path, backend+" ") {
		path = strings.TrimSpace(path[len(backend)+1:])
	}
	return strings.ReplaceAll(path, " ", "-")
}
