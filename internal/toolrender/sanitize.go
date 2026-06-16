package toolrender

import "strings"

// sanitizeTerminalText strips C0 control characters except tab and newline for safe terminal output.
func sanitizeTerminalText(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '\n' || r == '\t' || (r >= 32 && r != 127) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
