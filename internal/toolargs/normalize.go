package toolargs

import "strings"

// NormalizeForTool applies deterministic argument alias rules for specific tools.
// It preserves explicit canonical fields and only maps aliases when canonical key is empty.
func NormalizeForTool(toolName string, arguments map[string]interface{}) map[string]interface{} {
	if arguments == nil {
		return map[string]interface{}{}
	}
	out := make(map[string]interface{}, len(arguments))
	for k, v := range arguments {
		out[k] = v
	}
	for _, rule := range aliasRules[toolName] {
		if canonical, ok := out[rule.ToKey]; ok && !isEmptyValue(canonical) {
			delete(out, rule.FromKey)
			continue
		}
		if alias, ok := out[rule.FromKey]; ok && !isEmptyValue(alias) {
			if rule.Transform != nil {
				out[rule.ToKey] = rule.Transform(alias)
			} else {
				out[rule.ToKey] = alias
			}
		}
		delete(out, rule.FromKey)
	}
	return out
}

type argAliasRule struct {
	FromKey   string
	ToKey     string
	Transform func(interface{}) interface{}
}

var aliasRules = map[string][]argAliasRule{
	// Historical convenience alias kept for CLI ergonomics.
	"info_coin_get_coin_info": {
		{FromKey: "symbol", ToKey: "query"},
	},
	"info_markettrend_get_kline": {
		{FromKey: "interval", ToKey: "timeframe"},
	},
	"info_marketdetail_get_kline": {
		{FromKey: "interval", ToKey: "timeframe"},
	},
	"info_markettrend_get_indicator_history": {
		{FromKey: "indicator", ToKey: "indicators", Transform: wrapStringAsSlice},
	},
	"info_onchain_get_token_onchain": {
		{FromKey: "symbol", ToKey: "token"},
	},
	"info_compliance_check_token_security": {
		{FromKey: "symbol", ToKey: "token"},
	},
	"info_platformmetrics_get_platform_info": {
		{FromKey: "platform", ToKey: "platform_name"},
	},
	"info_platformmetrics_get_platform_history": {
		{FromKey: "platform", ToKey: "platform_name"},
	},
	"info_platformmetrics_get_bridge_metrics": {
		{FromKey: "bridge", ToKey: "bridge_name"},
	},
}

func wrapStringAsSlice(v interface{}) interface{} {
	if s, ok := v.(string); ok {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" {
			return []string{trimmed}
		}
		return []string{}
	}
	return v
}

func isEmptyValue(v interface{}) bool {
	switch x := v.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(x) == ""
	case []string:
		return len(x) == 0
	case []interface{}:
		return len(x) == 0
	default:
		return false
	}
}
