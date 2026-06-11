//go:build agent

package agentcmd_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gate/gate-cli/cmd"
	"github.com/gate/gate-cli/internal/cmdhint"
)

func TestAgentValidateDiscovery(t *testing.T) {
	report := cmdhint.ValidateAgentDiscovery(cmd.Root())
	if !report.OK {
		for _, m := range report.MCP.Mismatches {
			t.Errorf("mcp_catalog mismatch tool=%s expected=%q command=%q suggestion=%q",
				m.ToolName, m.Expected, m.Command, m.Suggestion)
		}
		for _, m := range report.Shortcuts.Mismatches {
			t.Errorf("shortcut mismatch expected=%q suggestion=%q", m.Expected, m.Suggestion)
		}
	}
	require.Equal(t, 10, len(cmdhint.InfoNewsShortcutPaths))
}

func TestDeferredAddressRiskShortcutNotRegistered(t *testing.T) {
	for _, c := range cmd.Root().Commands() {
		if c.Name() != "info" {
			continue
		}
		for _, sc := range c.Commands() {
			if sc.Name() == "+address-risk" || sc.Name() == "address-risk" {
				t.Fatal("+address-risk must stay deferred until info_compliance_check_address_risk ships")
			}
		}
	}
}
