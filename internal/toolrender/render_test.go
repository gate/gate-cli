package toolrender

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gate/gate-cli/internal/mcpclient"
	"github.com/gate/gate-cli/internal/output"
)

func TestRenderCallResult_JSONMode(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	p := output.NewWithStderr(&out, &errOut, output.FormatJSON)

	err := RenderCallResult(p, "news_feed_search_news", &mcpclient.CallResult{
		Content: []mcpclient.ContentItem{
			{"type": "text", "text": `{"ok":true}`},
		},
	}, 0)
	require.NoError(t, err)
	assert.NotContains(t, out.String(), "tool_name")
	assert.NotContains(t, out.String(), "data_source")
	assert.Contains(t, out.String(), `"ok":true`)
}

func TestRenderCallResult_PrettyModeUsesSegmentedBusinessOutput(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	p := output.NewWithStderr(&out, &errOut, output.FormatPretty)

	err := RenderCallResult(p, "info_coin_get_coin_info", &mcpclient.CallResult{
		Raw: map[string]interface{}{"v": 1},
	}, 0)
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Result\n\n")
	assert.NotContains(t, out.String(), "tool_name")
	assert.NotContains(t, out.String(), "data_source")
	assert.Contains(t, out.String(), `"v": 1`)
}

func TestRenderCallResult_PrettyModeNotesSectionForParseWarnings(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	p := output.NewWithStderr(&out, &errOut, output.FormatPretty)

	err := RenderCallResult(p, "tool", &mcpclient.CallResult{
		Content: []mcpclient.ContentItem{
			{"type": "text", "text": `{"a":1}`},
			{"type": "text", "text": "plain"},
		},
	}, 0)
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Notes\n\n")
	assert.Contains(t, out.String(), "- ")
}
