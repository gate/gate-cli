package intelcmd

import "testing"

func TestResolveMCPToolName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		backend string
		in      string
		want    string
	}{
		{"info", "info_coin_get_coin_info", "info_coin_get_coin_info"},
		{"info", "info coin get-coin-info", "info_coin_get_coin_info"},
		{"info", "coin get-coin-info", "info_coin_get_coin_info"},
		{"info", "Info Coin Get-Coin-Info", "info_coin_get_coin_info"},
		{"news", "news feed search-news", "news_feed_search_news"},
		{"news", "feed search-news", "news_feed_search_news"},
		{"news", "news_feed_search_news", "news_feed_search_news"},
		{"info", "unknown", "unknown"},
	}
	for _, tc := range cases {
		if got := ResolveMCPToolName(tc.backend, tc.in); got != tc.want {
			t.Fatalf("ResolveMCPToolName(%q, %q) = %q, want %q", tc.backend, tc.in, got, tc.want)
		}
	}
}
