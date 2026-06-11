package cmdindex

import "strings"

// expandQueryTokens adds domain-specific synonyms so informal agent queries still rank leaves.
func expandQueryTokens(tokens []string) []string {
	if len(tokens) == 0 {
		return nil
	}
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
		if extra, ok := querySynonyms[t]; ok {
			for _, e := range extra {
				add(e)
			}
		}
	}
	return out
}

var querySynonyms = map[string][]string{
	"redeem":    {"records", "uni", "earn"},
	"records":   {"redeem", "uni"},
	"simple":    {"uni", "earn"},
	"earn":      {"uni", "cex"},
	"lend":      {"lends", "uni"},
	"lends":     {"lend", "uni"},
	"kline":     {"markettrend", "marketdetail", "get-kline"},
	"candle":    {"kline", "markettrend"},
	"ticker":    {"market", "pair"},
	"tickers":   {"market", "alpha"},
	"alpha":     {"tickers", "market"},
	"overview":  {"coin-overview", "market-overview"},
	"brief":     {"news", "feed", "brief"},
	"latest":    {"events", "get-latest-events"},
	"sentiment": {"social"},
	"security":  {"compliance", "token-risk"},
	"explain":   {"explain-market-move", "events"},
	"move":      {"explain-market-move", "events"},
	"compare":   {"coin-compare"},
	"risk":      {"token-risk", "compliance"},
	"futures":   {"cex", "market"},
	"spot":      {"cex", "market"},
	"wallet":    {"cex", "balance"},
	"balance":   {"wallet", "account"},
	"orderbook": {"orderbook", "depth", "market"},
}
