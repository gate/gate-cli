package intelcmd

import "github.com/gate/gate-cli/internal/toolrender"

// ShortcutLogicalToolName maps a shortcut CLI path to a pseudo tool name for envelope meta (e.g. news_shortcut_brief).
func ShortcutLogicalToolName(_ string, path string) string {
	return toolrender.MetaToolName(path)
}
