package cmdutil

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"regexp"
	"strings"
)

var decimalInput = regexp.MustCompile(`^-?[0-9]+(?:\.[0-9]+)?$`)
var tradingToken = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

// TradingDecimal parses bounded base-10 input exactly; financial values never use floats.
func TradingDecimal(name, value string, positive bool) (*big.Rat, error) {
	if len(value) > 80 || !decimalInput.MatchString(value) {
		return nil, fmt.Errorf("%s must be a decimal string of at most 80 characters", name)
	}
	n, ok := new(big.Rat).SetString(value)
	if !ok || (positive && n.Sign() <= 0) {
		return nil, fmt.Errorf("%s must be positive", name)
	}
	return n, nil
}

// TradingToken keeps identifiers literal when the SDK inserts them into URL paths.
func TradingToken(name, value string) error {
	if len(value) == 0 || len(value) > 128 || !tradingToken.MatchString(value) || value == "." || value == ".." {
		return fmt.Errorf("%s must be a non-empty identifier of at most 128 characters", name)
	}
	return nil
}

// DecodeTradingObject accepts non-nullable SDK request fields only. It rejects
// ambiguity instead of silently defaulting an execution flag or dropping intent.
func DecodeTradingObject(name, raw string, target any) error {
	if len(raw) > 65536 || !strings.HasPrefix(strings.TrimSpace(raw), "{") {
		return fmt.Errorf("%s must be a JSON object of at most 65536 bytes", name)
	}
	check := json.NewDecoder(strings.NewReader(raw))
	check.UseNumber()
	if err := checkTradingJSONValue(check, 0); err != nil {
		return fmt.Errorf("invalid %s: %w", name, err)
	}
	if _, err := check.Token(); err != io.EOF {
		return fmt.Errorf("%s must contain exactly one JSON object", name)
	}
	d := json.NewDecoder(strings.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return fmt.Errorf("invalid %s: %w", name, err)
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("%s must contain exactly one JSON object", name)
	}
	return nil
}

// Bound recursive token validation and reject duplicate keys, including case aliases
// accepted by encoding/json. In particular, reduce_only:null must never become false.
func checkTradingJSONValue(d *json.Decoder, depth int) error {
	if depth > 32 {
		return fmt.Errorf("JSON nesting exceeds 32 levels")
	}
	token, err := d.Token()
	if err != nil {
		return err
	}
	if token == nil {
		return fmt.Errorf("null fields are not supported; omit the field instead")
	}
	delimiter, compound := token.(json.Delim)
	if !compound {
		return nil
	}
	switch delimiter {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			key, err := d.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			if !ok {
				return fmt.Errorf("invalid JSON object key")
			}
			// SDK wire field names are ASCII. Reject Unicode aliases that encoding/json
			// would otherwise fold onto an existing execution flag (for example long s).
			for _, r := range name {
				if r > 127 {
					return fmt.Errorf("JSON field names must be ASCII")
				}
			}
			normalized := strings.ToLower(name)
			if seen[normalized] {
				return fmt.Errorf("duplicate JSON field %q", name)
			}
			seen[normalized] = true
			if err := checkTradingJSONValue(d, depth+1); err != nil {
				return err
			}
		}
	case '[':
		for d.More() {
			if err := checkTradingJSONValue(d, depth+1); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter")
	}
	_, err = d.Token()
	return err
}
