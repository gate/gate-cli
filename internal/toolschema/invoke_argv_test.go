package toolschema

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestRemainderAfterBackendInvoke(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		argv []string
		be   string
		want []string
	}{
		{
			name: "invoke_consecutive",
			argv: []string{"gate-cli", "news", "invoke", "--name", "news_feed_search_news", "--query", "BTC"},
			be:   "news",
			want: []string{"--name", "news_feed_search_news", "--query", "BTC"},
		},
		{
			name: "call_alias",
			argv: []string{"gate-cli", "news", "call", "--name", "news_feed_search_news", "--query", "BTC"},
			be:   "news",
			want: []string{"--name", "news_feed_search_news", "--query", "BTC"},
		},
		{
			name: "flags_between",
			argv: []string{"gate-cli", "--format", "json", "news", "--refresh-schema", "invoke", "--name", "x"},
			be:   "news",
			want: []string{"--name", "x"},
		},
		{
			name: "format_value_between",
			argv: []string{"gate-cli", "news", "--format", "json", "call", "--name", "x"},
			be:   "news",
			want: []string{"--name", "x"},
		},
		{
			name: "not_invoke",
			argv: []string{"gate-cli", "news", "list"},
			be:   "news",
			want: nil,
		},
		{
			name: "info_invoke",
			argv: []string{"gate-cli", "info", "invoke", "--name", "info_coin_get_coin_info"},
			be:   "info",
			want: []string{"--name", "info_coin_get_coin_info"},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := RemainderAfterBackendInvoke(tc.argv, tc.be)
			if len(got) != len(tc.want) {
				t.Fatalf("len got %d want %d: %#v vs %#v", len(got), len(tc.want), got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("idx %d got %q want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestAttachInvokeFlagsFromArgv(t *testing.T) {
	t.Parallel()
	cmd := &cobra.Command{Use: "invoke"}
	cmd.Flags().String("name", "", "")
	schema := map[string]interface{}{
		"properties": map[string]interface{}{
			"query": map[string]interface{}{"type": "string"},
		},
	}
	summaries := map[string]ToolSummary{
		"news_feed_search_news": {Name: "news_feed_search_news", InputSchema: schema},
	}
	argv := []string{"gate-cli", "news", "invoke", "--name", "news_feed_search_news"}
	AttachInvokeFlagsFromArgv(cmd, argv, "news", summaries)
	if cmd.Flags().Lookup("query") == nil {
		t.Fatal("expected query flag from input schema")
	}
	argvAlias := []string{"gate-cli", "news", "call", "--name", "news_feed_search_news"}
	cmd2 := &cobra.Command{Use: "invoke"}
	cmd2.Flags().String("name", "", "")
	AttachInvokeFlagsFromArgv(cmd2, argvAlias, "news", summaries)
	if cmd2.Flags().Lookup("query") == nil {
		t.Fatal("expected query flag when using legacy alias token in argv")
	}
}

func TestParseLongFlag(t *testing.T) {
	t.Parallel()
	tokens := []string{"--name", "news_feed_search_news", "--limit", "5"}
	if got := ParseLongFlag(tokens, "name"); got != "news_feed_search_news" {
		t.Fatalf("name: got %q", got)
	}
	if got := ParseLongFlag(tokens, "limit"); got != "5" {
		t.Fatalf("limit: got %q", got)
	}
	if got := ParseLongFlag([]string{"--name=foo"}, "name"); got != "foo" {
		t.Fatalf("equals: got %q", got)
	}
	if got := ParseLongFlag([]string{"--other", "x"}, "name"); got != "" {
		t.Fatalf("missing: got %q", got)
	}
}
