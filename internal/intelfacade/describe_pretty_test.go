package intelfacade

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDescribePrettyText_Basic(t *testing.T) {
	out := DescribePrettyText(&ToolSummary{
		Name:        "info_coin_get_coin_info",
		Description: "Coin fundamentals.",
	})
	require.NotEmpty(t, out)
	assert.Contains(t, out, "Overview")
	assert.Contains(t, out, "info_coin_get_coin_info")
	assert.Contains(t, out, "Coin fundamentals.")
	assert.Contains(t, out, "Next steps")
	assert.NotContains(t, strings.ToLower(out), "input_schema")
}

func TestDescribePrettyText_ParametersFromJSONSchema(t *testing.T) {
	out := DescribePrettyText(&ToolSummary{
		Name:           "news_feed_search_news",
		Description:    "Search headlines.",
		HasInputSchema: true,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"coin": map[string]interface{}{"type": "string"},
				"limit": map[string]interface{}{"type": "integer"},
			},
			"required": []interface{}{"coin"},
		},
	})
	assert.Contains(t, out, "Parameters")
	assert.Contains(t, out, "Required:")
	assert.Contains(t, out, "coin")
	assert.Contains(t, out, "Optional:")
	assert.Contains(t, out, "limit")
}

func TestDescribePrettyText_NilTool(t *testing.T) {
	assert.Equal(t, "", DescribePrettyText(nil))
}
