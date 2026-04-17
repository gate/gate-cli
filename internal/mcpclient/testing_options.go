package mcpclient

import "time"

// CacheTTLForTest overrides tools/list cache TTLs for unit tests.
func CacheTTLForTest(full, empty time.Duration) Option {
	return func(c *Client) {
		c.listCacheTTLFull = full
		c.listCacheTTLEmpty = empty
	}
}

// UserAgentForTest overrides the default MCP User-Agent (tests only).
func UserAgentForTest(ua string) Option {
	return func(c *Client) {
		c.userAgent = ua
	}
}
