package oauth2

// RedactToken masks a token string for safe display in --debug output.
// Format: ****<last6> (e.g. "pkce_at_abc123xyz" → "****23xyz").
// Empty input → "".
func RedactToken(token string) string {
	if token == "" {
		return ""
	}
	if len(token) <= 6 {
		return "****"
	}
	return "****" + token[len(token)-6:]
}
