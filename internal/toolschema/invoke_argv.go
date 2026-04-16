package toolschema

import (
	"strings"

	"github.com/spf13/cobra"
)

// hiddenIntelInvokeLiterals are accepted subcommand tokens between <backend> and tool flags (primary + compat alias).
var hiddenIntelInvokeLiterals = []string{"invoke", "call"}

// RemainderAfterBackendInvoke returns argv tokens after "<backend> … invoke|call …",
// skipping flags between backend and that token (e.g. news --format json invoke).
// Returns nil when the invocation is not that hidden subcommand path.
func RemainderAfterBackendInvoke(argv []string, backend string) []string {
	b := strings.ToLower(strings.TrimSpace(backend))
	if b == "" {
		return nil
	}
	for i := 0; i < len(argv); i++ {
		if strings.ToLower(strings.TrimSpace(argv[i])) != b {
			continue
		}
		tokens := argv[i+1:]
		idx := skipFlagsUntilOneOf(tokens, hiddenIntelInvokeLiterals)
		if idx < 0 {
			continue
		}
		out := tokens[idx+1:]
		if len(out) == 0 {
			return nil
		}
		return out
	}
	return nil
}

func tokenEqualsAny(tok string, literals []string) bool {
	t := strings.ToLower(strings.TrimSpace(tok))
	for _, lit := range literals {
		if t == strings.ToLower(strings.TrimSpace(lit)) {
			return true
		}
	}
	return false
}

func skipFlagsUntilOneOf(tokens []string, literals []string) int {
	j := 0
	for j < len(tokens) {
		cur := strings.TrimSpace(tokens[j])
		if tokenEqualsAny(cur, literals) {
			return j
		}
		if cur == "--" {
			return -1
		}
		if !strings.HasPrefix(cur, "-") {
			return -1
		}
		if strings.Contains(cur, "=") {
			j++
			continue
		}
		if j+1 >= len(tokens) {
			j++
			continue
		}
		next := strings.TrimSpace(tokens[j+1])
		if tokenEqualsAny(next, literals) || strings.HasPrefix(next, "-") {
			j++
			continue
		}
		j += 2
	}
	return -1
}

// ParseLongFlag returns the value for --flag or --flag=value in tokens (first match).
func ParseLongFlag(tokens []string, flag string) string {
	if len(tokens) == 0 || flag == "" {
		return ""
	}
	prefix := "--" + flag + "="
	for i := 0; i < len(tokens); i++ {
		t := strings.TrimSpace(tokens[i])
		if strings.HasPrefix(t, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(t, prefix))
		}
		if t == "--"+flag && i+1 < len(tokens) {
			return strings.TrimSpace(tokens[i+1])
		}
	}
	return ""
}

// AttachInvokeFlagsFromArgv registers flat input-schema flags on cmd when argv names a tool
// and summaries contains its schema. Reserved flags (name, params, args-json, args-file)
// must already exist on cmd; ApplyInputSchemaFlags skips name collisions.
func AttachInvokeFlagsFromArgv(cmd *cobra.Command, argv []string, backend string, summaries map[string]ToolSummary) {
	if cmd == nil || len(argv) == 0 {
		return
	}
	tail := RemainderAfterBackendInvoke(argv, backend)
	if len(tail) == 0 {
		return
	}
	name := ParseLongFlag(tail, "name")
	if name == "" {
		return
	}
	t, ok := summaries[name]
	if !ok {
		return
	}
	ApplyInputSchemaFlags(cmd, t.InputSchema)
}
