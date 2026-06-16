//go:build agent

package cmdhint

import (
	"sort"
	"strings"
)

// ResolvedLeaf is a leaf match with discovery provenance for agent-resolve JSON.
type ResolvedLeaf struct {
	Leaf
	MatchSource string `json:"match_source"`
}

// ResolveAgentIntent matches curated agent-leaves first, then baseline MCP catalog leaves.
// domain limits results to info or news (aliases: intel→info); empty means both.
func ResolveAgentIntent(query string, curatedLimit, mcpLimit int, domain string) []ResolvedLeaf {
	if curatedLimit <= 0 {
		curatedLimit = 3
	}
	if mcpLimit <= 0 {
		mcpLimit = 3
	}
	domain = normalizeResolveDomain(domain)
	curated := filterLeavesByDomain(MatchAgentLeaves(query, curatedLimit), domain)
	seen := make(map[string]struct{}, curatedLimit+mcpLimit)
	var out []ResolvedLeaf
	for _, leaf := range curated {
		path := catalogCLIPath(leaf.Command)
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, ResolvedLeaf{Leaf: leaf, MatchSource: "curated"})
	}
	for _, leaf := range filterLeavesByDomain(MatchLeaves(BaselineMCPCatalog(), query, mcpLimit), domain) {
		path := catalogCLIPath(leaf.Command)
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, ResolvedLeaf{Leaf: leaf, MatchSource: "mcp_catalog"})
		if len(out) >= curatedLimit+mcpLimit {
			break
		}
	}
	return out
}

func normalizeResolveDomain(domain string) string {
	domain = strings.ToLower(strings.TrimSpace(domain))
	switch domain {
	case "", "all", "*":
		return ""
	case "intel":
		return "info"
	default:
		return domain
	}
}

func filterLeavesByDomain(leaves []Leaf, domain string) []Leaf {
	if domain == "" || len(leaves) == 0 {
		return leaves
	}
	out := make([]Leaf, 0, len(leaves))
	for _, leaf := range leaves {
		if leafMatchesResolveDomain(leaf, domain) {
			out = append(out, leaf)
		}
	}
	return out
}

func leafMatchesResolveDomain(leaf Leaf, domain string) bool {
	if strings.EqualFold(strings.TrimSpace(leaf.RequiredPrefix), domain) {
		return true
	}
	path := catalogCLIPath(leaf.Command)
	return strings.HasPrefix(path, domain+" ")
}

// MatchAgentLeaves ranks curated info/news agent-leaves against a free-text query.
func MatchAgentLeaves(query string, limit int) []Leaf {
	return MatchLeaves(AgentLeaves, query, limit)
}

// MatchLeaves ranks leaves against a free-text query (intent keywords or argv tail).
func MatchLeaves(leaves []Leaf, query string, limit int) []Leaf {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" || len(leaves) == 0 {
		return nil
	}
	tokens := expandLeafQueryTokens(strings.Fields(query))
	type scored struct {
		leaf  Leaf
		score int
	}
	var hits []scored
	for _, leaf := range leaves {
		intent := strings.ToLower(leaf.Intent)
		cmd := strings.ToLower(leaf.Command)
		score := 0
		if strings.Contains(cmd, query) {
			score += 25
		}
		for _, tok := range tokens {
			if tok == "" {
				continue
			}
			if strings.Contains(intent, tok) {
				score += 6
			}
			if strings.Contains(cmd, tok) {
				score += 10
			}
		}
		score += leafIntentBoost(leaf.Intent, tokens)
		if score > 0 {
			hits = append(hits, scored{leaf: leaf, score: score})
		}
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		return hits[i].leaf.Intent < hits[j].leaf.Intent
	})
	if limit <= 0 || limit > len(hits) {
		limit = len(hits)
	}
	out := make([]Leaf, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, hits[i].leaf)
	}
	return out
}

// QueryFromArgv builds a search phrase from non-flag positionals (skips binary name).
func QueryFromArgv(argv []string) string {
	if len(argv) < 2 {
		return ""
	}
	return strings.Join(nonFlagPositionals(argv[1:]), " ")
}

// EnrichDiagnosticWithAgentLeaf fills Suggested when empty and a leaf matches argv/query.
func EnrichDiagnosticWithAgentLeaf(d *Diagnostic, argv []string) {
	if d == nil || shouldSkipAgentLeafEnrich(d) {
		return
	}
	if d.Suggested != "" {
		return
	}
	q := QueryFromArgv(argv)
	if q == "" {
		return
	}
	leaves := MatchAgentLeaves(q, 1)
	if len(leaves) == 0 {
		return
	}
	d.Suggested = leaves[0].Command
	if d.SuggestedNextAction == "" {
		d.SuggestedNextAction = "matched intent " + leaves[0].Intent + "; substitute placeholders and run"
	}
}

func shouldSkipAgentLeafEnrich(d *Diagnostic) bool {
	if d == nil || !d.Blocked {
		return false
	}
	switch d.Reason {
	case "HELP_CRAWL_FORBIDDEN", "wrong_command_path", "wrong_top_level":
		return true
	default:
		return false
	}
}

var leafQueryAliases = map[string][]string{
	"latest":     {"events", "latest", "get-latest-events"},
	"sentiment":  {"social", "sentiment"},
	"security":   {"token-risk", "compliance", "check-token-security"},
	"announcement": {"exchange", "announcements"},
	"ugc":        {"community", "search-ugc"},
	"投票":         {"events", "explain"},
}

func expandLeafQueryTokens(tokens []string) []string {
	seen := make(map[string]struct{}, len(tokens)*2)
	var out []string
	add := func(s string) {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	for _, t := range tokens {
		add(t)
		for _, a := range leafQueryAliases[t] {
			add(a)
		}
	}
	return out
}

func leafIntentBoost(intent string, tokens []string) int {
	boosts, ok := leafIntentBoosts[intent]
	if !ok {
		return 0
	}
	score := 0
	for _, tok := range tokens {
		for _, b := range boosts {
			if tok == b {
				score += 8
			}
		}
	}
	return score
}

var leafIntentBoosts = map[string][]string{
	"news_latest_events":     {"latest", "events"},
	"news_search_news":       {"news", "search"},
	"news_brief":             {"brief", "summary"},
	"news_social_sentiment":  {"sentiment"},
	"news_search_ugc":        {"ugc", "community", "reddit"},
	"info_market_overview_tool":     {"overview", "market"},
	"info_token_security":           {"security", "risk"},
	"info_token_risk_by_address":    {"security", "risk", "address", "contract"},
	"info_token_onchain_by_address": {"onchain", "holder", "address", "contract"},
	"info_institutional_metrics":    {"institutional", "metrics"},
	"info_batch_market_snapshot":    {"batch", "symbols"},
	"news_explain_market_move":      {"explain", "move"},
	"news_exchange_announcements":   {"announcement", "announcements"},
	"news_event_detail":             {"event", "detail", "event_id"},
	"news_prediction_orderbook":     {"orderbook", "polymarket"},
	"news_prediction_search_events": {"prediction", "search"},
}
