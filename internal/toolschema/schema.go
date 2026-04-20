package toolschema

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	defaultCacheTTL = 10 * time.Minute
	cacheTTLEnvKey  = "GATE_INTEL_SCHEMA_CACHE_TTL"
	forceRefreshEnv = "GATE_INTEL_REFRESH_SCHEMA"
)

type ToolSummary struct {
	Name           string      `json:"name"`
	Description    string      `json:"description,omitempty"`
	HasInputSchema bool        `json:"has_input_schema"`
	InputSchema    interface{} `json:"input_schema,omitempty"`
}

type cachePayload struct {
	Backend   string        `json:"backend"`
	UpdatedAt time.Time     `json:"updated_at"`
	Tools     []ToolSummary `json:"tools"`
}

func cachePath(backend string) (string, error) {
	b := strings.ToLower(strings.TrimSpace(backend))
	if b != "info" && b != "news" {
		return "", fmt.Errorf("unsupported backend for schema cache: %q", backend)
	}
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "gate-cli", "intel", b+"-tools-schema.json"), nil
}

func LoadCache(backend string) ([]ToolSummary, bool, error) {
	p, err := cachePath(backend)
	if err != nil {
		return nil, false, err
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	var payload cachePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, false, fmt.Errorf("invalid schema cache: %w", err)
	}
	ttl, err := getCacheTTL()
	if err != nil {
		return payload.Tools, false, err
	}
	fresh := time.Since(payload.UpdatedAt) <= ttl
	return payload.Tools, fresh, nil
}

// ForceRefreshEnabled reports whether Intel tool schema caches should be treated as stale
// for this process. Non-empty `GATE_INTEL_REFRESH_SCHEMA` set to 1/true/yes enables refresh
// (see README Intel MCP section; CR-833).
func ForceRefreshEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(forceRefreshEnv)))
	if v == "1" || v == "true" || v == "yes" {
		return true
	}
	return false
}

// IsBackendInvoked reports whether os.Args contains the given backend name as its own token.
//
// Deprecated: argv sniffing is brittle (flags, command order). Prefer checking the active
// Cobra command. Kept for existing unit tests; do not use in new call sites. See
// specs/open-items-and-dependencies.md (Intel / MCP) and specs/cli/cli-first-mcp-technical-implementation-plan.md.
func IsBackendInvoked(backend string) bool {
	b := strings.TrimSpace(strings.ToLower(backend))
	if b == "" {
		return false
	}
	for _, arg := range os.Args[1:] {
		if strings.ToLower(strings.TrimSpace(arg)) == b {
			return true
		}
	}
	return false
}

func SaveCache(backend string, tools []ToolSummary) error {
	p, err := cachePath(backend)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	existingMode := os.FileMode(0o600)
	if st, err := os.Stat(p); err == nil {
		existingMode = st.Mode().Perm()
	}
	if prevRaw, err := os.ReadFile(p); err == nil {
		var prev cachePayload
		if json.Unmarshal(prevRaw, &prev) == nil && sameToolSummaries(prev.Tools, tools) {
			return nil
		}
	}
	payload := cachePayload{
		Backend:   strings.ToLower(backend),
		UpdatedAt: time.Now().UTC(),
		Tools:     tools,
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	// Always create the temp file with tight permissions; final mode comes from existing file or default 0o600 (CR-409).
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, p); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if st, err := os.Stat(p); err == nil && st.Mode().Perm() != existingMode {
		_ = os.Chmod(p, existingMode)
	}
	return nil
}

func sameToolSummaries(a, b []ToolSummary) bool {
	if len(a) != len(b) {
		return false
	}
	left, err := json.Marshal(a)
	if err != nil {
		return false
	}
	right, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(left) == string(right)
}

func getCacheTTL() (time.Duration, error) {
	v := strings.TrimSpace(os.Getenv(cacheTTLEnvKey))
	if v == "" {
		return defaultCacheTTL, nil
	}
	if d, err := time.ParseDuration(v); err == nil && d > 0 {
		return d, nil
	}
	seconds, err := strconv.Atoi(v)
	if err != nil || seconds <= 0 {
		return 0, fmt.Errorf("invalid %s: %q", cacheTTLEnvKey, v)
	}
	return time.Duration(seconds) * time.Second, nil
}

// IsEmptyInputSchema reports true when schema has no usable properties for flag generation.
func IsEmptyInputSchema(schemaAny interface{}) bool {
	schema, ok := schemaAny.(map[string]interface{})
	if !ok {
		return true
	}
	props, ok := schema["properties"].(map[string]interface{})
	return !ok || len(props) == 0
}

func ApplyInputSchemaFlags(cmd *cobra.Command, schemaAny interface{}) {
	schema, ok := schemaAny.(map[string]interface{})
	if !ok {
		return
	}
	props, ok := schema["properties"].(map[string]interface{})
	if !ok || len(props) == 0 {
		return
	}
	requiredSet := map[string]struct{}{}
	if reqRaw, ok := schema["required"].([]interface{}); ok {
		for _, v := range reqRaw {
			if s, ok := v.(string); ok && s != "" {
				requiredSet[s] = struct{}{}
			}
		}
	}

	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, propName := range keys {
		raw := props[propName]
		spec, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		flagName := strings.ReplaceAll(propName, "_", "-")
		if cmd.Flags().Lookup(flagName) != nil {
			continue
		}
		desc := schemaDescription(spec)
		if desc == "" {
			desc = "Argument for " + propName
		}
		required := false
		if _, ok := requiredSet[propName]; ok {
			required = true
		}
		desc = enrichUsage(desc, spec, required)
		switch schemaType(spec) {
		case "boolean":
			def, _ := spec["default"].(bool)
			fb := newFlexBool(def)
			cmd.Flags().Var(fb, flagName, desc)
			// Do not set NoOptDefVal: pflag treats it as "optional value" and will
			// never consume the next argv token, so "--flag true" leaves "true" as
			// a positional (Cobra reports unknown subcommand). Use "--flag=true" or
			// "--flag true" with flexBool (non-native bool) instead.
		case "integer":
			def := 0
			if v, ok := coerceIntDefault(spec["default"]); ok {
				def = v
			}
			cmd.Flags().Int(flagName, def, desc)
		case "number":
			def := 0.0
			if v, ok := coerceFloatDefault(spec["default"]); ok {
				def = v
			}
			cmd.Flags().Float64(flagName, def, desc)
		case "array":
			// StringArray accepts a single JSON array token (e.g. '["rsi"]'); StringSlice uses CSV and breaks on JSON.
			cmd.Flags().StringArray(flagName, nil, desc)
		default:
			def, _ := spec["default"].(string)
			cmd.Flags().String(flagName, def, desc)
		}
	}
}

func schemaType(spec map[string]interface{}) string {
	if t, ok := spec["type"].(string); ok {
		return t
	}
	if ts, ok := spec["type"].([]interface{}); ok {
		for _, v := range ts {
			s, ok := v.(string)
			if !ok || s == "" || s == "null" {
				continue
			}
			return s
		}
	}
	return "string"
}

func schemaDescription(spec map[string]interface{}) string {
	if d, ok := spec["description"].(string); ok && strings.TrimSpace(d) != "" {
		return strings.TrimSpace(d)
	}
	if t, ok := spec["title"].(string); ok && strings.TrimSpace(t) != "" {
		return strings.TrimSpace(t)
	}
	return ""
}

func enrichUsage(base string, spec map[string]interface{}, required bool) string {
	t := schemaType(spec)
	parts := []string{"type=" + t}
	if required {
		parts = append(parts, "required")
	}
	if enumVals := schemaEnum(spec); len(enumVals) > 0 {
		parts = append(parts, "enum="+strings.Join(enumVals, "|"))
	}
	if def, ok := schemaDefault(spec); ok {
		parts = append(parts, "default="+def)
	}
	for _, pair := range []struct {
		key   string
		label string
	}{
		{"minimum", "min"},
		{"maximum", "max"},
		{"minLength", "minLen"},
		{"maxLength", "maxLen"},
		{"minItems", "minItems"},
		{"maxItems", "maxItems"},
	} {
		if s := formatSchemaBound(spec[pair.key]); s != "" {
			parts = append(parts, pair.label+"="+s)
		}
	}
	if pat, ok := spec["pattern"].(string); ok && strings.TrimSpace(pat) != "" {
		parts = append(parts, "pattern="+strings.TrimSpace(pat))
	}
	return strings.TrimSpace(base) + " [" + strings.Join(parts, ", ") + "]"
}

func schemaEnum(spec map[string]interface{}) []string {
	raw, ok := spec["enum"].([]interface{})
	if !ok || len(raw) == 0 {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		out = append(out, fmt.Sprint(v))
	}
	sort.Strings(out)
	return out
}

func schemaDefault(spec map[string]interface{}) (string, bool) {
	v, ok := spec["default"]
	if !ok {
		return "", false
	}
	return fmt.Sprint(v), true
}

// coerceIntDefault accepts JSON-number shapes used in hand-written and unmarshaled schemas.
func coerceIntDefault(v interface{}) (int, bool) {
	switch x := v.(type) {
	case float64:
		return int(x), true
	case int:
		return x, true
	case int64:
		return int(x), true
	case json.Number:
		i, err := x.Int64()
		if err != nil {
			return 0, false
		}
		return int(i), true
	default:
		return 0, false
	}
}

func coerceFloatDefault(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case json.Number:
		f, err := x.Float64()
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

// formatSchemaBound renders JSON Schema numeric keywords for flag usage (integers and whole floats).
func formatSchemaBound(v interface{}) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case float64:
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'g', -1, 64)
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case json.Number:
		return strings.TrimSpace(x.String())
	default:
		return ""
	}
}
