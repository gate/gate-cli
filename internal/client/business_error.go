package client

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
)

// decodeBusinessError accepts only one complete object with unambiguous string
// label/message fields. Unknown fields are ignored, never copied into diagnostics.
// The existing transport bound also applies to direct callers of this parser.
func decodeBusinessError(body []byte) (label, message string, ok bool) {
	if int64(len(body)) > maxAPIResponseBytes {
		return "", "", false
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	start, err := decoder.Token()
	if err != nil || start != json.Delim('{') {
		return "", "", false
	}
	var hasLabel, hasMessage bool
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return "", "", false
		}
		name, valid := key.(string)
		if !valid {
			return "", "", false
		}
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return "", "", false
		}
		switch strings.ToLower(name) {
		case "label":
			if hasLabel || json.Unmarshal(raw, &label) != nil || strings.TrimSpace(label) == "" {
				return "", "", false
			}
			hasLabel = true
		case "message":
			if hasMessage || json.Unmarshal(raw, &message) != nil || strings.TrimSpace(message) == "" {
				return "", "", false
			}
			hasMessage = true
		}
	}
	if end, err := decoder.Token(); err != nil || end != json.Delim('}') {
		return "", "", false
	}
	// Decode through EOF, rather than accepting the first document in a stream.
	var trailing json.RawMessage
	if decoder.Decode(&trailing) != io.EOF || !hasLabel || !hasMessage {
		return "", "", false
	}
	return label, message, true
}
