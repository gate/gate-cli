package migration

import (
	"encoding/json"
	"fmt"
	"io"
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
	home, _ := os.UserHomeDir()
	return &Scanner{home: home}
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

	orderedProviders := []string{"codex", "cursor", "claude_desktop"}
	providers := map[string][]string{
		"codex": {
			filepath.Join(s.home, ".codex", "config.toml"),
			filepath.Join(s.home, ".config", "codex", "config.toml"),
		},
		"cursor": {
			filepath.Join(s.home, ".cursor", "mcp.json"),
			filepath.Join(s.home, ".cursor", "config.json"),
		},
		"claude_desktop": {
			filepath.Join(s.home, "Library", "Application Support", "Claude", "claude_desktop_config.json"),
		},
	}
	if len(targets) == 0 {
		all := make([]ProviderScan, 0, len(orderedProviders))
		for _, id := range orderedProviders {
			paths := providers[id]
			all = append(all, s.scanProvider(id, paths))
		}
		return all
	}

	out := make([]ProviderScan, 0, len(targets))
	for _, id := range orderedProviders {
		if _, selected := targets[id]; !selected {
			continue
		}
		paths, ok := providers[id]
		if !ok {
			continue
		}
		out = append(out, s.scanProvider(id, paths))
	}
	return out
}

func (s *Scanner) scanProvider(providerID string, paths []string) ProviderScan {
	result := ProviderScan{
		ProviderID:   providerID,
		FilesChecked: paths,
		Entries:      make([]Entry, 0),
	}
	pathErrors := make([]string, 0)
	for _, p := range paths {
		info, err := os.Lstat(p)
		if err != nil {
			if !os.IsNotExist(err) {
				pathErrors = append(pathErrors, err.Error())
			}
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			pathErrors = append(pathErrors, fmt.Sprintf("refuse symlink config: %s", p))
			continue
		}
		if info.Size() > 1<<20 {
			pathErrors = append(pathErrors, fmt.Sprintf("config too large (>1MiB): %s", p))
			continue
		}
		f, err := os.Open(p)
		if err != nil {
			pathErrors = append(pathErrors, err.Error())
			continue
		}
		data, err := io.ReadAll(io.LimitReader(f, 1<<20))
		_ = f.Close()
		if err != nil {
			pathErrors = append(pathErrors, err.Error())
			continue
		}
		content := strings.ToLower(string(data))
		for _, marker := range gateMarkers {
			if containsGateToken(content, string(data), marker) {
				result.Entries = append(result.Entries, Entry{
					ServerName: marker,
					FilePath:   p,
					EntryPath:  "contains:" + marker,
				})
			}
		}
	}
	result.EntriesFound = len(result.Entries)
	if result.EntriesFound == 0 && len(pathErrors) > 0 {
		result.Error = strings.Join(pathErrors, "; ")
	}
	return result
}

func containsGateToken(contentLower, raw, marker string) bool {
	found, parsed := containsGateTokenStructuredJSON(raw, marker)
	if found {
		return true
	}
	if parsed {
		return false
	}
	if strings.Contains(contentLower, `"`+marker+`"`) {
		return true
	}
	for _, hint := range gateTokenHints {
		if strings.EqualFold(hint, marker) && strings.Contains(contentLower, hint) {
			return true
		}
	}
	return false
}

func containsGateTokenStructuredJSON(raw, marker string) (bool, bool) {
	var decoded interface{}
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return false, false
	}
	return visitJSONForMarker(decoded, marker), true
}

func visitJSONForMarker(v interface{}, marker string) bool {
	switch x := v.(type) {
	case map[string]interface{}:
		for k, vv := range x {
			if strings.EqualFold(strings.TrimSpace(k), marker) {
				return true
			}
			if visitJSONForMarker(vv, marker) {
				return true
			}
		}
	case []interface{}:
		for _, vv := range x {
			if visitJSONForMarker(vv, marker) {
				return true
			}
		}
	case string:
		return strings.EqualFold(strings.TrimSpace(x), marker)
	}
	return false
}
