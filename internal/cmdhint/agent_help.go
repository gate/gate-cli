//go:build agent

package cmdhint

import "strings"

// ShouldBlockParentHelp reports whether argv is a parent/group --help crawl in agent mode.
// Leaf help (deep paths) is allowed once after INVALID_ARGS per PRD.
func ShouldBlockParentHelp(argv []string) bool {
	if len(argv) < 2 {
		return false
	}
	args := argv[1:]
	if !argsContainHelp(args) {
		return false
	}
	pos := nonFlagPositionals(args)
	if len(pos) >= 4 {
		return false
	}
	// info/news leaves are typically gate-cli <domain> <group> <tool> (3 positionals).
	if len(pos) == 3 {
		switch strings.ToLower(pos[0]) {
		case "info", "news":
			return false
		}
	}
	// info/news shortcuts and discovery leaves: gate-cli info|news +<shortcut>|list|describe (2 positionals).
	if len(pos) == 2 {
		switch strings.ToLower(pos[0]) {
		case "info", "news":
			switch pos[1] {
			case "list", "describe":
				return false
			default:
				if strings.HasPrefix(pos[1], "+") {
					return false
				}
			}
		}
	}
	return true
}

func argsContainHelp(args []string) bool {
	for _, a := range args {
		switch a {
		case "-h", "--help", "-help":
			return true
		}
	}
	return false
}

func nonFlagPositionals(args []string) []string {
	var out []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			break
		}
		if strings.HasPrefix(a, "-") {
			if strings.Contains(a, "=") {
				continue
			}
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") && !strings.Contains(a, "=") {
				// skip value for -flag value pairs (not --flag=value)
				if len(a) == 2 || (len(a) > 2 && a[1] != '-') {
					i++
				}
			}
			continue
		}
		out = append(out, a)
	}
	return out
}
