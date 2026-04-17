package mcpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gate/gate-cli/internal/toolconfig"
	"github.com/gate/gate-cli/internal/version"
)

const sessionHeader = "MCP-Session-Id"
const maxResponseBytes = 16 << 20 // 16 MiB

var (
	sessionIDPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,256}$`)
	sensitiveArgKeys = map[string]struct{}{
		"authorization": {},
		"secret":        {},
		"token":         {},
		"key":           {},
		"signature":     {},
		"passphrase":    {},
	}
)

// gateCLIProductHeader aligns MCP HTTP requests with REST client defaults
// (internal/client): identifies the caller as gate-cli for analytics / routing.
const gateCLIProductHeader = "X-Gate-Cli-Name"
const gateCLIProductValue = "gate-cli"

// Tool is a simplified MCP tool metadata representation.
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	InputSchema interface{} `json:"inputSchema,omitempty"`
}

// Option mutates client construction behavior.
type Option func(*Client)

// WithDebug enables RPC transport logs to stderr with a [debug] prefix (Gate API --debug
// convention). Prefer WithTransportDiag when choosing a prefix explicitly.
func WithDebug(debug bool) Option {
	return WithTransportDiag(debug, "[debug]")
}

// WithTransportDiag enables or disables RPC transport logs to stderr. When enabled, tag
// is printed as a line prefix (e.g. "[verbose]" or "[debug]"); empty tag defaults to "[debug]".
func WithTransportDiag(enabled bool, tag string) Option {
	return func(c *Client) {
		c.transportDiag = enabled
		if !enabled {
			c.transportDiagTag = ""
			return
		}
		if tag != "" {
			c.transportDiagTag = tag
		} else {
			c.transportDiagTag = "[debug]"
		}
	}
}

// WithDefaultGateCliNameHeader enables the same default product header as REST
// (X-Gate-Cli-Name: gate-cli) when it is not already set, including via
// GATE_INTEL_EXTRA_HEADERS. Used by gate-cli info/news only.
func WithDefaultGateCliNameHeader() Option {
	return func(c *Client) {
		c.injectDefaultGateCliName = true
	}
}

type rpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      string      `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error,omitempty"`
}

type listToolsResult struct {
	Tools []Tool `json:"tools"`
}

type callToolsParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ContentItem is a generic MCP content payload item.
type ContentItem map[string]interface{}

// CallResult is a simplified projection of tools/call result.
type CallResult struct {
	ContentRaw        []interface{}          `json:"content_raw,omitempty"`
	IsError           bool                   `json:"isError,omitempty"`
	StructuredContent map[string]interface{} `json:"structuredContent,omitempty"`
	Meta              map[string]interface{} `json:"_meta,omitempty"`
	Raw               map[string]interface{} `json:"raw,omitempty"`
}

// Client calls MCP JSON-RPC over HTTP for intel features.
type Client struct {
	backend                  string
	baseURL                  string
	bearerToken              string
	extraHeaders             map[string]string
	httpClient               *http.Client
	transportDiag            bool
	transportDiagTag         string
	injectDefaultGateCliName bool
	errOut                   io.Writer

	sessionID string
	idSeq     uint64

	mu             sync.Mutex
	listCache      []Tool
	listCacheUntil time.Time
	initializing   bool
	initErr        error
	initWait       chan struct{}
}

// New constructs an MCP client from resolved endpoint.
func New(endpoint *toolconfig.ResolvedEndpoint, opts ...Option) *Client {
	c := &Client{
		backend:      endpoint.Backend,
		baseURL:      endpoint.BaseURL,
		bearerToken:  endpoint.BearerToken,
		extraHeaders: endpoint.ExtraHeaders,
		httpClient: &http.Client{
			Timeout: endpoint.Timeout,
			Transport: &http.Transport{
				Proxy:                 http.ProxyFromEnvironment,
				DialContext:           (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 30 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
		errOut:       os.Stderr,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// ListTools requests tools/list and returns available tools.
func (c *Client) ListTools(ctx context.Context) ([]Tool, *http.Response, error) {
	c.mu.Lock()
	if len(c.listCache) > 0 && time.Now().Before(c.listCacheUntil) {
		tools := append([]Tool(nil), c.listCache...)
		c.mu.Unlock()
		return tools, nil, nil
	}
	c.mu.Unlock()

	if err := c.ensureInitialized(ctx); err != nil {
		return nil, nil, err
	}

	params := map[string]interface{}{}
	respPayload, httpResp, reqID, err := c.callWithRetry(ctx, "tools/list", params, true)
	if err != nil {
		c.invalidateListCache()
		var protoErr *Error
		if errors.As(err, &protoErr) {
			protoErr.RequestID = reqID
			return nil, httpResp, protoErr
		}
		return nil, httpResp, &Error{
			Kind:      ErrorKindProtocol,
			Err:       err,
			RequestID: reqID,
		}
	}

	var parsed listToolsResult
	if err := json.Unmarshal(respPayload.Result, &parsed); err != nil {
		c.invalidateListCache()
		return nil, httpResp, &Error{
			Kind:      ErrorKindTransport,
			Err:       fmt.Errorf("invalid tools/list result: %w", err),
			RequestID: reqID,
		}
	}

	c.mu.Lock()
	c.listCache = append([]Tool(nil), parsed.Tools...)
	c.listCacheUntil = time.Now().Add(30 * time.Second)
	c.mu.Unlock()

	return parsed.Tools, httpResp, nil
}

// DescribeTool returns one tool schema/description by name.
func (c *Client) DescribeTool(ctx context.Context, name string) (*Tool, *http.Response, error) {
	tools, resp, err := c.ListTools(ctx)
	if err != nil {
		return nil, resp, err
	}
	for _, t := range tools {
		if t.Name == name {
			cp := t
			return &cp, resp, nil
		}
	}
	return nil, resp, &Error{
		Kind:      ErrorKindProtocol,
		Err:       fmt.Errorf("tool %q not found", name),
		ToolName:  name,
		RequestID: "",
	}
}

// CallTool calls one tool with final merged arguments.
func (c *Client) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*CallResult, *http.Response, error) {
	if err := c.ensureInitialized(ctx); err != nil {
		return nil, nil, err
	}

	params := callToolsParams{
		Name:      name,
		Arguments: arguments,
	}
	c.logCallArguments(name, arguments)
	respPayload, httpResp, reqID, err := c.callWithRetry(ctx, "tools/call", params, false)
	if err != nil {
		var protoErr *Error
		if errors.As(err, &protoErr) {
			protoErr.RequestID = reqID
			protoErr.ToolName = name
			return nil, httpResp, protoErr
		}
		return nil, httpResp, &Error{
			Kind:      ErrorKindProtocol,
			Err:       err,
			RequestID: reqID,
			ToolName:  name,
		}
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(respPayload.Result, &raw); err != nil {
		return nil, httpResp, &Error{
			Kind:      ErrorKindTransport,
			Err:       fmt.Errorf("invalid tools/call result: %w", err),
			RequestID: reqID,
			ToolName:  name,
		}
	}

	rawCloned := cloneMapAny(raw)
	out := &CallResult{Raw: rawCloned}
	if v, ok := rawCloned["isError"].(bool); ok {
		out.IsError = v
	}
	if sc, ok := rawCloned["structuredContent"].(map[string]interface{}); ok {
		out.StructuredContent = cloneMapAny(sc)
	}
	if meta, ok := rawCloned["_meta"].(map[string]interface{}); ok {
		out.Meta = cloneMapAny(meta)
	}
	if contentAny, ok := rawCloned["content"].([]interface{}); ok {
		out.ContentRaw = cloneSliceAny(contentAny)
	}
	return out, httpResp, nil
}

func (c *Client) logCallArguments(toolName string, arguments map[string]interface{}) {
	if !c.transportDiag {
		return
	}
	tag := c.transportDiagTag
	if tag == "" {
		tag = "[debug]"
	}
	args := "{}"
	if arguments != nil {
		if b, err := json.Marshal(redactSensitive(arguments)); err == nil {
			args = string(b)
		} else {
			args = `{"_error":"failed to marshal arguments"}`
		}
	}
	_, _ = fmt.Fprintf(c.errOut, "%s backend=%s rpc_method=tools/call tool_name=%s arguments=%s\n",
		tag, c.backend, toolName, args)
}

func (c *Client) ensureInitialized(ctx context.Context) error {
	c.mu.Lock()
	sid := c.sessionID
	if sid != "" {
		c.mu.Unlock()
		return nil
	}
	if c.initializing {
		waitCh := c.initWait
		c.mu.Unlock()
		<-waitCh
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.sessionID != "" {
			return nil
		}
		return c.initErr
	}
	c.initializing = true
	c.initWait = make(chan struct{})
	waitCh := c.initWait
	c.mu.Unlock()

	params := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"clientInfo": map[string]string{
			"name":    "gate-cli",
			"version": version.Version,
		},
		"capabilities": map[string]interface{}{},
	}
	_, _, _, err := c.call(ctx, "initialize", params)

	c.mu.Lock()
	c.initializing = false
	c.initErr = err
	close(waitCh)
	c.mu.Unlock()
	return err
}

func (c *Client) callWithRetry(ctx context.Context, method string, params interface{}, retryOnUnauthorized bool) (*rpcResponse, *http.Response, string, error) {
	resp, httpResp, reqID, err := c.call(ctx, method, params)
	if err == nil || !retryOnUnauthorized {
		return resp, httpResp, reqID, err
	}
	if httpResp == nil || httpResp.StatusCode != http.StatusUnauthorized {
		return resp, httpResp, reqID, err
	}
	// Session may be invalid. Re-initialize once for idempotent calls.
	c.resetSession()
	if initErr := c.ensureInitialized(ctx); initErr != nil {
		return resp, httpResp, reqID, errors.Join(err, initErr)
	}
	return c.call(ctx, method, params)
}

func (c *Client) call(ctx context.Context, method string, params interface{}) (*rpcResponse, *http.Response, string, error) {
	reqID := fmt.Sprintf("%d", atomic.AddUint64(&c.idSeq, 1))
	payload := rpcRequest{
		JSONRPC: "2.0",
		ID:      reqID,
		Method:  method,
		Params:  params,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		c.logTransportFailure(method, reqID, 0, err)
		return nil, nil, reqID, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		c.logTransportFailure(method, reqID, 0, err)
		return nil, nil, reqID, err
	}
	c.applyHeaders(req)

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		c.logTransportFailure(method, reqID, time.Since(start), err)
		return nil, resp, reqID, &Error{Kind: ErrorKindTransport, Err: err, RequestID: reqID}
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if readErr != nil {
		c.logTransportFailure(method, reqID, time.Since(start), readErr)
		return nil, resp, reqID, &Error{Kind: ErrorKindTransport, Err: readErr, RequestID: reqID}
	}
	if len(respBody) > maxResponseBytes {
		tooLargeErr := fmt.Errorf("response body exceeded %d bytes", maxResponseBytes)
		c.logTransportFailure(method, reqID, time.Since(start), tooLargeErr)
		return nil, resp, reqID, &Error{Kind: ErrorKindTransport, Err: tooLargeErr, RequestID: reqID}
	}

	if sid := strings.TrimSpace(resp.Header.Get(sessionHeader)); sid != "" {
		if !sessionIDPattern.MatchString(sid) {
			invalidSessionErr := fmt.Errorf("invalid session id format")
			c.logTransportFailure(method, reqID, time.Since(start), invalidSessionErr)
			return nil, resp, reqID, &Error{Kind: ErrorKindProtocol, Err: invalidSessionErr, RequestID: reqID}
		}
		c.mu.Lock()
		c.sessionID = sid
		c.mu.Unlock()
	}

	var parsed rpcResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		c.logTransportFailure(method, reqID, time.Since(start), fmt.Errorf("invalid json-rpc response: %w", err))
		return nil, resp, reqID, &Error{Kind: ErrorKindTransport, Err: fmt.Errorf("invalid json-rpc response: %w", err), RequestID: reqID}
	}
	if parsed.Error != nil {
		c.logTransportFailure(method, reqID, time.Since(start), fmt.Errorf("json-rpc error code=%d message=%s", parsed.Error.Code, parsed.Error.Message))
		return nil, resp, reqID, &Error{
			Kind:        ErrorKindProtocol,
			Err:         errors.New(parsed.Error.Message),
			RequestID:   reqID,
			JSONRPCCode: &parsed.Error.Code,
		}
	}

	c.logDebug(method, reqID, time.Since(start), resp)
	return &parsed, resp, reqID, nil
}

func (c *Client) applyHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range c.extraHeaders {
		req.Header.Set(k, v)
	}
	if c.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearerToken)
	}
	c.mu.Lock()
	sid := c.sessionID
	c.mu.Unlock()
	if sid != "" {
		req.Header.Set(sessionHeader, sid)
	}
	if c.injectDefaultGateCliName && strings.TrimSpace(req.Header.Get(gateCLIProductHeader)) == "" {
		req.Header.Set(gateCLIProductHeader, gateCLIProductValue)
	}
}

func (c *Client) logDebug(method, requestID string, elapsed time.Duration, resp *http.Response) {
	if !c.transportDiag {
		return
	}
	tag := c.transportDiagTag
	if tag == "" {
		tag = "[debug]"
	}
	traceID := ""
	status := 0
	if resp != nil {
		traceID = resp.Header.Get("x-gate-trace-id")
		status = resp.StatusCode
	}
	c.mu.Lock()
	sessionSet := c.sessionID != ""
	c.mu.Unlock()
	_, _ = fmt.Fprintf(c.errOut, "%s backend=%s rpc_method=%s request_id=%s status=%d elapsed_ms=%d trace_id=%s session_set=%t\n",
		tag, c.backend, method, requestID, status, elapsed.Milliseconds(), traceID, sessionSet)
}

func (c *Client) logTransportFailure(method, requestID string, elapsed time.Duration, err error) {
	if !c.transportDiag {
		return
	}
	tag := c.transportDiagTag
	if tag == "" {
		tag = "[debug]"
	}
	c.mu.Lock()
	sessionSet := c.sessionID != ""
	c.mu.Unlock()
	_, _ = fmt.Fprintf(c.errOut, "%s backend=%s rpc_method=%s request_id=%s transport_error=%q elapsed_ms=%d session_set=%t\n",
		tag, c.backend, method, requestID, err.Error(), elapsed.Milliseconds(), sessionSet)
}

func (c *Client) resetSession() {
	c.mu.Lock()
	c.sessionID = ""
	c.mu.Unlock()
}

func (c *Client) invalidateListCache() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.listCache = nil
	c.listCacheUntil = time.Time{}
}

func redactSensitive(input map[string]interface{}) map[string]interface{} {
	if len(input) == 0 {
		return map[string]interface{}{}
	}
	out := make(map[string]interface{}, len(input))
	maps.Copy(out, input)
	for k, v := range out {
		lk := strings.ToLower(strings.TrimSpace(k))
		for sk := range sensitiveArgKeys {
			if strings.Contains(lk, sk) {
				out[k] = "***REDACTED***"
				break
			}
		}
		if nested, ok := v.(map[string]interface{}); ok {
			out[k] = redactSensitive(nested)
		}
	}
	return out
}

func cloneAny(v interface{}) interface{} {
	switch x := v.(type) {
	case map[string]interface{}:
		return cloneMapAny(x)
	case []interface{}:
		return cloneSliceAny(x)
	default:
		return x
	}
}

func cloneMapAny(in map[string]interface{}) map[string]interface{} {
	if in == nil {
		return nil
	}
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = cloneAny(v)
	}
	return out
}

func cloneSliceAny(in []interface{}) []interface{} {
	if in == nil {
		return nil
	}
	out := make([]interface{}, len(in))
	for i, v := range in {
		out[i] = cloneAny(v)
	}
	return out
}

