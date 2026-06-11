package cmdindex

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// Entry is one runnable leaf command in the gate-cli tree.
type Entry struct {
	Path  string `json:"path"`
	Short string `json:"short,omitempty"`
	Score int    `json:"score,omitempty"`
}

// CollectLeaves walks the cobra tree and returns runnable leaf commands (no subcommands).
// The root command name (e.g. gate-cli) is omitted from paths.
func CollectLeaves(root *cobra.Command) []Entry {
	if root == nil {
		return nil
	}
	var out []Entry
	for _, sub := range root.Commands() {
		collectLeaves(sub, nil, &out)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func collectLeaves(cmd *cobra.Command, prefix []string, out *[]Entry) {
	if cmd == nil || cmd.Hidden {
		return
	}
	name := strings.TrimSpace(cmd.Name())
	if name == "" {
		return
	}
	path := append(append([]string{}, prefix...), name)
	if len(cmd.Commands()) == 0 && isRunnable(cmd) {
		*out = append(*out, Entry{
			Path:  strings.Join(path, " "),
			Short: strings.TrimSpace(cmd.Short),
		})
		return
	}
	for _, sub := range cmd.Commands() {
		collectLeaves(sub, path, out)
	}
}

func isRunnable(cmd *cobra.Command) bool {
	return cmd.Run != nil || cmd.RunE != nil
}

// Search ranks leaves by token overlap with query (case-insensitive).
func Search(entries []Entry, query string, limit int) []Entry {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" || len(entries) == 0 {
		return nil
	}
	tokens := expandQueryTokens(strings.Fields(query))
	if len(tokens) == 0 {
		return nil
	}
	scored := make([]Entry, 0, len(entries))
	for _, e := range entries {
		path := strings.ToLower(e.Path)
		short := strings.ToLower(e.Short)
		score := 0
		for _, tok := range tokens {
			if tok == "" {
				continue
			}
			if strings.Contains(path, tok) {
				score += 10
			}
			if strings.Contains(short, tok) {
				score += 3
			}
		}
		if strings.Contains(path, query) {
			score += 20
		}
		if score > 0 {
			e.Score = score
			scored = append(scored, e)
		}
	}
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].Score != scored[j].Score {
			return scored[i].Score > scored[j].Score
		}
		return scored[i].Path < scored[j].Path
	})
	if limit <= 0 || limit > len(scored) {
		limit = len(scored)
	}
	return scored[:limit]
}

// ClosestPaths returns up to limit command paths best matching a mistaken token or phrase.
func ClosestPaths(root *cobra.Command, phrase string, limit int) []string {
	entries := Search(CollectLeaves(root), phrase, limit)
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, cliCommandLine(e.Path))
	}
	return out
}

func cliCommandLine(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return cliBinaryName
	}
	return cliBinaryName + " " + path
}

const cliBinaryName = "gate-cli"
