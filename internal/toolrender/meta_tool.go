package toolrender

import "strings"

// MetaToolName maps a CLI command path to the pseudo MCP-style name used for envelope meta.
func MetaToolName(commandPath string) string {
	commandPath = strings.TrimSpace(commandPath)
	if commandPath == "" {
		return ""
	}
	if !strings.Contains(commandPath, "/+") {
		return commandPath
	}
	parts := strings.SplitN(commandPath, "/", 2)
	if len(parts) != 2 {
		return commandPath
	}
	backend := parts[0]
	path := strings.TrimPrefix(parts[1], "+")
	path = strings.ReplaceAll(path, "/", "_")
	path = strings.ReplaceAll(path, "-", "_")
	if backend == "" {
		return "shortcut_" + path
	}
	return backend + "_shortcut_" + path
}

func isNewsFreshnessTool(toolName string) bool {
	return strings.HasPrefix(toolName, "news_")
}
