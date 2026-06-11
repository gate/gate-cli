package intelcmd

import "strings"

// ShortcutLogicalToolName maps a shortcut CLI path to a pseudo tool name for envelope meta (e.g. news_shortcut_brief).
func ShortcutLogicalToolName(backend, path string) string {
	backend = strings.TrimSpace(backend)
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, backend+"/")
	path = strings.TrimPrefix(path, "+")
	path = strings.ReplaceAll(path, "/", "_")
	path = strings.ReplaceAll(path, "-", "_")
	if backend == "" {
		return "shortcut_" + path
	}
	return backend + "_shortcut_" + path
}
