package intelcmd

import "strings"

// ResolveMCPToolName maps a user-facing CLI command path to the MCP wire tool name.
// MCP names (info_* / news_*) pass through unchanged. Paths like "info coin get-coin-info"
// or "coin get-coin-info" (with backend "info") resolve to info_coin_get_coin_info.
func ResolveMCPToolName(backend, nameOrPath string) string {
	nameOrPath = strings.TrimSpace(nameOrPath)
	if nameOrPath == "" {
		return ""
	}
	backend = strings.TrimSpace(backend)
	if looksLikeMCPToolName(backend, nameOrPath) {
		return nameOrPath
	}
	path := nameOrPath
	if backend != "" {
		prefix := backend + " "
		if len(path) > len(prefix) && strings.EqualFold(path[:len(prefix)], prefix) {
			path = strings.TrimSpace(path[len(prefix):])
		}
	}
	parts := strings.Fields(path)
	if len(parts) < 2 {
		return nameOrPath
	}
	group := strings.ToLower(parts[0])
	leaf := strings.ToLower(strings.Join(parts[1:], " "))
	leaf = strings.ReplaceAll(leaf, "-", "_")
	if backend == "" {
		return group + "_" + leaf
	}
	return backend + "_" + group + "_" + leaf
}

func looksLikeMCPToolName(backend, name string) bool {
	if !strings.Contains(name, "_") {
		return false
	}
	if backend != "" && strings.HasPrefix(name, backend+"_") {
		return true
	}
	return strings.HasPrefix(name, "info_") || strings.HasPrefix(name, "news_")
}
