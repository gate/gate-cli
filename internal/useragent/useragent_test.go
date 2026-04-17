package useragent

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// allEnvKeys collects every env key used by the detection logic so we can
// cleanly save/restore state between subtests.
var allEnvKeys = []string{
	"GATE_CLI_AGENT", "GATE_CLI_AGENT_VERSION",
	"CLAUDECODE", "CLAUDE_CODE_ENTRYPOINT",
	"CURSOR_AGENT", "CURSOR_INVOKED_AS",
	"QODER_CLI", "QODERCLI_INTEGRATION_MODE",
	"ANTIGRAVITY_AGENT", "TERM_PROGRAM_VERSION",
	"AI_AGENT", "TRAE_BRAND_NAME",
	"OPENCODE",
	"CODEX_CI", "CODEX_INTERNAL_ORIGINATOR_OVERRIDE",
	"TERMINAL_EMULATOR", "__CFBundleIdentifier",
	"TERM_PROGRAM",
}

// clearEnv unsets all detection-related env vars and returns a restore function.
func clearEnv(t *testing.T) func() {
	t.Helper()
	saved := make(map[string]string)
	for _, k := range allEnvKeys {
		if v, ok := os.LookupEnv(k); ok {
			saved[k] = v
		}
		_ = os.Unsetenv(k)
	}
	return func() {
		for _, k := range allEnvKeys {
			if v, ok := saved[k]; ok {
				_ = os.Setenv(k, v)
			} else {
				_ = os.Unsetenv(k)
			}
		}
	}
}

func TestDetect(t *testing.T) {
	tests := []struct {
		name      string
		envs      map[string]string
		wantName  string
		wantExtra string
	}{
		{
			name:      "explicit override with version",
			envs:      map[string]string{"GATE_CLI_AGENT": "my-bot", "GATE_CLI_AGENT_VERSION": "2.0"},
			wantName:  "my-bot",
			wantExtra: "2.0",
		},
		{
			name:      "explicit override without version",
			envs:      map[string]string{"GATE_CLI_AGENT": "ci-runner"},
			wantName:  "ci-runner",
			wantExtra: "-",
		},
		{
			name:      "Claude Code CLI",
			envs:      map[string]string{"CLAUDECODE": "1", "CLAUDE_CODE_ENTRYPOINT": "cli"},
			wantName:  "claude-code",
			wantExtra: "cli",
		},
		{
			name:      "Claude Code Desktop",
			envs:      map[string]string{"CLAUDECODE": "1", "CLAUDE_CODE_ENTRYPOINT": "desktop"},
			wantName:  "claude-code",
			wantExtra: "desktop",
		},
		{
			name:      "Claude Code without entrypoint",
			envs:      map[string]string{"CLAUDECODE": "1"},
			wantName:  "claude-code",
			wantExtra: "-",
		},
		{
			name:      "Cursor Agent (IDE)",
			envs:      map[string]string{"CURSOR_AGENT": "1"},
			wantName:  "cursor",
			wantExtra: "-",
		},
		{
			name:      "Cursor CLI",
			envs:      map[string]string{"CURSOR_AGENT": "1", "CURSOR_INVOKED_AS": "agent"},
			wantName:  "cursor",
			wantExtra: "agent",
		},
		{
			name:      "Qoder",
			envs:      map[string]string{"QODER_CLI": "1", "QODERCLI_INTEGRATION_MODE": "quest"},
			wantName:  "qoder",
			wantExtra: "quest",
		},
		{
			name:      "Antigravity",
			envs:      map[string]string{"ANTIGRAVITY_AGENT": "1", "TERM_PROGRAM_VERSION": "1.107.0"},
			wantName:  "antigravity",
			wantExtra: "1.107.0",
		},
		{
			name:      "Trae",
			envs:      map[string]string{"AI_AGENT": "TRAE", "TERM_PROGRAM_VERSION": "1.5.0"},
			wantName:  "trae",
			wantExtra: "1.5.0",
		},
		{
			name:      "Trae case insensitive",
			envs:      map[string]string{"AI_AGENT": "trae"},
			wantName:  "trae",
			wantExtra: "-",
		},
		{
			name:      "AI_AGENT with non-Trae value falls through",
			envs:      map[string]string{"AI_AGENT": "OTHER"},
			wantName:  "terminal",
			wantExtra: "-",
		},
		{
			name:      "OpenCode",
			envs:      map[string]string{"OPENCODE": "1"},
			wantName:  "opencode",
			wantExtra: "-",
		},
		{
			name:      "Codex Desktop",
			envs:      map[string]string{"CODEX_CI": "1", "CODEX_INTERNAL_ORIGINATOR_OVERRIDE": "Codex Desktop"},
			wantName:  "codex",
			wantExtra: "Codex Desktop",
		},
		{
			name:      "Codex CLI",
			envs:      map[string]string{"CODEX_CI": "1"},
			wantName:  "codex",
			wantExtra: "-",
		},
		{
			name:      "JetBrains GoLand",
			envs:      map[string]string{"TERMINAL_EMULATOR": "JetBrains-JediTerm", "__CFBundleIdentifier": "com.jetbrains.goland"},
			wantName:  "jetbrains",
			wantExtra: "com.jetbrains.goland",
		},
		{
			name:      "JetBrains without bundle ID",
			envs:      map[string]string{"TERMINAL_EMULATOR": "JetBrains-JediTerm"},
			wantName:  "jetbrains",
			wantExtra: "-",
		},
		{
			name:      "VSCode",
			envs:      map[string]string{"TERM_PROGRAM": "vscode", "TERM_PROGRAM_VERSION": "1.90.0"},
			wantName:  "vscode",
			wantExtra: "1.90.0",
		},
		{
			name:      "Windsurf via CFBundleIdentifier",
			envs:      map[string]string{"__CFBundleIdentifier": "com.exafunction.windsurf", "TERM_PROGRAM_VERSION": "1.2.0"},
			wantName:  "windsurf",
			wantExtra: "1.2.0",
		},
		{
			name:      "Windsurf without version",
			envs:      map[string]string{"__CFBundleIdentifier": "com.exafunction.windsurf"},
			wantName:  "windsurf",
			wantExtra: "-",
		},
		{
			name:      "generic TERM_PROGRAM (iTerm)",
			envs:      map[string]string{"TERM_PROGRAM": "iTerm.app"},
			wantName:  "iterm.app",
			wantExtra: "-",
		},
		{
			name:      "no env vars at all",
			envs:      nil,
			wantName:  "terminal",
			wantExtra: "-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restore := clearEnv(t)
			defer restore()

			for k, v := range tt.envs {
				_ = os.Setenv(k, v)
			}

			got := Detect()
			assert.Equal(t, tt.wantName, got.Name, "Name mismatch")
			assert.Equal(t, tt.wantExtra, got.Extra, "Extra mismatch")
		})
	}
}

func TestDetectPriority_CursorOverVSCode(t *testing.T) {
	restore := clearEnv(t)
	defer restore()

	// Cursor sets both CURSOR_AGENT and TERM_PROGRAM=vscode
	_ = os.Setenv("CURSOR_AGENT", "1")
	_ = os.Setenv("TERM_PROGRAM", "vscode")

	got := Detect()
	assert.Equal(t, "cursor", got.Name, "Cursor should take priority over VSCode")
}

func TestDetectPriority_QoderOverVSCode(t *testing.T) {
	restore := clearEnv(t)
	defer restore()

	_ = os.Setenv("QODER_CLI", "1")
	_ = os.Setenv("QODERCLI_INTEGRATION_MODE", "quest")
	_ = os.Setenv("TERM_PROGRAM", "vscode")

	got := Detect()
	assert.Equal(t, "qoder", got.Name, "Qoder should take priority over VSCode")
}

func TestDetectPriority_AntigravityOverVSCode(t *testing.T) {
	restore := clearEnv(t)
	defer restore()

	_ = os.Setenv("ANTIGRAVITY_AGENT", "1")
	_ = os.Setenv("TERM_PROGRAM", "vscode")

	got := Detect()
	assert.Equal(t, "antigravity", got.Name, "Antigravity should take priority over VSCode")
}

func TestDetectPriority_TraeOverVSCode(t *testing.T) {
	restore := clearEnv(t)
	defer restore()

	_ = os.Setenv("AI_AGENT", "TRAE")
	_ = os.Setenv("TERM_PROGRAM", "vscode")

	got := Detect()
	assert.Equal(t, "trae", got.Name, "Trae should take priority over VSCode")
}

func TestDetectPriority_ExplicitOverrideWins(t *testing.T) {
	restore := clearEnv(t)
	defer restore()

	// Set everything — explicit override should still win
	_ = os.Setenv("GATE_CLI_AGENT", "custom")
	_ = os.Setenv("GATE_CLI_AGENT_VERSION", "9.9")
	_ = os.Setenv("CLAUDECODE", "1")
	_ = os.Setenv("CURSOR_AGENT", "1")

	got := Detect()
	assert.Equal(t, "custom", got.Name, "Explicit override should always win")
	assert.Equal(t, "9.9", got.Extra)
}

func TestBuild(t *testing.T) {
	restore := clearEnv(t)
	defer restore()

	_ = os.Setenv("CLAUDECODE", "1")
	_ = os.Setenv("CLAUDE_CODE_ENTRYPOINT", "cli")

	ua := Build("spot/order/create", "OpenAPI-Generator/7.2.40/go")

	assert.True(t, strings.HasPrefix(ua, "gate-cli/"))
	assert.Contains(t, ua, "/spot/order/create/")
	assert.Contains(t, ua, "/claude-code/cli ")
	assert.True(t, strings.HasSuffix(ua, "OpenAPI-Generator/7.2.40/go"))
}

func TestBuildDefaultEnv(t *testing.T) {
	restore := clearEnv(t)
	defer restore()

	ua := Build("wallet/balance", "OpenAPI-Generator/7.2.40/go")

	assert.Contains(t, ua, "/terminal/-")
}

func TestExtractCmdPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"gate-cli spot order create", "spot/order/create"},
		{"gate-cli futures position list", "futures/position/list"},
		{"gate-cli config init", "config/init"},
		{"gate-cli", "unknown"},
		{"", "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, ExtractCmdPath(tt.input))
		})
	}
}
