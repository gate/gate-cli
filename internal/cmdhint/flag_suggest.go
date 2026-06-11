package cmdhint

import (
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var unknownFlagRE = regexp.MustCompile(`unknown flag: (--[^\s]+)`)

// suggestUnknownFlag tries static hints, then cobra flag names on the resolved command.
func suggestUnknownFlag(root *cobra.Command, argv []string, msg string) string {
	if hint := suggestFlagCorrection(argv); hint != "" {
		return hint
	}
	if root == nil {
		return ""
	}
	m := unknownFlagRE.FindStringSubmatch(msg)
	if len(m) < 2 {
		return ""
	}
	badFlag := strings.TrimSpace(m[1])
	pathArgs, _ := splitCommandPathAndFlags(argv[1:])
	if len(pathArgs) == 0 {
		return ""
	}
	target, _, err := root.Find(pathArgs)
	if err != nil || target == nil {
		return ""
	}
	names := collectFlagNames(target)
	best := closestFlagName(strings.TrimPrefix(badFlag, "--"), names)
	if best == "" {
		return ""
	}
	return replaceFlagInArgv(argv, badFlag, "--"+best)
}

func splitCommandPathAndFlags(args []string) ([]string, []string) {
	var path []string
	var flags []string
	inFlags := false
	for _, a := range args {
		if a == "--" {
			inFlags = true
			flags = append(flags, a)
			continue
		}
		if inFlags || strings.HasPrefix(a, "-") {
			inFlags = true
			flags = append(flags, a)
			continue
		}
		path = append(path, a)
	}
	return path, flags
}

func collectFlagNames(cmd *cobra.Command) []string {
	if cmd == nil {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f == nil || f.Name == "" {
			return
		}
		if _, ok := seen[f.Name]; ok {
			return
		}
		seen[f.Name] = struct{}{}
		out = append(out, f.Name)
	})
	return out
}

func closestFlagName(bad string, names []string) string {
	bad = strings.ToLower(strings.ReplaceAll(bad, "_", "-"))
	if bad == "" {
		return ""
	}
	best := ""
	bestScore := 0
	for _, n := range names {
		norm := strings.ToLower(strings.ReplaceAll(n, "_", "-"))
		score := 0
		if norm == bad {
			return n
		}
		if strings.Contains(norm, bad) || strings.Contains(bad, norm) {
			score = 5
		}
		if editDistance(norm, bad) <= 2 {
			score += 3
		}
		if score > bestScore {
			bestScore = score
			best = n
		}
	}
	if bestScore == 0 {
		return ""
	}
	return best
}

func editDistance(a, b string) int {
	if a == b {
		return 0
	}
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	dp := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		dp[j] = j
	}
	for i := 1; i <= la; i++ {
		prev := dp[0]
		dp[0] = i
		for j := 1; j <= lb; j++ {
			cur := dp[j]
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			dp[j] = min(dp[j]+1, dp[j-1]+1, prev+cost)
			prev = cur
		}
	}
	return dp[lb]
}

func replaceFlagInArgv(argv []string, oldFlag, newFlag string) string {
	if len(argv) < 2 {
		return ""
	}
	out := make([]string, len(argv))
	copy(out, argv)
	for i := 1; i < len(out); i++ {
		if out[i] == oldFlag {
			out[i] = newFlag
			break
		}
	}
	return strings.Join(out[1:], " ")
}

func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
