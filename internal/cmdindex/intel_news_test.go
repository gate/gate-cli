package cmdindex

import "testing"

func TestFilterInfoNewsOnly(t *testing.T) {
	t.Parallel()
	entries := []Entry{
		{Path: "info coin get-coin-info"},
		{Path: "news feed search-news"},
		{Path: "cex spot ticker"},
		{Path: "config list"},
	}
	got := FilterInfoNewsOnly(entries)
	if len(got) != 2 {
		t.Fatalf("got=%v", got)
	}
}
