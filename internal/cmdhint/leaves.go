//go:build agent

package cmdhint

// Leaf describes a high-frequency intent-to-command mapping for GateAI agents.
type Leaf struct {
	Intent         string `json:"intent"`
	Command        string `json:"command"`
	RequiredPrefix string `json:"required_prefix"`
	OutputType     string `json:"output_type"`
	Risk           string `json:"risk,omitempty"`
	DefaultLimit   int    `json:"default_limit,omitempty"`
}

// AgentLeaves is the info/news cli_leaves set for GateAI agents.
// CEX high-frequency intents are out of scope here; use agent-search --domain cex or a separate catalog.
var AgentLeaves = []Leaf{
	{
		Intent:         "market_kline",
		Command:        "gate-cli info markettrend get-kline --symbol {symbol} --timeframe {timeframe} --period {period} --size {size} --format json",
		RequiredPrefix: "info",
		OutputType:     "large_json",
		Risk:           "public_read",
		DefaultLimit:   200,
	},
	{
		Intent:         "news_explain_market_move",
		Command:        "gate-cli news events explain-market-move --coin {coin} --query {query} --time-range {time_range} --format json",
		RequiredPrefix: "news",
		OutputType:     "json",
		Risk:           "public_read",
	},
	{
		Intent:         "info_coin_overview",
		Command:        "gate-cli info +coin-overview --symbol {symbol} --format json",
		RequiredPrefix: "info",
		OutputType:     "json",
		Risk:           "public_read",
	},
	{
		Intent:         "info_market_overview",
		Command:        "gate-cli info +market-overview --format json",
		RequiredPrefix: "info",
		OutputType:     "json",
		Risk:           "public_read",
	},
	{
		Intent:         "info_coin_compare",
		Command:        "gate-cli info +coin-compare --symbols {symbols} --format json",
		RequiredPrefix: "info",
		OutputType:     "json",
		Risk:           "public_read",
	},
	{
		Intent:         "info_trend_analysis",
		Command:        "gate-cli info +trend-analysis --symbol {symbol} --format json",
		RequiredPrefix: "info",
		OutputType:     "json",
		Risk:           "public_read",
	},
	{
		Intent:         "info_token_risk",
		Command:        "gate-cli info +token-risk --symbol {symbol} --format json",
		RequiredPrefix: "info",
		OutputType:     "json",
		Risk:           "public_read",
	},
	{
		Intent:         "info_token_risk_by_address",
		Command:        "gate-cli info +token-risk --address {address} --chain {chain} --format json",
		RequiredPrefix: "info",
		OutputType:     "json",
		Risk:           "public_read",
	},
	{
		Intent:         "info_address_tracker",
		Command:        "gate-cli info +address-tracker --address {address} --chain {chain} --format json",
		RequiredPrefix: "info",
		OutputType:     "json",
		Risk:           "public_read",
	},
	{
		Intent:         "info_token_onchain",
		Command:        "gate-cli info +token-onchain --symbol {symbol} --format json",
		RequiredPrefix: "info",
		OutputType:     "json",
		Risk:           "public_read",
	},
	{
		Intent:         "info_token_onchain_by_address",
		Command:        "gate-cli info +token-onchain --address {address} --chain {chain} --format json",
		RequiredPrefix: "info",
		OutputType:     "json",
		Risk:           "public_read",
	},
	{
		Intent:         "news_brief",
		Command:        "gate-cli news +brief --coin {coin} --format json",
		RequiredPrefix: "news",
		OutputType:     "json",
		Risk:           "public_read",
	},
	{
		Intent:         "news_event_explain",
		Command:        "gate-cli news +event-explain --coin {coin} --format json",
		RequiredPrefix: "news",
		OutputType:     "json",
		Risk:           "public_read",
	},
	{
		Intent:         "news_community_scan",
		Command:        "gate-cli news +community-scan --coin {coin} --format json",
		RequiredPrefix: "news",
		OutputType:     "json",
		Risk:           "public_read",
	},
	{
		Intent:         "info_coin_info",
		Command:        "gate-cli info coin get-coin-info --query {symbol} --format json",
		RequiredPrefix: "info",
		OutputType:     "json",
		Risk:           "public_read",
	},
	{
		Intent:         "info_market_snapshot",
		Command:        "gate-cli info marketsnapshot get-market-snapshot --symbol {symbol} --format json",
		RequiredPrefix: "info",
		OutputType:     "json",
		Risk:           "public_read",
	},
	{
		Intent:         "info_technical_analysis",
		Command:        "gate-cli info markettrend get-technical-analysis --symbol {symbol} --format json",
		RequiredPrefix: "info",
		OutputType:     "json",
		Risk:           "public_read",
	},
	{
		Intent:         "info_token_security",
		Command:        "gate-cli info compliance check-token-security --token {token} --format json",
		RequiredPrefix: "info",
		OutputType:     "json",
		Risk:           "public_read",
	},
	{
		Intent:         "news_search_news",
		Command:        "gate-cli news feed search-news --coin {coin} --time-range 24h --limit 20 --format json",
		RequiredPrefix: "news",
		OutputType:     "json_list",
		Risk:           "public_read",
		DefaultLimit:   20,
	},
	{
		Intent:         "news_search_x",
		Command:        "gate-cli news feed search-x --query {query} --time-range 24h --format json",
		RequiredPrefix: "news",
		OutputType:     "json",
		Risk:           "public_read",
	},
	{
		Intent:         "news_latest_events",
		Command:        "gate-cli news events get-latest-events --coin {coin} --time-range 24h --limit 20 --format json",
		RequiredPrefix: "news",
		OutputType:     "json_list",
		Risk:           "public_read",
		DefaultLimit:   20,
	},
	{
		Intent:         "news_web_search",
		Command:        "gate-cli news feed web-search --query {query} --time-range 24h --limit 5 --format json",
		RequiredPrefix: "news",
		OutputType:     "json",
		Risk:           "public_read",
		DefaultLimit:   5,
	},
	{
		Intent:         "news_social_sentiment",
		Command:        "gate-cli news feed get-social-sentiment --coin {coin} --time-range 24h --format json",
		RequiredPrefix: "news",
		OutputType:     "json",
		Risk:           "public_read",
	},
	{
		Intent:         "news_search_ugc",
		Command:        "gate-cli news feed search-ugc --coin {coin} --time-range 7d --limit 10 --format json",
		RequiredPrefix: "news",
		OutputType:     "json_list",
		Risk:           "public_read",
		DefaultLimit:   10,
	},
	{
		Intent:         "info_market_overview_tool",
		Command:        "gate-cli info marketsnapshot get-market-overview --format json",
		RequiredPrefix: "info",
		OutputType:     "json",
		Risk:           "public_read",
	},
	{
		Intent:         "news_event_detail",
		Command:        "gate-cli news events get-event-detail --event-id {event_id} --format json",
		RequiredPrefix: "news",
		OutputType:     "json",
		Risk:           "public_read",
	},
	{
		Intent:         "news_prediction_orderbook",
		Command:        "gate-cli news prediction get-market-orderbook --venue {venue} --market-id {market_id} --format json",
		RequiredPrefix: "news",
		OutputType:     "json",
		Risk:           "public_read",
	},
	{
		Intent:         "news_prediction_search_events",
		Command:        "gate-cli news prediction search-events --coin {coin} --limit 20 --format json",
		RequiredPrefix: "news",
		OutputType:     "json_list",
		Risk:           "public_read",
		DefaultLimit:   20,
	},
	{
		Intent:         "info_institutional_metrics",
		Command:        "gate-cli info marketsnapshot get-institutional-metrics --asset {asset} --channel all --limit 30 --format json",
		RequiredPrefix: "info",
		OutputType:     "json",
		Risk:           "public_read",
		DefaultLimit:   30,
	},
	{
		Intent:         "info_batch_market_snapshot",
		Command:        "gate-cli info marketsnapshot batch-market-snapshot --symbols {symbols} --format json",
		RequiredPrefix: "info",
		OutputType:     "json",
		Risk:           "public_read",
	},
	{
		Intent:         "news_exchange_announcements",
		Command:        "gate-cli news feed get-exchange-announcements --coin {coin} --time-range 7d --limit 20 --format json",
		RequiredPrefix: "news",
		OutputType:     "json_list",
		Risk:           "public_read",
		DefaultLimit:   20,
	},
}
