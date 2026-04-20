package toolschema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func TestGetCacheTTLDefault(t *testing.T) {
	t.Setenv(cacheTTLEnvKey, "")
	ttl, err := getCacheTTL()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ttl != defaultCacheTTL {
		t.Fatalf("expected %v, got %v", defaultCacheTTL, ttl)
	}
}

func TestGetCacheTTLDuration(t *testing.T) {
	t.Setenv(cacheTTLEnvKey, "30s")
	ttl, err := getCacheTTL()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ttl != 30*time.Second {
		t.Fatalf("expected 30s, got %v", ttl)
	}
}

func TestGetCacheTTLSeconds(t *testing.T) {
	t.Setenv(cacheTTLEnvKey, "120")
	ttl, err := getCacheTTL()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ttl != 120*time.Second {
		t.Fatalf("expected 120s, got %v", ttl)
	}
}

func TestGetCacheTTLInvalid(t *testing.T) {
	t.Setenv(cacheTTLEnvKey, "bad")
	if _, err := getCacheTTL(); err == nil {
		t.Fatal("expected error for invalid ttl")
	}
}

func TestForceRefreshEnabledFromEnv(t *testing.T) {
	t.Setenv(forceRefreshEnv, "true")
	if !ForceRefreshEnabled() {
		t.Fatal("expected force refresh enabled from env")
	}
}

func TestIsBackendInvoked(t *testing.T) {
	oldArgs := os.Args
	os.Args = []string{"gate-cli", "info", "list"}
	t.Cleanup(func() { os.Args = oldArgs })
	if !IsBackendInvoked("info") {
		t.Fatal("expected info backend invoked")
	}
	if IsBackendInvoked("news") {
		t.Fatal("did not expect news backend invoked")
	}
}

func TestApplyInputSchemaFlagsAddsRichUsage(t *testing.T) {
	cmd := &cobra.Command{Use: "x"}
	schema := map[string]interface{}{
		"required": []interface{}{"query"},
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Query text",
			},
			"mode": map[string]interface{}{
				"type":    "string",
				"default": "smart",
				"enum":    []interface{}{"fast", "smart"},
			},
			"limit": map[string]interface{}{
				"type":    "integer",
				"default": 10,
				"maximum": 100,
			},
		},
	}

	ApplyInputSchemaFlags(cmd, schema)

	query := cmd.Flags().Lookup("query")
	if query == nil {
		t.Fatal("missing query flag")
	}
	if !stringsContainsAll(query.Usage, []string{"Query text", "type=string", "required"}) {
		t.Fatalf("unexpected query usage: %s", query.Usage)
	}

	mode := cmd.Flags().Lookup("mode")
	if mode == nil {
		t.Fatal("missing mode flag")
	}
	if !stringsContainsAll(mode.Usage, []string{"type=string", "enum=fast|smart", "default=smart"}) {
		t.Fatalf("unexpected mode usage: %s", mode.Usage)
	}

	limit := cmd.Flags().Lookup("limit")
	if limit == nil {
		t.Fatal("missing limit flag")
	}
	if !stringsContainsAll(limit.Usage, []string{"type=integer", "default=10", "max=100"}) {
		t.Fatalf("unexpected limit usage: %s", limit.Usage)
	}
}

func TestApplyInputSchemaFlags_SortsFlagsDeterministically(t *testing.T) {
	cmd := &cobra.Command{Use: "x"}
	schema := map[string]interface{}{
		"properties": map[string]interface{}{
			"z_field": map[string]interface{}{"type": "string"},
			"a_field": map[string]interface{}{"type": "string"},
		},
	}
	ApplyInputSchemaFlags(cmd, schema)
	ordered := []string{}
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		ordered = append(ordered, f.Name)
	})
	if len(ordered) < 2 || ordered[0] != "a-field" || ordered[1] != "z-field" {
		t.Fatalf("expected deterministic order [a-field z-field], got %v", ordered)
	}
}

func TestSchemaTypeSupportsUnionTypeArray(t *testing.T) {
	got := schemaType(map[string]interface{}{
		"type": []interface{}{"null", "integer"},
	})
	if got != "integer" {
		t.Fatalf("expected integer, got %s", got)
	}
}

func TestIsEmptyInputSchema(t *testing.T) {
	t.Parallel()
	if !IsEmptyInputSchema(nil) {
		t.Fatal("nil should be empty")
	}
	if !IsEmptyInputSchema(map[string]interface{}{"type": "object"}) {
		t.Fatal("missing properties should be empty")
	}
	if !IsEmptyInputSchema(map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}) {
		t.Fatal("empty properties should be empty")
	}
	if IsEmptyInputSchema(map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"x": map[string]interface{}{"type": "string"},
		},
	}) {
		t.Fatal("non-empty properties should not be empty")
	}
}

func stringsContainsAll(in string, parts []string) bool {
	for _, p := range parts {
		if !strings.Contains(in, p) {
			return false
		}
	}
	return true
}

func TestSameToolSummaries(t *testing.T) {
	t.Parallel()
	base := []ToolSummary{{Name: "info_coin_get_coin_info", HasInputSchema: true}}
	same := []ToolSummary{{Name: "info_coin_get_coin_info", HasInputSchema: true}}
	diff := []ToolSummary{{Name: "info_coin_get_coin_info", HasInputSchema: false}}
	if !sameToolSummaries(base, same) {
		t.Fatal("expected equal summaries to match")
	}
	if sameToolSummaries(base, diff) {
		t.Fatal("expected different summaries to not match")
	}
}

func TestSaveCachePreservesExistingPermissions(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("HOME", cacheDir)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(cacheDir, ".cache"))

	path, err := cachePath("info")
	if err != nil {
		t.Fatalf("cache path error: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir cache dir: %v", err)
	}
	seed := cachePayload{
		Backend:   "info",
		UpdatedAt: time.Now().UTC(),
		Tools:     []ToolSummary{{Name: "seed"}},
	}
	raw, _ := json.Marshal(seed)
	if err := os.WriteFile(path, raw, 0o640); err != nil {
		t.Fatalf("seed cache write failed: %v", err)
	}
	if err := SaveCache("info", []ToolSummary{{Name: "fresh", HasInputSchema: true}}); err != nil {
		t.Fatalf("save cache failed: %v", err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat cache failed: %v", err)
	}
	if st.Mode().Perm() != 0o640 {
		t.Fatalf("expected mode 0640, got %o", st.Mode().Perm())
	}
}

func TestSaveCacheNoRewriteWhenToolsUnchanged(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("HOME", cacheDir)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(cacheDir, ".cache"))
	tools := []ToolSummary{{Name: "info_coin_get_coin_info", HasInputSchema: true}}
	if err := SaveCache("info", tools); err != nil {
		t.Fatalf("initial save failed: %v", err)
	}
	path, err := cachePath("info")
	if err != nil {
		t.Fatalf("cache path error: %v", err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read first cache failed: %v", err)
	}
	time.Sleep(20 * time.Millisecond)
	if err := SaveCache("info", tools); err != nil {
		t.Fatalf("second save failed: %v", err)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read second cache failed: %v", err)
	}
	if string(first) != string(second) {
		t.Fatal("cache should not change when tools content is unchanged")
	}
}
