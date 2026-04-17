package migration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var gateMarkers = []string{
	"gate-info",
	"gate-news",
}

// gateTokenHints derives JSON/string hints from gateMarkers (CR-832).
var gateTokenHints = buildGateTokenHints(gateMarkers)

func buildGateTokenHints(markers []string) []string {
	out := make([]string, 0, len(markers)*2)
	for _, m := range markers {
		out = append(out, `"`+m+`"`, m)
	}
	return out
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
	// Warnings are non-fatal notes (e.g. symlink followed within home; CR-1009).
	Warnings []string `json:"warnings,omitempty"`
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

// subpathWithinHome reports whether target is under home. Both paths are cleaned and
// EvalSymlinks so macOS /var vs /private/var does not falsely reject valid configs (CR-1009).
func subpathWithinHome(home, target string) bool {
	homeRes, err := filepath.EvalSymlinks(home)
	if err != nil {
		homeRes = filepath.Clean(home)
	} else {
		homeRes = filepath.Clean(homeRes)
	}
	targetRes, err := filepath.EvalSymlinks(target)
	if err != nil {
		targetRes = filepath.Clean(target)
	} else {
		targetRes = filepath.Clean(targetRes)
	}
	rel, err := filepath.Rel(homeRes, targetRes)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func (s *Scanner) scanProvider(providerID string, paths []string) ProviderScan {
	result := ProviderScan{
		ProviderID:   providerID,
		FilesChecked: paths,
		Entries:      make([]Entry, 0),
	}
	pathErrors := make([]string, 0)
	warnings := make([]string, 0)
	for _, p := range paths {
		info, err := os.Lstat(p)
		if err != nil {
			if !os.IsNotExist(err) {
				pathErrors = append(pathErrors, err.Error())
			}
			continue
		}
		openPath := p
		if info.Mode()&os.ModeSymlink != 0 {
			resolved, err := filepath.EvalSymlinks(p)
			if err != nil {
				pathErrors = append(pathErrors, fmt.Sprintf("symlink %q: %v", p, err))
				continue
			}
			resolved = filepath.Clean(resolved)
			if !subpathWithinHome(s.home, resolved) {
				pathErrors = append(pathErrors, fmt.Sprintf("symlink target outside home (refused): %q -> %q", p, resolved))
				continue
			}
			openPath = resolved
			warnings = append(warnings, fmt.Sprintf("followed symlink within home: %q -> %q", p, resolved))
		}
		statInfo, err := os.Stat(openPath)
		if err != nil {
			pathErrors = append(pathErrors, err.Error())
			continue
		}
		if statInfo.Size() > 1<<20 {
			pathErrors = append(pathErrors, fmt.Sprintf("config too large (>1MiB): %s", p))
			continue
		}
		f, err := os.Open(openPath)
		if err != nil {
			pathErrors = append(pathErrors, err.Error())
			continue
		}
		data, err := readFromReaderLimited(f, 1<<20)
		_ = f.Close()
		if err != nil {
			pathErrors = append(pathErrors, err.Error())
			continue
		}
		raw := string(data)
		var decoded interface{}
		jsonOK := json.Unmarshal(data, &decoded) == nil

		for _, marker := range gateMarkers {
			var match bool
			if jsonOK {
				match = visitJSONForMarker(decoded, marker)
			} else {
				match = containsGateTokenLegacy(strings.ToLower(raw), marker)
			}
			if match {
				result.Entries = append(result.Entries, Entry{
					ServerName: marker,
					FilePath:   p,
					EntryPath:  "contains:" + marker,
				})
			}
		}
	}
	result.EntriesFound = len(result.Entries)
	result.Warnings = warnings
	if result.EntriesFound == 0 && len(pathErrors) > 0 {
		result.Error = strings.Join(pathErrors, "; ")
	}
	return result
}

// containsGateTokenLegacy is the substring / hint path for non-JSON or invalid JSON files (CR-207).
// legacyTextLower must be strings.ToLower(rawContent); substring matches can false-positive in
// comments/URLs vs structured JSON (CR-411) — prefer valid JSON configs when possible.
func containsGateTokenLegacy(legacyTextLower, marker string) bool {
	if strings.Contains(legacyTextLower, `"`+marker+`"`) {
		return true
	}
	for _, hint := range gateTokenHints {
		if strings.EqualFold(hint, marker) && strings.Contains(legacyTextLower, hint) {
			return true
		}
	}
	return false
}

// containsGateToken reports whether rawContent references marker.
// legacyTextLower should be strings.ToLower(rawContent); ignored when rawContent is valid JSON (CR-207).
// Kept for unit tests; scanProvider uses the same JSON-once logic inline.
func containsGateToken(legacyTextLower, rawContent, marker string) bool {
	var decoded interface{}
	if err := json.Unmarshal([]byte(rawContent), &decoded); err == nil {
		return visitJSONForMarker(decoded, marker)
	}
	return containsGateTokenLegacy(legacyTextLower, marker)
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
