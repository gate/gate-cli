package migration

import (
	"os"
	"path/filepath"
	"strings"
)

var gateMarkers = []string{
	"gate-info",
	"gate-news",
}

var gateTokenHints = []string{
	`"gate-info"`,
	`"gate-news"`,
	`gate-info`,
	`gate-news`,
}

// Entry describes one legacy Gate MCP match.
type Entry struct {
	ServerName string `json:"server_name"`
	FilePath   string `json:"file_path"`
	EntryPath  string `json:"entry_path"`
}

// ProviderScan contains scan result for one provider.
type ProviderScan struct {
	ProviderID   string   `json:"provider_id"`
	FilesChecked []string `json:"files_checked"`
	EntriesFound int      `json:"entries_found"`
	Entries      []Entry  `json:"entries,omitempty"`
	Error        string   `json:"error,omitempty"`
}

// Scanner scans known provider config files for legacy Gate MCP entries.
type Scanner struct {
	home string
}

// NewScanner constructs a scanner using current HOME.
func NewScanner() *Scanner {
	return &Scanner{home: os.Getenv("HOME")}
}

// NewScannerWithHome is used by tests.
func NewScannerWithHome(home string) *Scanner {
	return &Scanner{home: home}
}

// Scan scans selected providers. Empty list means all known providers.
func (s *Scanner) Scan(providerIDs []string) []ProviderScan {
	targets := make(map[string]struct{})
	for _, id := range providerIDs {
		v := strings.TrimSpace(strings.ToLower(id))
		if v == "" {
			continue
		}
		targets[v] = struct{}{}
	}

	all := []ProviderScan{
		s.scanProvider("codex", []string{
			filepath.Join(s.home, ".codex", "config.toml"),
			filepath.Join(s.home, ".config", "codex", "config.toml"),
		}),
		s.scanProvider("cursor", []string{
			filepath.Join(s.home, ".cursor", "mcp.json"),
			filepath.Join(s.home, ".cursor", "config.json"),
		}),
		s.scanProvider("claude_desktop", []string{
			filepath.Join(s.home, "Library", "Application Support", "Claude", "claude_desktop_config.json"),
		}),
	}
	if len(targets) == 0 {
		return all
	}

	out := make([]ProviderScan, 0, len(all))
	for _, item := range all {
		if _, ok := targets[item.ProviderID]; ok {
			out = append(out, item)
		}
	}
	return out
}

func (s *Scanner) scanProvider(providerID string, paths []string) ProviderScan {
	result := ProviderScan{
		ProviderID:   providerID,
		FilesChecked: paths,
		Entries:      make([]Entry, 0),
	}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		content := strings.ToLower(string(data))
		for _, marker := range gateMarkers {
			if containsGateToken(content, marker) {
				result.Entries = append(result.Entries, Entry{
					ServerName: marker,
					FilePath:   p,
					EntryPath:  "contains:" + marker,
				})
			}
		}
	}
	result.EntriesFound = len(result.Entries)
	return result
}

func containsGateToken(content, marker string) bool {
	if strings.Contains(content, `"`+marker+`"`) {
		return true
	}
	for _, hint := range gateTokenHints {
		if strings.Contains(content, hint) && strings.Contains(hint, marker) {
			return true
		}
	}
	return false
}
