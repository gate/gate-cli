package intelcmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/gate/gate-cli/internal/toolschema"
)

// RewriteFlexBoolSpaceArgs collapses "--flag VALUE" into "--flag=VALUE" for flags
// whose pflag Value Type equals toolschema.FlexBoolTypeName. flexBool flags also set
// NoOptDefVal="true" so bare "--flag" already means true; the side effect is that
// pflag refuses to consume the following argv token (the bug fixed in this change).
// Pre-rewriting the argv layer keeps backward compatibility for legacy scripts that
// wrote "--flag true|false" with a space, without re-introducing the original silent
// failure.
//
// Rules:
//   - Only flags registered on the leaf command resolved from args participate; flag
//     names from other subcommands are ignored.
//   - Only the long form "--name" (no embedded "=") is rewritten.
//   - The following argv token must be a recognized boolean literal (true|false|1|0,
//     case-insensitive). Anything else (including "--next-flag") is left untouched.
//   - Tokens after a bare "--" are never rewritten (POSIX argv terminator).
//
// Returns the (possibly rewritten) slice and a bool indicating whether any rewrite
// occurred. Callers typically pass the result to cobra.Command.SetArgs only when the
// bool is true to avoid replacing cobra's default argv source unnecessarily.
func RewriteFlexBoolSpaceArgs(root *cobra.Command, args []string) ([]string, bool) {
	if root == nil || len(args) == 0 {
		return args, false
	}
	leaf, _, err := root.Find(args)
	if err != nil || leaf == nil {
		return args, false
	}

	flexNames := map[string]struct{}{}
	leaf.Flags().VisitAll(func(f *pflag.Flag) {
		if f != nil && f.Value != nil && f.Value.Type() == toolschema.FlexBoolTypeName {
			flexNames[f.Name] = struct{}{}
		}
	})
	if len(flexNames) == 0 {
		return args, false
	}

	out := make([]string, 0, len(args))
	rewritten := false
	seenDoubleDash := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		if seenDoubleDash {
			out = append(out, a)
			continue
		}
		if a == "--" {
			seenDoubleDash = true
			out = append(out, a)
			continue
		}
		if strings.HasPrefix(a, "--") && !strings.Contains(a, "=") && i+1 < len(args) {
			name := strings.TrimPrefix(a, "--")
			if _, ok := flexNames[name]; ok && isBoolLiteral(args[i+1]) {
				out = append(out, "--"+name+"="+args[i+1])
				i++
				rewritten = true
				continue
			}
		}
		out = append(out, a)
	}
	return out, rewritten
}

// isBoolLiteral reports whether s is a value accepted by strconv.ParseBool (case-insensitive).
func isBoolLiteral(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "0", "t", "f", "true", "false":
		return true
	}
	return false
}
