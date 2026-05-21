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
//   - info_platformmetrics_get_platform_history: platform_name XOR exchange_slug
//   - info_platformmetrics_get_stablecoin_info: scope/sections/date cross-field rules for issuance_flow
//   - info_platformmetrics_get_exchange_reserves: include_history/history_window/asset vs scope
//   - info_platformmetrics_get_platform_info: include_oi_symbol_detail/oi_symbol_limit vs scope
//   - news_feed_search_ugc, news_feed_web_search
//   - news_events_get_event_detail, news_events_explain_market_move
//   - news_prediction_search_events, get_market_orderbook, get_event_signal
func ValidateForTool(toolName string, arguments map[string]interface{}) error {
	if arguments == nil {
		arguments = map[string]interface{}{}
	}
	switch toolName {
	case "info_platformmetrics_get_platform_history":
		if !nonEmptyStringArg(arguments, "platform_name") && !nonEmptyStringArg(arguments, "exchange_slug") {
			return errors.New("missing required fields: provide platform_name or exchange_slug (at least one)")
		}
	case "info_platformmetrics_get_stablecoin_info":
		return validateInfoStablecoinInfo(arguments)
	case "info_platformmetrics_get_exchange_reserves":
		return validateInfoExchangeReserves(arguments)
	case "info_platformmetrics_get_platform_info":
		return validateInfoPlatformInfo(arguments)
	case "news_feed_search_ugc":
		if err := requireAtLeastOneString(arguments, []string{"query", "coin"}, "query or coin"); err != nil {
			return err
		}
	case "news_feed_web_search":
		if !nonEmptyStringArg(arguments, "query") {
			return errors.New("missing required field: query")
		}
	case "news_events_get_event_detail":
		if !nonEmptyStringArg(arguments, "event_id") {
			return errors.New("missing required field: event_id")
		}
	case "news_events_explain_market_move":
		if missing := missingRequiredStringArgs(arguments, "query", "coin"); len(missing) > 0 {
			return errors.New("missing required fields: " + strings.Join(missing, ", "))
		}
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
