package toolargs

import (
	"github.com/gate/gate-cli/internal/cmdhint"
)

// applyAgentArgumentDefaults fills safe defaults when GATE_CLI_AGENT=1 and the caller omitted bounds.
// Runs after alias normalization, before ValidateForTool.
func applyAgentArgumentDefaults(toolName string, arguments map[string]interface{}) map[string]interface{} {
	if !cmdhint.AgentModeEnabled() || arguments == nil {
		return arguments
	}
	out := arguments
	copied := false
	ensure := func(key string, val interface{}) {
		if v, ok := out[key]; ok && !isEmptyValue(v) {
			return
		}
		if !copied {
			out = copyArgMap(arguments)
			copied = true
		}
		out[key] = val
	}
	switch toolName {
	case "info_markettrend_get_kline":
		ensure("size", 200)
	case "info_marketdetail_get_kline":
		ensure("limit", 200)
	case "news_feed_search_news":
		ensure("time_range", "24h")
		ensureIntKey(&out, &copied, arguments, "limit", 20)
	case "news_feed_search_x":
		ensure("time_range", "24h")
	case "news_feed_search_ugc":
		ensure("time_range", "24h")
		ensureIntKey(&out, &copied, arguments, "limit", 10)
	case "news_feed_web_search":
		ensure("time_range", "24h")
		ensureIntKey(&out, &copied, arguments, "limit", 5)
	case "news_events_get_latest_events":
		ensure("time_range", "24h")
		ensureIntKey(&out, &copied, arguments, "limit", 20)
	case "news_events_explain_market_move":
		ensure("time_range", "2h")
	case "news_feed_get_social_sentiment":
		ensure("time_range", "24h")
	}
	return out
}

func ensureIntKey(out *map[string]interface{}, copied *bool, src map[string]interface{}, key string, def int) {
	if v, ok := (*out)[key]; ok && !isEmptyValue(v) {
		return
	}
	if !*copied {
		*out = copyArgMap(src)
		*copied = true
	}
	(*out)[key] = def
}

func copyArgMap(in map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
