package mcpclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
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
		ua := r.Header.Get("User-Agent")
		assert.Contains(t, ua, "gate-cli/")
		assert.Contains(t, ua, "intel/news")
		assert.Contains(t, ua, "jsonrpc")

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
	var listCalls atomic.Uint32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		method := req["method"].(string)
		if method == "initialize" {
			w.Header().Set("MCP-Session-Id", "sess-1")
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"1","result":{"ok":true}}`))
			return
		}
		listCalls.Add(1)
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
	tool, _, err = c.DescribeTool(context.Background(), "info_coin_get_coin_info")
	require.NoError(t, err)
	require.NotNil(t, tool)
	assert.Equal(t, uint32(1), listCalls.Load(), "describe should use cached index on subsequent lookups")
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

func TestCallToolTransportDiagRedactsSensitiveInSlices(t *testing.T) {
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

	_, _, err := c.CallTool(context.Background(), "news_feed_search_news", map[string]interface{}{
		"items": []interface{}{
			map[string]interface{}{"api_token": "secret-value"},
		},
	})
	require.NoError(t, err)
	log := errOut.String()
	assert.NotContains(t, log, "secret-value")
	assert.Contains(t, log, "***REDACTED***")
}

func TestCallToolTransportDiagDoesNotRedactTokenCount(t *testing.T) {
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

	_, _, err := c.CallTool(context.Background(), "news_feed_search_news", map[string]interface{}{
		"token_count": 7,
	})
	require.NoError(t, err)
	log := errOut.String()
	assert.Contains(t, log, "\"token_count\":7")
	assert.NotContains(t, log, "\"token_count\":\"***REDACTED***\"")
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

func TestExtraHeadersSessionIDCannotOverrideClientSession(t *testing.T) {
	var methods []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		method := req["method"].(string)
		methods = append(methods, method)
		if method == "initialize" {
			assert.Empty(t, strings.TrimSpace(r.Header.Get("MCP-Session-Id")))
			w.Header().Set("MCP-Session-Id", "sess-good")
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"1","result":{"ok":true}}`))
			return
		}
		assert.Equal(t, "sess-good", r.Header.Get("MCP-Session-Id"))
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"2","result":{"tools":[]}}`))
	}))
	defer srv.Close()

	c := New(&toolconfig.ResolvedEndpoint{
		Backend: "news",
		BaseURL: srv.URL,
		ExtraHeaders: map[string]string{
			"MCP-Session-Id": "evil-value",
		},
		Timeout: 3 * time.Second,
	})
	_, _, err := c.ListTools(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{"initialize", "tools/list"}, methods)
}

func TestEnsureInitializedPanicDoesNotDeadlockWaiters(t *testing.T) {
	c := New(&toolconfig.ResolvedEndpoint{
		Backend: "news",
		BaseURL: "https://example.com/mcp/news",
		Timeout: 1 * time.Second,
	})
	// Force a panic in call path (nil receiver on Do) to verify waiters are released.
	c.httpClient = nil

	const workers = 3
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- c.ensureInitialized(context.Background())
		}()
	}
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ensureInitialized waiters blocked")
	}
	close(errs)
	for err := range errs {
		require.Error(t, err)
		assert.Contains(t, err.Error(), "initialize panic")
	}
}

func TestSessionIDPatternAllowsCommonTokenChars(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		method := req["method"].(string)
		if method == "initialize" {
			w.Header().Set("MCP-Session-Id", "abc+/=._:-123")
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"1","result":{"ok":true}}`))
			return
		}
		assert.Equal(t, "abc+/=._:-123", r.Header.Get("MCP-Session-Id"))
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"2","result":{"tools":[]}}`))
	}))
	defer srv.Close()

	c := New(&toolconfig.ResolvedEndpoint{
		Backend: "news",
		BaseURL: srv.URL,
		Timeout: 3 * time.Second,
	})
	_, _, err := c.ListTools(context.Background())
	require.NoError(t, err)
}

func TestShouldInvalidateListCacheOnListError(t *testing.T) {
	code := -32000
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil mcp", errors.New("plain"), false},
		{"transport timeout", &Error{Kind: ErrorKindTransport, Err: errors.New("i/o timeout")}, false},
		{"transport invalid json-rpc", &Error{Kind: ErrorKindTransport, Err: errors.New("invalid json-rpc response: eof")}, true},
		{"transport invalid tools list", &Error{Kind: ErrorKindTransport, Err: errors.New("invalid tools/list result: x")}, true},
		{"protocol with jsonrpc code", &Error{Kind: ErrorKindProtocol, Err: errors.New("e"), JSONRPCCode: &code}, false},
		{"protocol without jsonrpc code", &Error{Kind: ErrorKindProtocol, Err: errors.New("e"), JSONRPCCode: nil}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldInvalidateListCacheOnListError(tc.err); got != tc.want {
				t.Fatalf("shouldInvalidateListCacheOnListError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestListToolsUnmarshalFailureAfterGoodListInvalidatesCache(t *testing.T) {
	var toolListCalls atomic.Uint32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		method := req["method"].(string)
		id := req["id"].(string)
		if method == "initialize" {
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"` + id + `","result":{"ok":true}}`))
			return
		}
		n := toolListCalls.Add(1)
		switch n {
		case 1:
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"` + id + `","result":{"tools":[{"name":"t0"}]}}`))
		case 2:
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"` + id + `","result":"not-an-object"}`))
		default:
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"` + id + `","result":{"tools":[{"name":"t1"}]}}`))
		}
	}))
	defer srv.Close()

	c := New(&toolconfig.ResolvedEndpoint{
		Backend: "news",
		BaseURL: srv.URL,
		Timeout: 3 * time.Second,
	}, CacheTTLForTest(100*time.Millisecond, 100*time.Millisecond))

	tools, _, err := c.ListTools(context.Background())
	require.NoError(t, err)
	require.Len(t, tools, 1)
	assert.Equal(t, "t0", tools[0].Name)
	assert.Equal(t, uint32(1), toolListCalls.Load())

	time.Sleep(120 * time.Millisecond)
	_, _, err = c.ListTools(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid tools/list result")

	time.Sleep(120 * time.Millisecond)
	tools, _, err = c.ListTools(context.Background())
	require.NoError(t, err)
	require.Len(t, tools, 1)
	assert.Equal(t, "t1", tools[0].Name)
	assert.Equal(t, uint32(3), toolListCalls.Load(), "expect fresh tools/list after bad result invalidated cache")
}

func TestListToolsMissingToolsFieldReturnsErrorAndNoCache(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		method := req["method"].(string)
		id := req["id"].(string)
		if method == "initialize" {
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"` + id + `","result":{"ok":true}}`))
			return
		}
		callCount++
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"` + id + `","result":{}}`))
	}))
	defer srv.Close()

	c := New(&toolconfig.ResolvedEndpoint{
		Backend: "news",
		BaseURL: srv.URL,
		Timeout: 3 * time.Second,
	})
	_, _, err := c.ListTools(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing tools field")

	_, _, err = c.ListTools(context.Background())
	require.Error(t, err)
	assert.Equal(t, 2, callCount, "missing tools result must not be cached")
}

func TestListToolsEmptyToolsUsesShortTTL(t *testing.T) {
	var listCalls atomic.Uint32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		method := req["method"].(string)
		id := req["id"].(string)
		if method == "initialize" {
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"` + id + `","result":{"ok":true}}`))
			return
		}
		listCalls.Add(1)
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"` + id + `","result":{"tools":[]}}`))
	}))
	defer srv.Close()

	c := New(&toolconfig.ResolvedEndpoint{
		Backend: "news",
		BaseURL: srv.URL,
		Timeout: 3 * time.Second,
	}, CacheTTLForTest(30*time.Second, 250*time.Millisecond))
	_, _, err := c.ListTools(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint32(1), listCalls.Load())

	_, _, err = c.ListTools(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint32(1), listCalls.Load(), "empty tools list should be cached briefly")

	time.Sleep(350 * time.Millisecond)
	_, _, err = c.ListTools(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint32(2), listCalls.Load(), "empty tools cache should expire quickly")
}

func TestListToolsTransientJSONRPCErrorDoesNotInvalidateGoodCache(t *testing.T) {
	var listCalls atomic.Uint32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		method := req["method"].(string)
		id := req["id"].(string)
		if method == "initialize" {
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"` + id + `","result":{"ok":true}}`))
			return
		}
		n := listCalls.Add(1)
		if n == 2 {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"` + id + `","error":{"code":-32000,"message":"temporary"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"` + id + `","result":{"tools":[{"name":"t1","description":"d"}]}}`))
	}))
	defer srv.Close()

	c := New(&toolconfig.ResolvedEndpoint{
		Backend: "news",
		BaseURL: srv.URL,
		Timeout: 3 * time.Second,
	}, CacheTTLForTest(350*time.Millisecond, 350*time.Millisecond))
	_, _, err := c.ListTools(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint32(1), listCalls.Load())

	time.Sleep(400 * time.Millisecond)
	tools, _, err := c.ListTools(context.Background())
	require.NoError(t, err)
	require.Len(t, tools, 1)
	assert.Equal(t, "t1", tools[0].Name)
	assert.Equal(t, uint32(2), listCalls.Load())

	_, _, err = c.ListTools(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint32(2), listCalls.Load(), "good list cache should survive transient tools/list failure")
}

func TestFallbackListToolsRebuildsDescribeIndex(t *testing.T) {
	var listCalls atomic.Uint32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		method := req["method"].(string)
		id := req["id"].(string)
		if method == "initialize" {
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"` + id + `","result":{"ok":true}}`))
			return
		}
		n := listCalls.Add(1)
		if n == 1 {
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"` + id + `","result":{"tools":[{"name":"t1","description":"d"}]}}`))
			return
		}
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"` + id + `","error":{"code":-32000,"message":"temporary"}}`))
	}))
	defer srv.Close()

	c := New(&toolconfig.ResolvedEndpoint{
		Backend: "news",
		BaseURL: srv.URL,
		Timeout: 3 * time.Second,
	}, CacheTTLForTest(200*time.Millisecond, 200*time.Millisecond))

	_, _, err := c.ListTools(context.Background())
	require.NoError(t, err)
	time.Sleep(250 * time.Millisecond)
	_, _, err = c.ListTools(context.Background())
	require.NoError(t, err)

	c.mu.Lock()
	_, ok := c.toolByName["t1"]
	c.mu.Unlock()
	assert.True(t, ok, "fallback cache restore should rebuild toolByName index")
}

func TestGATE_INTEL_MAX_RESPONSE_BYTESOverridesReadLimit(t *testing.T) {
	t.Setenv("GATE_INTEL_MAX_RESPONSE_BYTES", "10")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		method := req["method"].(string)
		if method == "initialize" {
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"1","result":{"ok":true}}`))
			return
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"2","result":{"tools":[]}}` + strings.Repeat(" ", 50)))
	}))
	defer srv.Close()

	c := New(&toolconfig.ResolvedEndpoint{
		Backend: "news",
		BaseURL: srv.URL,
		Timeout: 3 * time.Second,
	})
	_, _, err := c.ListTools(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "response body exceeded 10 bytes")
	require.ErrorIs(t, err, errIntelHTTPBodyTooLarge)
}

func TestListToolsRejectsMismatchedResponseID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		method := req["method"].(string)
		if method == "initialize" {
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"1","result":{"ok":true}}`))
			return
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"999","result":{"tools":[]}}`))
	}))
	defer srv.Close()

	c := New(&toolconfig.ResolvedEndpoint{
		Backend: "news",
		BaseURL: srv.URL,
		Timeout: 3 * time.Second,
	})
	_, _, err := c.ListTools(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "json-rpc id mismatch")
	var mcpErr *Error
	require.ErrorAs(t, err, &mcpErr)
	assert.Equal(t, ErrorKindProtocol, mcpErr.Kind)
}

func TestInitializeRejectsNonObjectResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		method := req["method"].(string)
		if method == "initialize" {
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"1","result":"ok"}`))
			return
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":"2","result":{"tools":[]}}`))
	}))
	defer srv.Close()

	c := New(&toolconfig.ResolvedEndpoint{
		Backend: "news",
		BaseURL: srv.URL,
		Timeout: 3 * time.Second,
	})
	_, _, err := c.ListTools(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid initialize result")
	var mcpErr *Error
	require.ErrorAs(t, err, &mcpErr)
	assert.Equal(t, ErrorKindProtocol, mcpErr.Kind)
}
