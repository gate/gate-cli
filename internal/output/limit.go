package output

import "encoding/json"

// TruncateDataIfNeeded replaces data with a compact placeholder when serialized size exceeds maxBytes.
func TruncateDataIfNeeded(data interface{}, maxBytes int64) (interface{}, bool) {
	if maxBytes <= 0 || data == nil {
		return data, false
	}
	b, err := json.Marshal(data)
	if err != nil || int64(len(b)) <= maxBytes {
		return data, false
	}
	return map[string]interface{}{
		"truncated":           true,
		"message":             "stdout payload exceeded --max-output-bytes; use a narrower query or higher limit",
		"original_size_bytes": len(b),
		"max_output_bytes":    maxBytes,
	}, true
}
