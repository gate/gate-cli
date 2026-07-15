package toolargs

import (
	"encoding/json"
	"errors"
	"strings"
)

// ValidateForTool applies static argument rules before MCP tools/call (caller-fixable input).
// Server-side JSON Schema and business validation remain authoritative at runtime.
//
// Covered tools (local 400 / INVALID_ARGUMENTS, no MCP round-trip):
//   - info_compliance_check_token_security: token XOR address
//   - info_platformmetrics_get_platform_history: platform_name XOR exchange_slug
//   - info_marketsnapshot_get_institutional_metrics: asset/channel/date/limit bounds
//   - info_platformmetrics_get_stablecoin_info: scope/sections/date cross-field rules for extension sections
//   - info_platformmetrics_get_exchange_reserves: include_history/history_window/asset vs scope
//   - info_platformmetrics_get_platform_info: include_oi_symbol_detail/oi_symbol_limit vs scope
//   - info_platformmetrics_get_cex_orderbook_depth: symbol required; market_type/data_scope/limit bounds
//   - info_marketsnapshot_batch_market_snapshot: symbols required, max 20
//   - info_markettrend_get_kline: size/limit max 500 (agent stdout guard; baseline default 200)
//   - news_feed_search_news: query or coin, time_range enum, limit max 100
//   - news_feed_search_x: handles XOR, time_range enum
//   - news_feed_search_ugc: query/coin, enums, limit max 50
//   - news_feed_web_search: query, time_range, limit max 10
//   - news_events_get_latest_events: time_range vs start/end, limit max
//   - news_events_get_event_detail, news_events_explain_market_move
//   - news_events_get_market_move_report, news_events_list_market_move_reports
//   - news_prediction_get_volume_delta_ranking, get_fastest_rising_ranking: date_utc, venue, status, limit
//   - news_prediction_search_events, get_market_orderbook, get_event_signal
//   - info_coin_get_coin_info: query or symbol; size/limit caps
//   - info_marketsnapshot_get_market_snapshot: symbol required
//   - news_feed_get_exchange_announcements: at least one filter; limit max 100
//   - news_feed_get_social_sentiment: time_range enum
//   - news_feed_get_mention_burst, news_feed_get_hot_topics: coin, fixed window, platform enum, topic limit
//   - info_markettrend_get_indicator_history, info_marketdetail_get_orderbook/recent_trades
//   - info_coin_search_coins, info_platformmetrics_search_platforms
//   - info_onchain_* (address/tx_hash/token), info_macro_* (indicator, calendar dates)
//   - info_platformmetrics defi/bridge/yield/liquidation/chain_activity, info_coin_get_coin_rankings
func ValidateForTool(toolName string, arguments map[string]interface{}) error {
	if arguments == nil {
		arguments = map[string]interface{}{}
	}
	switch toolName {
	case "info_compliance_check_token_security":
		hasToken := nonEmptyStringArg(arguments, "token")
		hasAddress := nonEmptyStringArg(arguments, "address")
		if !hasToken && !hasAddress {
			return errors.New("missing required fields: provide exactly one of token or address")
		}
		if hasToken && hasAddress {
			return errors.New("invalid arguments: provide exactly one of token or address, not both")
		}
	case "info_platformmetrics_get_platform_history":
		if !nonEmptyStringArg(arguments, "platform_name") && !nonEmptyStringArg(arguments, "exchange_slug") {
			return errors.New("missing required fields: provide platform_name or exchange_slug (at least one)")
		}
	case "info_marketsnapshot_get_institutional_metrics":
		return validateInfoInstitutionalMetrics(arguments)
	case "info_platformmetrics_get_stablecoin_info":
		return validateInfoStablecoinInfo(arguments)
	case "info_platformmetrics_get_exchange_reserves":
		return validateInfoExchangeReserves(arguments)
	case "info_platformmetrics_get_platform_info":
		return validateInfoPlatformInfo(arguments)
	case "info_platformmetrics_get_cex_orderbook_depth":
		return validateInfoCexOrderbookDepth(arguments)
	case "info_marketsnapshot_batch_market_snapshot":
		return validateInfoBatchMarketSnapshot(arguments)
	case "info_coin_get_coin_info":
		return validateInfoCoinGetCoinInfo(arguments)
	case "info_marketsnapshot_get_market_snapshot":
		return validateInfoMarketSnapshot(arguments)
	case "info_markettrend_get_kline":
		return validateInfoMarkettrendGetKline(arguments)
	case "info_marketdetail_get_kline":
		return validateInfoMarketdetailGetKline(arguments)
	case "info_markettrend_get_indicator_history":
		return validateInfoMarkettrendGetIndicatorHistory(arguments)
	case "info_markettrend_get_technical_analysis":
		return validateInfoMarkettrendGetTechnicalAnalysis(arguments)
	case "info_marketdetail_get_orderbook":
		return validateInfoMarketdetailOrderbook(arguments)
	case "info_marketdetail_get_recent_trades":
		return validateInfoMarketdetailRecentTrades(arguments)
	case "info_coin_search_coins":
		return validateInfoCoinSearchCoins(arguments)
	case "info_platformmetrics_search_platforms":
		return validateInfoPlatformmetricsSearchPlatforms(arguments)
	case "info_onchain_get_address_info":
		return validateInfoOnchainGetAddressInfo(arguments)
	case "info_onchain_get_address_transactions":
		return validateInfoOnchainGetAddressTransactions(arguments)
	case "info_onchain_get_transaction":
		return validateInfoOnchainGetTransaction(arguments)
	case "info_onchain_get_token_onchain":
		return validateInfoOnchainGetTokenOnchain(arguments)
	case "info_macro_get_macro_indicator":
		return validateInfoMacroGetMacroIndicator(arguments)
	case "info_macro_get_economic_calendar":
		return validateInfoMacroGetEconomicCalendar(arguments)
	case "info_platformmetrics_get_defi_overview":
		return validateInfoPlatformmetricsDefiOverview(arguments)
	case "info_platformmetrics_get_bridge_metrics":
		return validateInfoPlatformmetricsBridgeMetrics(arguments)
	case "info_platformmetrics_get_yield_pools":
		return validateInfoPlatformmetricsYieldPools(arguments)
	case "info_platformmetrics_get_liquidation_heatmap":
		return validateInfoPlatformmetricsLiquidationHeatmap(arguments)
	case "info_platformmetrics_get_chain_activity":
		return validateInfoPlatformmetricsChainActivity(arguments)
	case "info_coin_get_coin_rankings":
		return validateInfoCoinGetCoinRankings(arguments)
	case "news_feed_search_news":
		return validateNewsFeedSearchNews(arguments)
	case "news_feed_search_x":
		return validateNewsFeedSearchX(arguments)
	case "news_feed_search_ugc":
		return validateNewsFeedSearchUGC(arguments)
	case "news_feed_web_search":
		return validateNewsFeedWebSearch(arguments)
	case "news_feed_get_exchange_announcements":
		return validateNewsFeedExchangeAnnouncements(arguments)
	case "news_feed_get_social_sentiment":
		return validateNewsFeedSocialSentiment(arguments)
	case "news_feed_get_mention_burst":
		return validateNewsFeedMentionBurst(arguments)
	case "news_feed_get_hot_topics":
		return validateNewsFeedHotTopics(arguments)
	case "news_events_get_latest_events":
		return validateNewsEventsGetLatestEvents(arguments)
	case "news_events_get_event_detail":
		if !nonEmptyStringArg(arguments, "event_id") {
			return errors.New("missing required field: event_id")
		}
	case "news_events_explain_market_move":
		return validateNewsEventsExplainMarketMove(arguments)
	case "news_events_get_market_move_report":
		return validateNewsEventsGetMarketMoveReport(arguments)
	case "news_events_list_market_move_reports":
		return validateNewsEventsListMarketMoveReports(arguments)
	case "news_prediction_get_volume_delta_ranking", "news_prediction_get_fastest_rising_ranking":
		return validateNewsPredictionRanking(arguments)
	case "news_prediction_search_events":
		return validateNewsPredictionSearchEvents(arguments)
	case "news_prediction_get_market_orderbook":
		return validateNewsPredictionOrderbook(arguments)
	case "news_prediction_get_event_signal":
		return validateNewsPredictionEventSignal(arguments)
	}
	return nil
}

func requireAtLeastOneString(arguments map[string]interface{}, keys []string, label string) error {
	for _, key := range keys {
		if nonEmptyStringArg(arguments, key) {
			return nil
		}
	}
	return errors.New("missing required fields: provide " + label + " (at least one)")
}

func missingRequiredStringArgs(arguments map[string]interface{}, keys ...string) []string {
	var missing []string
	for _, key := range keys {
		if !nonEmptyStringArg(arguments, key) {
			missing = append(missing, key)
		}
	}
	return missing
}

func nonEmptyStringArg(arguments map[string]interface{}, key string) bool {
	v, ok := arguments[key]
	if !ok || v == nil {
		return false
	}
	switch s := v.(type) {
	case string:
		return strings.TrimSpace(s) != ""
	case json.Number:
		return strings.TrimSpace(s.String()) != ""
	default:
		return false
	}
}
