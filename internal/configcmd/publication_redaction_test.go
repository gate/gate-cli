package configcmd

import (
	"strings"
	"testing"
)

// Arbitrary custom headers may carry credentials; displaying configuration must
// not assume that only the built-in bearer token fields are sensitive.
func TestConfigurationMasksCustomHeaders(t *testing.T) {
	for _, input := range []string{
		"intel:\n  extra_headers:\n    X-Api-Key: synthetic-header-secret\n    rule: synthetic-routing-value\nprofiles:\n  sample:\n    base_url: https://api.example.test\n",
		"intel: {extra_headers: {X-Api-Key: synthetic-header-secret, rule: synthetic-routing-value}}\n",
		"intel:\n  extra_headers: [synthetic-header-secret, synthetic-routing-value]\n",
		"headers: &h {X-Api-Key: synthetic-header-secret, rule: synthetic-routing-value}\nintel: {extra_headers: *h}\n",
		"headers: &h {X-Api-Key: synthetic-header-secret}\nintel: {extra_headers: {<<: *h}}\n",
	} {
		got, err := redact(input)
		if err != nil {
			t.Fatal(err)
		}
		for _, value := range []string{"synthetic-header-secret", "synthetic-routing-value"} {
			if strings.Contains(got, value) {
				t.Fatal("custom header value was displayed")
			}
		}
		if !strings.Contains(got, "****") {
			t.Fatal("missing redaction marker")
		}
		if strings.Contains(input, "base_url") && !strings.Contains(got, "https://api.example.test") {
			t.Fatal("non-secret configuration was lost")
		}
	}
}

// Endpoint settings may carry credentials outside the dedicated token fields.
// Default display preserves routing context, but never userinfo or query data.
func TestConfigurationMasksEndpointCredentials(t *testing.T) {
	for _, input := range []string{
		"intel: {info_mcp_url: 'https://user:synthetic-url-secret@example.invalid/mcp?key=synthetic-url-secret#synthetic-url-secret'}\n",
		"resource: 'https://example.invalid/api/v4?token=synthetic-url-secret'\n",
		"intel: {info_mcp_url: [synthetic-url-secret]}\n",
		"intel: {info_mcp_url: ''}\n",
		"endpoint: &e 'https://example.invalid/mcp?key=synthetic-url-secret'\nintel: {news_mcp_url: *e}\n",
		"intel: {info_mcp_url: 'https://user:synthetic-url-secret@example.invalid/%invalid'}\n",
	} {
		got, err := redact(input)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(got, "synthetic-url-secret") {
			t.Fatal("endpoint credentials were displayed")
		}
	}
}
