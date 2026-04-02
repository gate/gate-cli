// Package useragent builds a structured User-Agent string for gate-cli HTTP requests.
// It auto-detects the calling environment (IDE / AI agent / terminal) via environment variables.
package useragent

import (
	"os"
	"strings"

	"github.com/gate/gate-cli/internal/version"
)

// AgentInfo describes the detected calling environment.
type AgentInfo struct {
	Name  string // e.g. "cursor", "claude-code", "terminal"
	Extra string // supplementary info, e.g. "cli", "agent", "quest"; "-" when unavailable
}

// detector defines a single environment detection rule.
type detector struct {
	envKey string // environment variable to check
	envVal string // expected value; empty means any non-empty value matches
	name   string // agent name to return on match
	extKey string // env var for extra info; empty means no extra
}

// Ordered list of known environments.
// VSCode-based IDEs (Cursor, Qoder, Antigravity, Trae, Windsurf) MUST appear
// before the generic TERM_PROGRAM=vscode fallback, because they all inherit VSCODE_* vars.
var knownDetectors = []detector{
	{"CLAUDECODE", "1", "claude-code", "CLAUDE_CODE_ENTRYPOINT"},
	{"CURSOR_AGENT", "1", "cursor", "CURSOR_INVOKED_AS"},
	{"QODER_CLI", "1", "qoder", "QODERCLI_INTEGRATION_MODE"},
	{"ANTIGRAVITY_AGENT", "1", "antigravity", "TERM_PROGRAM_VERSION"},
	{"AI_AGENT", "TRAE", "trae", "TERM_PROGRAM_VERSION"},
	{"OPENCODE", "1", "opencode", ""},
	{"CODEX_CI", "1", "codex", "CODEX_INTERNAL_ORIGINATOR_OVERRIDE"},
	{"TERMINAL_EMULATOR", "JetBrains-JediTerm", "jetbrains", "__CFBundleIdentifier"},
	{"TERM_PROGRAM", "vscode", "vscode", "TERM_PROGRAM_VERSION"},
}

// Detect identifies the calling environment from environment variables.
func Detect() AgentInfo {
	// Priority 1: explicit override
	if name := os.Getenv("GATE_CLI_AGENT"); name != "" {
		return AgentInfo{
			Name:  name,
			Extra: envOrDefault("GATE_CLI_AGENT_VERSION", "-"),
		}
	}

	// Priority 2–10: known environments (table-driven)
	for _, d := range knownDetectors {
		v := os.Getenv(d.envKey)
		if v == "" {
			continue
		}
		if d.envVal != "" && !strings.EqualFold(v, d.envVal) {
			continue
		}
		return AgentInfo{
			Name:  d.name,
			Extra: envOrDefault(d.extKey, "-"),
		}
	}

	// Windsurf: no WINDSURF_* vars; only macOS __CFBundleIdentifier contains "windsurf".
	if bid := os.Getenv("__CFBundleIdentifier"); strings.Contains(strings.ToLower(bid), "windsurf") {
		return AgentInfo{
			Name:  "windsurf",
			Extra: envOrDefault("TERM_PROGRAM_VERSION", "-"),
		}
	}

	// Generic TERM_PROGRAM fallback (e.g. "iTerm.app", "Apple_Terminal")
	if tp := os.Getenv("TERM_PROGRAM"); tp != "" {
		return AgentInfo{Name: strings.ToLower(tp), Extra: "-"}
	}

	// Default
	return AgentInfo{Name: "terminal", Extra: "-"}
}

// Build assembles the full User-Agent string:
//
//	gate-cli/{version}/{cmdPath}/{agent}/{extra} {sdkUA}
func Build(cmdPath, sdkUA string) string {
	agent := Detect()
	return "gate-cli/" + version.Version +
		"/" + cmdPath +
		"/" + agent.Name +
		"/" + agent.Extra +
		" " + sdkUA
}

// ExtractCmdPath converts a cobra CommandPath to slash-separated format,
// stripping the root command name.
//
//	"gate-cli spot order create" → "spot/order/create"
func ExtractCmdPath(commandPath string) string {
	parts := strings.Fields(commandPath)
	if len(parts) > 1 {
		return strings.Join(parts[1:], "/")
	}
	return "unknown"
}

func envOrDefault(key, fallback string) string {
	if key == "" {
		return fallback
	}
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
