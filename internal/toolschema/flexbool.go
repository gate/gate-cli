package toolschema

import (
	"strconv"
	"strings"
)

// flexBool is a pflag.Value for JSON-schema boolean fields so that "--flag true"
// and "--flag=false" work. Native Bool flags do not consume a spaced "true" token.
// Note: do not use NoOptDefVal on this flag: pflag would always substitute the
// default and never read the following token, breaking "--flag true".
type flexBool struct {
	v bool
}

func newFlexBool(def bool) *flexBool {
	return &flexBool{v: def}
}

func (b *flexBool) Set(s string) error {
	v, err := strconv.ParseBool(strings.TrimSpace(s))
	if err != nil {
		return err
	}
	b.v = v
	return nil
}

func (b *flexBool) String() string { return strconv.FormatBool(b.v) }

// Type must not be "bool" or pflag applies boolean-flag parsing and drops the
// separate "true"/"false" token (breaking "--flag true").
func (b *flexBool) Type() string { return "flexBool" }
