package cmdhint

// Public command behavior is documented by the bundled contracts and leaf help.
var InfoNewsShortcutPaths = []string{
	"info +coin-overview",
	"info +market-overview",
	"info +coin-compare",
	"info +trend-analysis",
	"info +token-risk",
	"info +address-tracker",
	"info +token-onchain",
	"news +brief",
	"news +event-explain",
	"news +community-scan",
}

// DeferredInfoShortcutPaths are spec-defined but not registered until MCP baseline ships.
var DeferredInfoShortcutPaths = []string{
	"info +address-risk",
}
