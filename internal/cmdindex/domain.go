package cmdindex

import "strings"

// FilterByDomain returns entries whose path belongs to domain.
// domain: cex|info|news|config|doctor|migrate|preflight|completion|agent-leaves|agent-search|agent-index
// Aliases: trading→cex, intel→info. Empty or "all" returns entries unchanged.
func FilterByDomain(entries []Entry, domain string) []Entry {
	domain = normalizeDomain(domain)
	if domain == "" {
		return entries
	}
	out := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if entryMatchesDomain(e.Path, domain) {
			out = append(out, e)
		}
	}
	return out
}

func normalizeDomain(domain string) string {
	domain = strings.ToLower(strings.TrimSpace(domain))
	switch domain {
	case "", "all", "*":
		return ""
	case "trading":
		return "cex"
	case "intel":
		return "info"
	default:
		return domain
	}
}

func entryMatchesDomain(path, domain string) bool {
	parts := strings.Fields(strings.TrimSpace(path))
	if len(parts) == 0 {
		return false
	}
	return parts[0] == domain
}
