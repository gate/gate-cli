//go:build !agent

package agentfeature

import (
	"strings"
	"testing"
)

func TestDiscoveryHintsOmitAgentCommandsWhenDisabled(t *testing.T) {
	t.Parallel()
	for _, hint := range []string{
		DiscoveryCatalogHint(),
		DiscoveryResolveOrLeavesAction(),
		DiscoveryResolveOrSearchAction(),
		TopLevelWrongNextAction(),
		TopLevelInvalidGroupedMessage("foo"),
	} {
		if strings.Contains(hint, "agent-") {
			t.Fatalf("hint must not reference agent-* when -tags agent is off: %q", hint)
		}
	}
}
