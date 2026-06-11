//go:build agent

package cmdhint

import (
	"fmt"
	"strings"

	"github.com/gate/gate-cli/internal/agentfeature"
)

// AgentResolveHint returns a machine-readable follow-up discovery command for agents.
func AgentResolveHint(query string) string {
	q := strings.TrimSpace(query)
	if q == "" {
		return agentfeature.DiscoveryCatalogHint()
	}
	if agentfeature.Commands {
		return fmt.Sprintf("gate-cli agent-resolve --query %q --format json", q)
	}
	return agentfeature.DiscoveryCatalogHint()
}

// AgentPreflightNextAction suggests discovery/doctor steps after preflight in agent mode.
func AgentPreflightNextAction(route string) string {
	switch route {
	case "BLOCK":
		return "run gate-cli doctor --format json; configure GATE_INTEL_* URLs and bearer tokens before retrying info/news tools"
	default:
		if agentfeature.Commands {
			return "use gate-cli agent-leaves --format json or agent-resolve --query <intent> for leaf commands"
		}
		return "use " + agentfeature.DiscoveryCatalogHint() + " to browse leaf commands"
	}
}

// AgentMigrateNextAction suggests next steps after migrate failures in agent mode.
func AgentMigrateNextAction(status string) string {
	switch status {
	case "fail":
		return "fix migrate failures (backup dir permissions, provider config paths); rerun gate-cli doctor --format json before info/news tools"
	case "warn":
		return "review migrate warnings; remove legacy Gate MCP entries manually if auto-migrate did not apply"
	default:
		return "gate-cli doctor --format json"
	}
}

// AgentDoctorNextAction suggests discovery after doctor failures in agent mode.
func AgentDoctorNextAction(status string) string {
	switch status {
	case "fail":
		if agentfeature.Commands {
			return "fix failing doctor checks (config, GATE_INTEL_* connectivity); then gate-cli agent-resolve --query <intent> --format json"
		}
		return "fix failing doctor checks (config, GATE_INTEL_* connectivity); then " + agentfeature.DiscoveryCatalogHint()
	case "warn":
		if agentfeature.Commands {
			return "review doctor warnings; prefer gate-cli agent-leaves --format json over --help crawl"
		}
		return "review doctor warnings; use " + agentfeature.DiscoveryCatalogHint()
	default:
		return agentfeature.DiscoveryCatalogHint()
	}
}
