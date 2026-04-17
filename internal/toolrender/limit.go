package toolrender

import "encoding/json"

// ApplyOutputLimit truncates oversized business payloads (the envelope "data" field) in a stable way.
// maxBytes <= 0 means unlimited. JSON stdout for Intel tool calls is the data value only, so the
// limit applies to serialised data, not wrapper fields.
func ApplyOutputLimit(envelope map[string]interface{}, maxBytes int64) map[string]interface{} {
	var measured []byte
	if data := envelope["data"]; data != nil {
		measured, _ = json.Marshal(data)
	}
	return ApplyOutputLimitWithData(envelope, maxBytes, measured)
}

// ApplyOutputLimitWithData applies output-size limit using caller-provided serialized data.
// dataJSON must be json.Marshal(envelope["data"]) when non-empty so size checks match what
// would be printed (CR-903: single measurement source for truncation decisions).
func ApplyOutputLimitWithData(envelope map[string]interface{}, maxBytes int64, dataJSON []byte) map[string]interface{} {
	if maxBytes <= 0 {
		return envelope
	}
	if len(dataJSON) == 0 || int64(len(dataJSON)) <= maxBytes {
		return envelope
	}

	out := cloneMap(envelope)
	meta := map[string]interface{}{
		"truncated":           true,
		"original_size_bytes": len(dataJSON),
		"max_output_bytes":    maxBytes,
	}
	if existing, ok := out["meta"].(map[string]interface{}); ok {
		for k, v := range existing {
			meta[k] = v
		}
	}
	out["meta"] = meta
	out["data"] = map[string]interface{}{
		"truncated": true,
		"message":   "output exceeded max bytes; data omitted",
	}
	return out
}

func cloneMap(in map[string]interface{}) map[string]interface{} {
	if in == nil {
		return nil
	}
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
