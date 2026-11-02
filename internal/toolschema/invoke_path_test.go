package toolschema

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// CLI paths and wire names are documented equivalents. Flag registration must
// resolve them before parsing, not only later when RunE sends the RPC.
func TestInvokeCLIPathFlagRegistration(t *testing.T) {
	for _, domain := range []string{"info", "news"} {
		method, wire := "coin get-coin-info", "info_coin_get_coin_info"
		if domain == "news" {
			method, wire = "feed search-news", "news_feed_search_news"
		}
		for _, entry := range []string{"invoke", "call"} {
			for _, name := range []string{wire, domain + " " + method, method} {
				for _, style := range []string{"separate", "equals"} {
					t.Run(domain+"/"+entry+"/"+name+"/"+style, func(t *testing.T) {
						cmd := &cobra.Command{Use: entry}
						cmd.Flags().String("name", "", "")
						argv := []string{"gate-cli", "--format", "json", domain, entry}
						if style == "equals" {
							argv = append(argv, "--name="+name)
						} else {
							argv = append(argv, "--name", name)
						}
						original := append([]string(nil), argv...)
						schemas := map[string]ToolSummary{wire: {InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"query": map[string]interface{}{"type": "string"}}}}}
						AttachInvokeFlagsFromArgv(cmd, argv, domain, schemas)
						require.NotNil(t, cmd.Flags().Lookup("query"), "documented --name form failed to register flat flags")
						require.NoError(t, cmd.ParseFlags([]string{"--query", "BTC"}))
						value, err := cmd.Flags().GetString("query")
						require.NoError(t, err)
						require.Equal(t, "BTC", value)
						require.Equal(t, original, argv, "registration must not rewrite user argv")
					})
				}
			}
		}
	}
}

// Moving the resolver must preserve its established pass-through and fallback
// behavior as well as the empty/trimmed boundary cases.
func TestToolNameResolutionBoundaries(t *testing.T) {
	for _, c := range []struct{ backend, input, want string }{
		{"info", "   ", ""},
		{"info", " unknown ", "unknown"},
		{" info ", "  Info Coin Get-Coin-Info ", "info_coin_get_coin_info"},
		{"info", "news_feed_search_news", "news_feed_search_news"},
		{"", "info_coin_get_coin_info", "info_coin_get_coin_info"},
		{"", "coin get-coin-info", "coin_get_coin_info"},
		{"", "unknown_wire_name", "unknown_wire_name"},
	} {
		require.Equal(t, c.want, ResolveMCPToolName(c.backend, c.input))
	}
}

func TestInvokeUnknownAndAbsentSchema(t *testing.T) {
	for _, argv := range [][]string{nil, {"gate-cli", "info", "list"}, {"gate-cli", "info", "invoke"}, {"gate-cli", "info", "invoke", "--name", "unknown"}, {"gate-cli", "info", "invoke", "--name="}, {"gate-cli", "info", "invoke", "--name", "   "}} {
		cmd := &cobra.Command{Use: "invoke"}
		cmd.Flags().String("name", "", "")
		// Even an accidental empty schema key must not bind business flags
		// when the user supplied no actual tool identity.
		schemas := map[string]ToolSummary{"": {InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"query": map[string]interface{}{"type": "string"}}}}}
		AttachInvokeFlagsFromArgv(cmd, argv, "info", schemas)
		require.Nil(t, cmd.Flags().Lookup("query"))
	}
	AttachInvokeFlagsFromArgv(nil, []string{"gate-cli", "info", "invoke"}, "info", nil)
}
