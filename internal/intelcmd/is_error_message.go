package intelcmd

import (
	"encoding/json"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/gate/gate-cli/internal/mcpclient"
)

// maxIntelToolIsErrorMessageRunes caps stderr / JSON error text from tool payloads.
const maxIntelToolIsErrorMessageRunes = 2048

var bearerTokenRE = regexp.MustCompile(`(?i)bearer\s+\S+`)

func redactIntelToolErrorMessage(s string) string {
	return bearerTokenRE.ReplaceAllString(s, "Bearer [redacted]")
}

// messageFromIntelToolIsError extracts a short human-readable explanation from an MCP
// tools/call result when isError is true. Returns empty if nothing usable is found.
func messageFromIntelToolIsError(result *mcpclient.CallResult) string {
	if result == nil {
		return ""
	}
	if s := stringFromStructuredContent(result.StructuredContent); s != "" {
		return redactIntelToolErrorMessage(truncateIntelToolIsErrorMessage(s))
	}
	if s := stringFromContentRaw(result.ContentRaw); s != "" {
		return redactIntelToolErrorMessage(truncateIntelToolIsErrorMessage(s))
	}
	if s := stringFromRawMap(result.Raw); s != "" {
		return redactIntelToolErrorMessage(truncateIntelToolIsErrorMessage(s))
	}
	return ""
}

func truncateIntelToolIsErrorMessage(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxIntelToolIsErrorMessageRunes {
		return s
	}
	r := []rune(s)
	return string(r[:maxIntelToolIsErrorMessageRunes]) + "…"
}

func stringFromStructuredContent(sc map[string]interface{}) string {
	if sc == nil {
		return ""
	}
	for _, k := range []string{"message", "detail", "reason", "description", "error_message"} {
		if s, ok := pickTrimmedString(sc[k]); ok {
			return s
		}
	}
	if v, ok := sc["error"]; ok {
		if s := stringFromErrorValue(v); s != "" {
			return s
		}
	}
	return ""
}

func stringFromErrorValue(v interface{}) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case map[string]interface{}:
		for _, k := range []string{"message", "detail", "reason", "description"} {
			if s, ok := pickTrimmedString(t[k]); ok {
				return s
			}
		}
	}
	return ""
}

func pickTrimmedString(v interface{}) (string, bool) {
	if v == nil {
		return "", false
	}
	switch t := v.(type) {
	case string:
		s := strings.TrimSpace(t)
		return s, s != ""
	case json.Number:
		s := strings.TrimSpace(t.String())
		return s, s != ""
	default:
		return "", false
	}
}

func stringFromContentRaw(items []interface{}) string {
	if len(items) == 0 {
		return ""
	}
	var b strings.Builder
	for _, it := range items {
		m, ok := it.(map[string]interface{})
		if !ok {
			continue
		}
		txt, ok := m["text"].(string)
		if !ok {
			continue
		}
		txt = strings.TrimSpace(txt)
		if txt == "" {
			continue
		}
		var obj map[string]interface{}
		if json.Unmarshal([]byte(txt), &obj) == nil {
			if inner := stringFromStructuredContent(obj); inner != "" {
				txt = inner
			}
		}
		if b.Len() > 0 {
			_ = b.WriteByte(' ')
		}
		_, _ = b.WriteString(txt)
	}
	return strings.TrimSpace(b.String())
}

func stringFromRawMap(raw map[string]interface{}) string {
	if raw == nil {
		return ""
	}
	for _, k := range []string{"message", "detail", "reason", "error"} {
		if v, ok := raw[k]; ok {
			if s := stringFromErrorValue(v); s != "" {
				return s
			}
		}
	}
	return ""
}
