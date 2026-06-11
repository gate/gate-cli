package toolrender

// AppendResultMeta adds agent-oriented hints for empty payloads and merges with freshness meta.
func AppendResultMeta(toolName string, envelope map[string]interface{}) {
	AppendFreshnessMeta(toolName, envelope)
	if envelope == nil || !isEffectivelyEmpty(envelope["data"]) {
		return
	}
	meta, _ := envelope["meta"].(map[string]interface{})
	if meta == nil {
		meta = map[string]interface{}{}
	} else {
		meta = cloneMap(meta)
	}
	meta["result_hint"] = "EMPTY_RESULT"
	meta["suggested_next_action"] = "no matching records; adjust coin/time_range/filters or answer that nothing was found (do not retry blindly)"
	envelope["meta"] = meta
}

func isEffectivelyEmpty(data interface{}) bool {
	switch x := data.(type) {
	case nil:
		return true
	case map[string]interface{}:
		if len(x) == 0 {
			return true
		}
		// Shortcut aggregates may be partial but non-empty structurally.
		for _, v := range x {
			if !isEffectivelyEmpty(v) {
				return false
			}
		}
		return true
	case []interface{}:
		return len(x) == 0
	default:
		return false
	}
}
