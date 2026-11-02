package cmdutil

import (
	"strconv"
	"strings"

	"github.com/gate/gate-cli/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// ArgumentPolicy owns startup-only metadata for leaves that previously omitted
// Args. Custom positional validators, hidden commands and Intel remain owners of
// their original grammar. The command tree must be fully registered first.
type ArgumentPolicy struct{ leaves map[*cobra.Command]struct{} }

// NewArgumentPolicy installs missing leaf validators for the existing common/CEX
// grammar. Its caller retains this policy so repeated startup stays idempotent.
func NewArgumentPolicy(root *cobra.Command) *ArgumentPolicy {
	p := &ArgumentPolicy{leaves: map[*cobra.Command]struct{}{}}
	if root == nil {
		return p
	}
	var visit func(*cobra.Command)
	visit = func(c *cobra.Command) {
		if c.Hidden || c.DisableFlagParsing {
			return
		}
		if c.Runnable() && !c.HasSubCommands() && c.Args == nil {
			p.leaves[c] = struct{}{}
			// Keep rejected argv out of unknown-command suggestions, whose
			// historical agent diagnostics may include credential flags.
			c.Args = func(_ *cobra.Command, args []string) error {
				if len(args) != 0 {
					return output.InvalidArgsError("unexpected positional arguments")
				}
				return nil
			}
		}
		for _, child := range c.Commands() {
			visit(child)
		}
	}
	for _, c := range root.Commands() {
		switch c.Name() {
		case "cex", "config", "doctor", "migrate", "preflight", "version", "agent-index", "agent-search", "agent-resolve":
			visit(c)
		}
	}
	return p
}

// Normalize preserves native boolean space forms on newly strict leaves.
// Consume actual flag arity before inspecting another token: a string value
// resembling a flag, a custom positional ID, or text after -- is not a bool.
// No flag values or caller-owned argv slices are mutated during preprocessing.
func (p *ArgumentPolicy) Normalize(root *cobra.Command, args []string) []string {
	if p == nil || root == nil {
		return args
	}
	leaf, _, err := root.Find(args)
	if err != nil || leaf == nil {
		return args
	}
	if _, ok := p.leaves[leaf]; !ok {
		return args
	}
	local, inherited := leaf.Flags(), leaf.InheritedFlags()
	lookup := func(name string, short bool) *pflag.Flag {
		if short {
			if f := local.ShorthandLookup(name); f != nil {
				return f
			}
			return inherited.ShorthandLookup(name)
		}
		if f := local.Lookup(name); f != nil {
			return f
		}
		return inherited.Lookup(name)
	}
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			out = append(out, args[i:]...)
			break
		}
		if len(arg) < 2 || arg[0] != '-' || strings.Contains(arg, "=") {
			out = append(out, arg)
			continue
		}
		var flag *pflag.Flag
		requiresValue := false
		if strings.HasPrefix(arg, "--") {
			flag = lookup(arg[2:], false)
			if flag != nil {
				requiresValue = flag.NoOptDefVal == ""
			}
		} else {
			// A required short value can be attached to the cluster or occupy
			// the next token. Preserve both rather than rewriting its contents.
			for j := 1; j < len(arg); j++ {
				flag = lookup(arg[j:j+1], true)
				if flag == nil {
					break
				}
				if flag.NoOptDefVal == "" {
					requiresValue = j == len(arg)-1
					flag = nil
					break
				}
			}
		}
		if requiresValue && i+1 < len(args) {
			out = append(out, arg, args[i+1])
			i++
			continue
		}
		if flag != nil && flag.Value.Type() == "bool" && flag.NoOptDefVal != "" && flag.Name != "help" && flag.Name != "version" && i+1 < len(args) {
			if _, err := strconv.ParseBool(args[i+1]); err == nil {
				out = append(out, arg+"="+args[i+1])
				i++
				continue
			}
		}
		out = append(out, arg)
	}
	return out
}
