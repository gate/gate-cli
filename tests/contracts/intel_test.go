package contracts

import (
	"strings"
	"testing"

	"github.com/gate/gate-cli/internal/intelfacade"
	"github.com/gate/gate-cli/internal/toolargs"
	"github.com/gate/gate-cli/internal/toolschema"
	"github.com/stretchr/testify/require"
)

// Inventory parity makes a new or removed leaf require an explicit fixture
// decision. Validation supplements the real subprocess wire/output assertions.
func TestIntelContractFixtureIntegrity(t *testing.T) {
	seen := map[string]bool{}
	paths := map[string]bool{}
	for _, c := range IntelCases(t) {
		require.False(t, seen[c.Tool], c.Tool)
		require.False(t, paths[c.Path], c.Path)
		seen[c.Tool], paths[c.Path] = true, true
		require.Len(t, strings.Fields(c.Path), 3)
		require.NotNil(t, c.Arguments)
		require.NotNil(t, c.Reply)
		require.NotEmpty(t, c.PrettyContains)
		require.NoError(t, toolargs.ValidateForTool(c.Tool, c.Arguments), c.Path)
		schema := intelfacade.InfoBaselineInputSchema(c.Tool)
		if strings.HasPrefix(c.Tool, "news_") {
			schema = intelfacade.NewsBaselineInputSchema(c.Tool)
		}
		require.Empty(t, toolschema.MissingRequiredArguments(c.Arguments, schema), c.Path)
	}
	want := append(append([]string{}, intelfacade.InfoToolBaseline...), intelfacade.NewsToolBaseline...)
	require.Len(t, seen, 50)
	require.Len(t, intelfacade.InfoToolBaseline, 32)
	require.Len(t, intelfacade.NewsToolBaseline, 18)
	for _, name := range want {
		require.True(t, seen[name], "missing independent contract: %s", name)
	}
}

// Inclusive limits, Unicode length and timezone ordering are explicit paired
// examples, not inferred from the current implementation's return value.
func TestIntelValidationBoundaries(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range IntelBoundaries(t) {
		t.Run(c.Name, func(t *testing.T) {
			require.False(t, seen[c.Name], "duplicate boundary case")
			seen[c.Name] = true
			err := toolargs.ValidateForTool(c.Tool, toolargs.NormalizeForTool(c.Tool, c.Arguments))
			if c.Invalid {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
	require.Len(t, seen, 41)
}
