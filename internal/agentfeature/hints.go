//go:build agent

package agentfeature

import "fmt"

// DiscoveryCatalogHint suggests how to browse runnable leaves.
func DiscoveryCatalogHint() string {
	if Commands {
		return "gate-cli agent-leaves --format json"
	}
	return "gate-cli info list or gate-cli news list"
}

// DiscoveryResolveOrLeavesAction is suggested_next_action when help crawl is blocked or command not found.
func DiscoveryResolveOrLeavesAction() string {
	if Commands {
		return "use gate-cli agent-resolve --query <intent> or gate-cli agent-leaves --format json"
	}
	return "use gate-cli info list or gate-cli news list to find leaf commands"
}

// DiscoveryResolveOrSearchAction is suggested_next_action for unknown leaf commands.
func DiscoveryResolveOrSearchAction() string {
	if Commands {
		return "run gate-cli agent-resolve --query <intent> or gate-cli agent-search --domain info|news"
	}
	return "run gate-cli info -h or gate-cli news -h for command groups"
}

// TopLevelWrongNextAction follows a wrong_top_level path correction.
func TopLevelWrongNextAction() string {
	if Commands {
		return "use the suggested command prefix; run gate-cli agent-leaves --format json for high-frequency leaf commands"
	}
	return "use the suggested command prefix; run gate-cli info list or gate-cli news list for available tools"
}

// TopLevelInvalidGroupedMessage explains an invalid top-level token.
func TopLevelInvalidGroupedMessage(token string) string {
	if Commands {
		return fmt.Sprintf("top-level %q is not valid; use a grouped leaf command (see gate-cli agent-leaves)", token)
	}
	return fmt.Sprintf("top-level %q is not valid; use gate-cli info or gate-cli news subcommands", token)
}
