package news

import (
	"testing"

	"github.com/gate/gate-cli/internal/toolschema"
)

func TestMergeNewsBaselineIntoFillsSearchNews(t *testing.T) {
	t.Parallel()
	out := map[string]toolschema.ToolSummary{}
	mergeNewsBaselineInto(out)
	s, ok := out["news_feed_search_news"]
	if !ok || toolschema.IsEmptyInputSchema(s.InputSchema) {
		t.Fatalf("expected baseline schema, got %#v", s)
	}
}
