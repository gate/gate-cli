package toolschema

import (
	"strconv"
	"strings"
)

// flexBool is a pflag.Value for JSON-schema boolean fields. Callers pair it with
// NoOptDefVal="true" so bare "--flag" means true (cobra/pflag bool ergonomics) and
// "--flag=true|false" still works.
//
// The spaced form "--flag true|false" is normally rejected by pflag once NoOptDefVal
// is set, because pflag uses the default and never consumes the next argv token. To
// preserve backward compatibility with legacy scripts, gate-cli rewrites the spaced
// form into "--flag=value" at the argv layer for flexBool flags only; see
// internal/intelcmd.RewriteFlexBoolSpaceArgs.
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

// FlexBoolTypeName is the value returned by flexBool.Type(). It is exported so
// out-of-package code (notably internal/intelcmd.RewriteFlexBoolSpaceArgs) can match
// flexBool flags via pflag.Flag.Value.Type() without depending on the literal string,
// giving us a compile-time contract between the two packages.
const FlexBoolTypeName = "flexBool"

// Type must not be "bool" or pflag applies boolean-flag parsing and drops the
// separate "true"/"false" token (breaking "--flag true").
func (b *flexBool) Type() string { return FlexBoolTypeName }
