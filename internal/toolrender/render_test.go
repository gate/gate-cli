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
	assert.Contains(t, out.String(), `"tool_name": "news_feed_search_news"`)
	assert.Contains(t, out.String(), `"data_source": "content"`)
	assert.Contains(t, out.String(), `"ok": true`)
}

func TestRenderCallResult_PrettyModeAlsoUsesStableEnvelope(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	p := output.NewWithStderr(&out, &errOut, output.FormatPretty)

	err := RenderCallResult(p, "info_coin_get_coin_info", &mcpclient.CallResult{
		Raw: map[string]interface{}{"v": 1},
	}, 0)
	require.NoError(t, err)
	assert.Contains(t, out.String(), `"tool_name": "info_coin_get_coin_info"`)
	assert.Contains(t, out.String(), `"data_source": "raw"`)
	assert.Contains(t, out.String(), `"v": 1`)
}
