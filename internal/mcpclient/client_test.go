package mcpclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gate/gate-cli/internal/toolconfig"
)

func TestListToolsInitializeThenList(t *testing.T) {
	var methods []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "gate-cli", r.Header.Get("X-Gate-Cli-Name"))
		assert.Equal(t, "data-mcp", r.Header.Get("rule"))

		var req map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		method := req["method"].(string)
		methods = append(methods, method)

		if method == "initialize" {
			w.Header().Set("MCP-Session-Id", "sess-1")
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"1","result":{"ok":true}}`))
			return
		}

		require.Equal(t, "sess-1", r.Header.Get("MCP-Session-Id"))
		w.Header().Set("x-gate-trace-id", "trace-123")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"2","result":{"tools":[{"name":"news_feed_search_news","description":"search"}]}}`))
	}))
	defer srv.Close()

	c := New(&toolconfig.ResolvedEndpoint{
		Backend:      "news",
		BaseURL:      srv.URL,
		ExtraHeaders: map[string]string{"rule": "data-mcp"},
		Timeout:      3 * time.Second,
	}, WithDefaultGateCliNameHeader())

	tools, resp, err := c.ListTools(context.Background())
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, []string{"initialize", "tools/list"}, methods)
	require.Len(t, tools, 1)
	assert.Equal(t, "news_feed_search_news", tools[0].Name)
}

func TestListToolsJSONRPCError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		method := req["method"].(string)
		if method == "initialize" {
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"1","result":{"ok":true}}`))
			return
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"2","error":{"code":-32601,"message":"method not found"}}`))
	}))
	defer srv.Close()

	c := New(&toolconfig.ResolvedEndpoint{
		Backend:      "news",
		BaseURL:      srv.URL,
		ExtraHeaders: map[string]string{},
		Timeout:      3 * time.Second,
	})

	_, _, err := c.ListTools(context.Background())
	require.Error(t, err)
	var mcpErr *Error
	require.ErrorAs(t, err, &mcpErr)
	require.NotNil(t, mcpErr.JSONRPCCode)
	assert.Equal(t, -32601, *mcpErr.JSONRPCCode)
}

func TestDescribeToolFromList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		method := req["method"].(string)
		if method == "initialize" {
			w.Header().Set("MCP-Session-Id", "sess-1")
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"1","result":{"ok":true}}`))
			return
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"2","result":{"tools":[{"name":"info_coin_get_coin_info","description":"desc","inputSchema":{"type":"object"}}]}}`))
	}))
	defer srv.Close()

	c := New(&toolconfig.ResolvedEndpoint{
		Backend: "info",
		BaseURL: srv.URL,
		Timeout: 3 * time.Second,
	})
	tool, _, err := c.DescribeTool(context.Background(), "info_coin_get_coin_info")
	require.NoError(t, err)
	require.NotNil(t, tool)
	assert.Equal(t, "info_coin_get_coin_info", tool.Name)
}

func TestCallTool(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		method := req["method"].(string)
		if method == "initialize" {
			w.Header().Set("MCP-Session-Id", "sess-1")
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"1","result":{"ok":true}}`))
			return
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"2","result":{"content":[{"type":"text","text":"{\"ok\":true}"}],"isError":false}}`))
	}))
	defer srv.Close()

	c := New(&toolconfig.ResolvedEndpoint{
		Backend: "news",
		BaseURL: srv.URL,
		Timeout: 3 * time.Second,
	})
	result, _, err := c.CallTool(context.Background(), "news_feed_search_news", map[string]interface{}{"query": "BTC"})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)
	require.Len(t, result.ContentRaw, 1)
}

func TestCallToolPreservesRawContentItems(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		method := req["method"].(string)
		if method == "initialize" {
			w.Header().Set("MCP-Session-Id", "sess-1")
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"1","result":{"ok":true}}`))
			return
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"2","result":{"content":[{"type":"text","text":"{\"ok\":true}"},123],"isError":false}}`))
	}))
	defer srv.Close()

	c := New(&toolconfig.ResolvedEndpoint{
		Backend: "news",
		BaseURL: srv.URL,
		Timeout: 3 * time.Second,
	})
	result, _, err := c.CallTool(context.Background(), "news_feed_search_news", map[string]interface{}{"query": "BTC"})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, result.ContentRaw, 2)
}

func TestDebugLogDoesNotContainMCPWord(t *testing.T) {
	var errOut strings.Builder
	c := New(&toolconfig.ResolvedEndpoint{
		Backend: "news",
		BaseURL: "http://example.invalid",
		Timeout: 1 * time.Second,
	}, WithDebug(true))
	c.errOut = &errOut

	c.logDebug("tools/list", "1", 20*time.Millisecond, &http.Response{
		StatusCode: 200,
		Header:     http.Header{"x-gate-trace-id": []string{"trace-1"}},
	})
	assert.NotContains(t, strings.ToLower(errOut.String()), "mcp")
	assert.Contains(t, errOut.String(), "[debug]")
}

func TestVerboseTransportDiagUsesVerboseTag(t *testing.T) {
	var errOut strings.Builder
	c := New(&toolconfig.ResolvedEndpoint{
		Backend: "info",
		BaseURL: "http://example.invalid",
		Timeout: 1 * time.Second,
	}, WithTransportDiag(true, "[verbose]"))
	c.errOut = &errOut

	c.logDebug("tools/call", "7", 5*time.Millisecond, &http.Response{StatusCode: 200})
	assert.Contains(t, errOut.String(), "[verbose]")
	assert.NotContains(t, errOut.String(), "[debug]")
}

func TestCallToolTransportDiagIncludesMergedArguments(t *testing.T) {
	var errOut strings.Builder
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		method := req["method"].(string)
		if method == "initialize" {
			w.Header().Set("MCP-Session-Id", "sess-1")
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"1","result":{"ok":true}}`))
			return
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"2","result":{"content":[{"type":"text","text":"{\"ok\":true}"}]}}`))
	}))
	defer srv.Close()

	c := New(&toolconfig.ResolvedEndpoint{
		Backend: "news",
		BaseURL: srv.URL,
		Timeout: 3 * time.Second,
	}, WithTransportDiag(true, "[verbose]"))
	c.errOut = &errOut

	_, _, err := c.CallTool(context.Background(), "news_feed_search_news", map[string]interface{}{"limit": int64(10)})
	require.NoError(t, err)
	log := errOut.String()
	assert.Contains(t, log, "rpc_method=tools/call")
	assert.Contains(t, log, "tool_name=news_feed_search_news")
	assert.Contains(t, log, "arguments={\"limit\":10}")
}

func TestCallToolTransportDiagRedactsSensitiveArguments(t *testing.T) {
	var errOut strings.Builder
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		method := req["method"].(string)
		if method == "initialize" {
			w.Header().Set("MCP-Session-Id", "sess-1")
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"1","result":{"ok":true}}`))
			return
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"2","result":{"content":[]}}`))
	}))
	defer srv.Close()

	c := New(&toolconfig.ResolvedEndpoint{
		Backend: "news",
		BaseURL: srv.URL,
		Timeout: 3 * time.Second,
	}, WithTransportDiag(true, "[verbose]"))
	c.errOut = &errOut

	_, _, err := c.CallTool(context.Background(), "news_feed_search_news", map[string]interface{}{"apiToken": "secret-value"})
	require.NoError(t, err)
	log := errOut.String()
	assert.NotContains(t, log, "secret-value")
	assert.Contains(t, log, "***REDACTED***")
}

func TestTransportDiagLogsInitializeFailure(t *testing.T) {
	var errOut strings.Builder
	c := New(&toolconfig.ResolvedEndpoint{
		Backend: "news",
		BaseURL: "http://127.0.0.1:1",
		Timeout: 50 * time.Millisecond,
	}, WithTransportDiag(true, "[verbose]"))
	c.errOut = &errOut

	_, _, err := c.ListTools(context.Background())
	require.Error(t, err)
	log := errOut.String()
	assert.Contains(t, log, "[verbose]")
	assert.Contains(t, log, "rpc_method=initialize")
	assert.Contains(t, log, "transport_error=")
}

func TestExtraHeadersOverrideXGateCliName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "override-value", r.Header.Get("X-Gate-Cli-Name"))

		var req map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		method := req["method"].(string)
		if method == "initialize" {
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"1","result":{"ok":true}}`))
			return
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"2","result":{"tools":[]}}`))
	}))
	defer srv.Close()

	c := New(&toolconfig.ResolvedEndpoint{
		Backend: "news",
		BaseURL: srv.URL,
		ExtraHeaders: map[string]string{
			"X-Gate-Cli-Name": "override-value",
		},
		Timeout: 3 * time.Second,
	}, WithDefaultGateCliNameHeader())
	_, _, err := c.ListTools(context.Background())
	require.NoError(t, err)
}

func TestDefaultGateCliNameHeaderOmittedWithoutOption(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, strings.TrimSpace(r.Header.Get("X-Gate-Cli-Name")))

		var req map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		method := req["method"].(string)
		if method == "initialize" {
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"1","result":{"ok":true}}`))
			return
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"2","result":{"tools":[]}}`))
	}))
	defer srv.Close()

	c := New(&toolconfig.ResolvedEndpoint{
		Backend:      "news",
		BaseURL:      srv.URL,
		ExtraHeaders: map[string]string{},
		Timeout:      3 * time.Second,
	})
	_, _, err := c.ListTools(context.Background())
	require.NoError(t, err)
}
