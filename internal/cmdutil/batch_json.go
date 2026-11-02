package cmdutil

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode"
)

// ValidateBatchJSONKeys rejects ambiguous keys after the existing SDK DTO decode.
// Unlike DecodeTradingObject it preserves legacy nulls, sizes and nesting. An
// iterative stack has memory bounded by the already-received input, not Go stack
// depth; no request is sent until the entire batch has been checked.
func ValidateBatchJSONKeys(raw string) error {
	type frame struct {
		object, key bool
		names       map[string]struct{}
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	var stack []frame
	root := false
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			if root && len(stack) == 0 {
				return nil
			}
			return fmt.Errorf("invalid --orders-json")
		}
		if err != nil {
			return fmt.Errorf("invalid --orders-json")
		}
		if d, ok := token.(json.Delim); ok && (d == '}' || d == ']') {
			if len(stack) == 0 || (stack[len(stack)-1].object != (d == '}')) {
				return fmt.Errorf("invalid --orders-json")
			}
			stack = stack[:len(stack)-1]
			continue
		}
		if len(stack) > 0 && stack[len(stack)-1].object && stack[len(stack)-1].key {
			name, ok := token.(string)
			if !ok {
				return fmt.Errorf("invalid --orders-json")
			}
			name = foldBatchJSONName(name)
			current := &stack[len(stack)-1]
			if _, exists := current.names[name]; exists {
				return fmt.Errorf("ambiguous --orders-json: duplicate object field")
			}
			current.names[name] = struct{}{}
			current.key = false
			continue
		}
		if len(stack) == 0 {
			if root {
				return fmt.Errorf("invalid --orders-json")
			}
			root = true
		} else if stack[len(stack)-1].object {
			stack[len(stack)-1].key = true
		}
		if d, ok := token.(json.Delim); ok {
			switch d {
			case '{':
				stack = append(stack, frame{object: true, key: true, names: map[string]struct{}{}})
			case '[':
				stack = append(stack, frame{})
			default:
				return fmt.Errorf("invalid --orders-json")
			}
		}
	}
}

// encoding/json matches names with Unicode simple folding, not just ToLower.
// For example, "ſize" and "size" must not silently overwrite one SDK field.
func foldBatchJSONName(name string) string {
	var out strings.Builder
	for _, r := range name {
		minimum := r
		for next := unicode.SimpleFold(r); next != r; next = unicode.SimpleFold(next) {
			if next < minimum {
				minimum = next
			}
		}
		out.WriteRune(minimum)
	}
	return out.String()
}
