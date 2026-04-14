package toolargs

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
		if canonical, ok := out[rule.ToKey]; ok && canonical != "" {
			delete(out, rule.FromKey)
			continue
		}
		if alias, ok := out[rule.FromKey]; ok && alias != "" {
			out[rule.ToKey] = alias
		}
		delete(out, rule.FromKey)
	}
	return out
}

type argAliasRule struct {
	FromKey string
	ToKey   string
}

var aliasRules = map[string][]argAliasRule{
	// Historical convenience alias kept for CLI ergonomics.
	"info_coin_get_coin_info": {
		{FromKey: "symbol", ToKey: "query"},
	},
}
