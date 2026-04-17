package toolschema

import (
	"os"
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
				"default": float64(10),
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
	if !stringsContainsAll(limit.Usage, []string{"type=integer", "default=10"}) {
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
