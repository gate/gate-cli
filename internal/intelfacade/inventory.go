package intelfacade

// Backend tool baselines aligned with the live Info/News MCP tool lists (Info: 29 tools on public gateway as of 2026-04).
var NewsToolBaseline = []string{
	"news_feed_search_news",
	"news_feed_search_ugc",
	"news_feed_search_x",
	"news_feed_web_search",
	"news_feed_get_social_sentiment",
	"news_feed_get_exchange_announcements",
	"news_events_get_latest_events",
	"news_events_get_event_detail",
}

var InfoToolBaseline = []string{
	"info_coin_get_coin_info",
	"info_marketsnapshot_get_market_snapshot",
	"info_markettrend_get_kline",
	"info_markettrend_get_indicator_history",
	"info_markettrend_get_technical_analysis",
	"info_onchain_get_address_info",
	"info_onchain_get_address_transactions",
	"info_onchain_get_transaction",
	"info_onchain_get_token_onchain",
	"info_platformmetrics_get_platform_info",
	"info_platformmetrics_search_platforms",
	"info_platformmetrics_get_defi_overview",
	"info_platformmetrics_get_stablecoin_info",
	"info_platformmetrics_get_bridge_metrics",
	"info_platformmetrics_get_yield_pools",
	"info_platformmetrics_get_platform_history",
	"info_platformmetrics_get_exchange_reserves",
	"info_platformmetrics_get_liquidation_heatmap",
	"info_marketdetail_get_orderbook",
	"info_marketdetail_get_recent_trades",
	"info_marketdetail_get_kline",
	"info_macro_get_macro_indicator",
	"info_macro_get_economic_calendar",
	"info_macro_get_macro_summary",
	"info_coin_search_coins",
	"info_coin_get_coin_rankings",
	"info_marketsnapshot_batch_market_snapshot",
	"info_marketsnapshot_get_market_overview",
	"info_compliance_check_token_security",
}

func BaselineToolCount() int {
	return len(NewsToolBaseline) + len(InfoToolBaseline)
}
