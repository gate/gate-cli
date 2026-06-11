//go:build !agent

package agentfeature

import (
	"fmt"
	"os"
	"strings"
)

// Commands is false in default/release builds. Enable with: go build -tags agent .
const Commands = false

// RuntimeEnvEnabled reports GATE_CLI_AGENT / GATE_AI_AGENT truthy values (User-Agent override uses any non-empty value separately in useragent).
func RuntimeEnvEnabled() bool {
	switch strings.TrimSpace(os.Getenv("GATE_CLI_AGENT")) {
	case "1", "true", "yes":
		return true
	}
	switch strings.TrimSpace(os.Getenv("GATE_AI_AGENT")) {
	case "1", "true", "yes":
		return true
	}
	return false
}

// RuntimeActive is always false without -tags agent.
func RuntimeActive() bool { return false }

// DefaultMaxOutputWhenUnset returns 0 without -tags agent.
func DefaultMaxOutputWhenUnset() int64 { return 0 }

// DiscoveryCatalogHint suggests how to browse runnable leaves (release build).
func DiscoveryCatalogHint() string {
	return "gate-cli info list or gate-cli news list"
}

// DiscoveryResolveOrLeavesAction is suggested_next_action when a command is missing.
func DiscoveryResolveOrLeavesAction() string {
	return "use gate-cli info list or gate-cli news list to find leaf commands"
}

// DiscoveryResolveOrSearchAction is suggested_next_action for unknown leaf commands.
func DiscoveryResolveOrSearchAction() string {
	return "run gate-cli info -h or gate-cli news -h for command groups"
}

// TopLevelWrongNextAction follows a wrong_top_level path correction.
func TopLevelWrongNextAction() string {
	return "use the suggested command prefix; run gate-cli info list or gate-cli news list for available tools"
}

// TopLevelInvalidGroupedMessage explains an invalid top-level token.
func TopLevelInvalidGroupedMessage(token string) string {
	return fmt.Sprintf("top-level %q is not valid; use gate-cli info or gate-cli news subcommands", token)
}
