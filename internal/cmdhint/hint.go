package cmdhint

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/agentfeature"
	"github.com/gate/gate-cli/internal/cmdindex"
)

var unknownCommandRE = regexp.MustCompile(`unknown command "([^"]+)"`)

// TopLevelSuggestion maps a mistaken first argument to the required gate-cli prefix.
var TopLevelSuggestion = map[string]string{
	"earn":                "cex earn",
	"spot":                "cex spot",
	"futures":             "cex futures",
	"alpha":               "cex alpha",
	"wallet":              "cex wallet",
	"margin":              "cex margin",
	"delivery":            "cex delivery",
	"options":             "cex options",
	"intel":               "info",
	"intelligence":        "info",
	"markettrend":         "info markettrend",
	"marketsnapshot":      "info marketsnapshot",
	"marketdetail":        "info marketdetail",
	"coin":                "info coin",
	"coinanalysis":        "info coin get-coin-info",
	"kline":               "info markettrend get-kline",
	"explain-market-move": "news events explain-market-move",
	"platformmetrics":     "info platformmetrics",
	"compliance":          "info compliance",
	"macro":               "info macro",
	"onchain":             "info onchain",
	"events":              "news events",
	"prediction":          "news prediction",
	"simple-earn":         "cex earn uni",
	"news":                "news feed",
	"feed":                "news feed",
}

// pathCorrections fixes common multi-segment mistakes (substring match on argv tail).
var pathCorrections = []struct {
	wrong string
	fix   string
}{
	{"news explain-market-move", "news events explain-market-move"},
	{"news feed explain-market-move", "news events explain-market-move"},
	{"explain-market-move", "news events explain-market-move"},
	{"news explain market move", "news events explain-market-move"},
	{"earn simple-earn", "cex earn uni"},
	{"earn uni records", "cex earn uni records"},
	{"earn uni lends", "cex earn uni lends"},
	{"info coinanalysis", "info coin get-coin-info"},
	{"info kline", "info markettrend get-kline"},
	{"info get-coin-info", "info coin get-coin-info"},
	{"info get-market-snapshot", "info marketsnapshot get-market-snapshot"},
	{"news get-latest-events", "news events get-latest-events"},
	{"news search-news", "news feed search-news"},
	{"news search news", "news feed search-news"},
	{"news latest events", "news events get-latest-events"},
	{"news latest", "news events get-latest-events"},
	{"info markettrend kline", "info markettrend get-kline"},
	{"info kline", "info markettrend get-kline"},
	{"info token security", "info compliance check-token-security"},
	{"info check-token-security", "info compliance check-token-security"},
	{"info technical", "info markettrend get-technical-analysis"},
	{"info coin analysis", "info coin get-coin-info"},
	{"news search", "news feed search-news"},
	{"news explain", "news events explain-market-move"},
	{"info batch snapshot", "info marketsnapshot batch-market-snapshot"},
	{"info institutional", "info marketsnapshot get-institutional-metrics"},
	{"news get-event-detail", "news events get-event-detail"},
	{"news event-detail", "news events get-event-detail"},
	{"news prediction orderbook", "news prediction get-market-orderbook"},
	{"news orderbook", "news prediction get-market-orderbook"},
}

// flagHints suggests replacements when a known bad flag appears in argv for a domain.
var flagHints = []struct {
	domainPrefix string
	wrongFlag    string
	rightFlag    string
}{
	{"news", "--symbol", "--coin"},
	{"news feed", "--symbol", "--coin"},
	{"news events", "--symbol", "--coin"},
	{"news events explain-market-move", "--symbol", "--coin"},
	{"info coin", "--coin", "--symbol"},
	{"info marketsnapshot", "--coin", "--symbol"},
}

// Diagnostic is a machine-readable hint for agents when a CLI invocation fails early.
type Diagnostic struct {
	Blocked             bool   `json:"blocked,omitempty"`
	Reason              string `json:"reason,omitempty"`
	ErrorType           string `json:"error_type,omitempty"`
	Original            string `json:"original,omitempty"`
	Suggested           string `json:"suggested,omitempty"`
	SuggestedNextAction string `json:"suggested_next_action,omitempty"`
	Retryable           bool   `json:"retryable,omitempty"`
	Message             string `json:"message,omitempty"`
}

const cliBinaryName = "gate-cli"

// SuggestTopLevel returns a diagnostic when argv[0] is a known wrong top-level token.
func SuggestTopLevel(argv []string) *Diagnostic {
	if len(argv) < 2 {
		return nil
	}
	token := strings.ToLower(strings.TrimSpace(argv[1]))
	prefix, ok := TopLevelSuggestion[token]
	if !ok {
		return nil
	}
	original := strings.Join(argv[1:], " ")
	suggested := append([]string{cliBinaryName}, strings.Fields(prefix)...)
	if len(argv) > 2 {
		rest := argv[2:]
		// Drop duplicated segment when user already typed part of the fix (e.g. gate-cli cex earn).
		if len(suggested) > 1 && len(rest) > 0 && rest[0] == suggested[len(suggested)-1] {
			rest = rest[1:]
		}
		suggested = append(suggested, rest...)
	}
	return &Diagnostic{
		Blocked:             true,
		Reason:              "wrong_top_level",
		ErrorType:           "COMMAND_NOT_FOUND",
		Original:            original,
		Suggested:           strings.Join(suggested, " "),
		SuggestedNextAction: agentfeature.TopLevelWrongNextAction(),
		Retryable:           false,
		Message:             topLevelMessage(token, prefix),
	}
}

func topLevelMessage(token, prefix string) string {
	switch {
	case strings.HasPrefix(prefix, "cex "):
		return fmt.Sprintf("top-level %q is not valid; trading and earn commands live under gate-cli cex", token)
	case strings.HasPrefix(prefix, "info "):
		return fmt.Sprintf("top-level %q is not valid; market intelligence commands live under gate-cli info", token)
	case strings.HasPrefix(prefix, "news "):
		return fmt.Sprintf("top-level %q is not valid; news commands live under gate-cli news", token)
	default:
		return agentfeature.TopLevelInvalidGroupedMessage(token)
	}
}

// SuggestFromError augments cobra execution errors with routing hints. Pass root for fuzzy leaf search.
func SuggestFromError(argv []string, err error, root *cobra.Command) *Diagnostic {
	if err == nil {
		return nil
	}
	msg := err.Error()
	lower := strings.ToLower(msg)
	if d := suggestAuthError(argv, msg); d != nil {
		return augmentDiagnostic(d, argv)
	}
	if d := suggestPathCorrection(argv); d != nil {
		d.Message = msg
		return augmentDiagnostic(d, argv)
	}
	if strings.Contains(lower, "help_crawl_forbidden") || strings.Contains(lower, "help disabled on parent") {
		return augmentDiagnostic(&Diagnostic{
			Blocked:             true,
			Reason:              "HELP_CRAWL_FORBIDDEN",
			ErrorType:           "COMMAND_NOT_FOUND",
			SuggestedNextAction: agentfeature.DiscoveryResolveOrLeavesAction(),
			Retryable:           false,
			Message:             msg,
		}, argv)
	}
	if strings.Contains(lower, "unknown command") {
		if d := SuggestTopLevel(argv); d != nil {
			return augmentDiagnostic(d, argv)
		}
		if d := suggestUnknownCommand(argv, msg, root); d != nil {
			return augmentDiagnostic(d, argv)
		}
		return augmentDiagnostic(&Diagnostic{
			ErrorType:           "COMMAND_NOT_FOUND",
			Original:            strings.Join(argv[1:], " "),
			SuggestedNextAction: agentfeature.DiscoveryResolveOrSearchAction(),
			Retryable:           false,
			Message:             msg,
		}, argv)
	}
	if strings.Contains(lower, "unknown flag") || strings.Contains(lower, "required flag") {
		d := &Diagnostic{
			ErrorType:           "INVALID_ARGS",
			Original:            strings.Join(argv[1:], " "),
			SuggestedNextAction: "check the leaf command --help once, then fix flags",
			Retryable:           true,
			Message:             msg,
		}
		if hint := suggestUnknownFlag(root, argv, msg); hint != "" {
			d.Suggested = cliBinaryName + " " + hint
			d.SuggestedNextAction = "replace mistyped flag (see suggested command)"
		}
		return augmentDiagnostic(d, argv)
	}
	return nil
}

func suggestAuthError(argv []string, msg string) *Diagnostic {
	lower := strings.ToLower(msg)
	if !strings.Contains(lower, "api key") || !strings.Contains(lower, "secret") {
		return nil
	}
	return &Diagnostic{
		ErrorType:           "AUTH_ERROR",
		Original:            strings.Join(argv[1:], " "),
		SuggestedNextAction: "configure GATE_API_KEY/GATE_API_SECRET or run gate-cli config init; do not retry without credentials",
		Retryable:           false,
		Message:             msg,
	}
}

func suggestPathCorrection(argv []string) *Diagnostic {
	if len(argv) < 2 {
		return nil
	}
	tail := strings.ToLower(strings.Join(argv[1:], " "))
	for _, pc := range pathCorrections {
		if strings.Contains(tail, pc.wrong) {
			if strings.HasPrefix(pc.wrong, "earn") && strings.Contains(tail, "cex earn") {
				continue
			}
			suggested := strings.Replace(tail, pc.wrong, pc.fix, 1)
			return &Diagnostic{
				Blocked:             true,
				Reason:              "wrong_command_path",
				ErrorType:           "COMMAND_NOT_FOUND",
				Original:            strings.Join(argv[1:], " "),
				Suggested:           cliBinaryName + " " + suggested,
				SuggestedNextAction: "use the corrected grouped leaf command",
				Retryable:           false,
				Message:             "command path should use grouped leaves under cex/info/news",
			}
		}
	}
	return nil
}

func suggestUnknownCommand(argv []string, msg string, root *cobra.Command) *Diagnostic {
	phrase := strings.Join(argv[1:], " ")
	if m := unknownCommandRE.FindStringSubmatch(msg); len(m) == 2 {
		phrase = m[1] + " " + phrase
	}
	if root == nil {
		return nil
	}
	closest := cmdindex.ClosestPaths(root, phrase, 3)
	if len(closest) == 0 {
		return nil
	}
	return &Diagnostic{
		ErrorType:           "COMMAND_NOT_FOUND",
		Original:            strings.Join(argv[1:], " "),
		Suggested:           closest[0],
		SuggestedNextAction: "closest leaf commands: " + strings.Join(closest, "; "),
		Retryable:           false,
		Message:             msg,
	}
}

func suggestFlagCorrection(argv []string) string {
	if len(argv) < 2 {
		return ""
	}
	tail := strings.Join(argv[1:], " ")
	lower := strings.ToLower(tail)
	for _, h := range flagHints {
		if !strings.Contains(lower, h.domainPrefix) {
			continue
		}
		if strings.Contains(lower, h.wrongFlag) {
			return strings.Replace(tail, h.wrongFlag, h.rightFlag, 1)
		}
	}
	return ""
}

// PrintDiagnostic writes a human hint and one JSON line for agent parsers.
func PrintDiagnostic(w io.Writer, d *Diagnostic) {
	PrintDiagnosticWithArgv(w, d, nil)
}

// PrintDiagnosticWithArgv enriches diagnostics with agent-leaves matches when argv is known.
func PrintDiagnosticWithArgv(w io.Writer, d *Diagnostic, argv []string) {
	if d == nil || w == nil {
		return
	}
	if AgentModeEnabled() && len(argv) > 0 {
		EnrichDiagnosticWithAgentLeaf(d, argv)
	}
	if d.Suggested != "" {
		_, _ = fmt.Fprintf(w, "Hint: try %s\n", d.Suggested)
	} else if d.SuggestedNextAction != "" {
		_, _ = fmt.Fprintf(w, "Hint: %s\n", d.SuggestedNextAction)
	}
	if !agentfeature.Commands {
		return
	}
	b, err := json.Marshal(d)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(w, "gate_cli_diagnostic=%s\n", string(b))
	if AgentModeEnabled() && len(argv) > 0 {
		printAgentResolveHint(w, argv)
	}
}

func augmentDiagnostic(d *Diagnostic, argv []string) *Diagnostic {
	if d == nil {
		return nil
	}
	if AgentModeEnabled() && len(argv) > 0 {
		EnrichDiagnosticWithAgentLeaf(d, argv)
	}
	return d
}
