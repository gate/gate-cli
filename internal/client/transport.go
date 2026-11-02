package client

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gate/gate-cli/internal/exitcode"
	"github.com/gate/gate-cli/internal/oauth2"
	"github.com/gate/gate-cli/internal/operations"
)

// 所有SDK服务共用有界连接池，不按命令/请求创建重型Transport。
var sharedAPITransport = &http.Transport{Proxy: http.ProxyFromEnvironment, DialContext: (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext, TLSHandshakeTimeout: 5 * time.Second, ResponseHeaderTimeout: 15 * time.Second, IdleConnTimeout: 90 * time.Second, MaxIdleConns: 32, MaxIdleConnsPerHost: 8, MaxConnsPerHost: 16, ForceAttemptHTTP2: true}

type apiTransport struct {
	resource           string
	hmac, oauth, debug bool
	base               http.RoundTripper
}

// apiRedirect 依据第一跳的方法做判断，防止POST经302变GET后被误当作只读请求。
func apiRedirect(req *http.Request, via []*http.Request) error {
	if len(via) == 0 {
		return http.ErrUseLastResponse
	}
	first := via[0]
	if first.Method != "GET" && first.Method != "HEAD" {
		return http.ErrUseLastResponse
	}
	if len(via) > 3 || req.URL.Scheme != first.URL.Scheme || req.URL.Host != first.URL.Host {
		return fmt.Errorf("API redirect escaped trusted origin")
	}
	// 资源边界的路径核对由同一transport在每一跳发送前再次完成。
	return nil
}
func (transport *apiTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	resource, err := url.Parse(transport.resource)
	if err != nil {
		return nil, err
	}
	if request.URL.Scheme != resource.Scheme || request.URL.Host != resource.Host || (request.URL.Path != resource.Path && !strings.HasPrefix(request.URL.Path, resource.Path+"/")) || request.URL.User != nil || request.URL.Fragment != "" {
		return nil, fmt.Errorf("API request destination does not match bound resource")
	}
	// 对实际发送的原始路径核对，避免代理解码/清理后越过resource或scope边界。
	if strings.ContainsAny(request.URL.EscapedPath(), "%\\") || strings.Contains(request.URL.Path, "//") {
		return nil, fmt.Errorf("ambiguous API request path")
	}
	for _, part := range strings.Split(request.URL.Path, "/") {
		if part == "." || part == ".." {
			return nil, fmt.Errorf("ambiguous API request path")
		}
	}
	route := "/api/v4" + strings.TrimPrefix(request.URL.Path, resource.Path)
	// Release availability precedes credential lookup/refresh and network I/O,
	// including HMAC and anonymous calls from retained command implementations.
	disabled, err := operations.IsDisabled(request.Method, route)
	if err != nil {
		return nil, err
	}
	if disabled {
		return nil, exitcode.New(1, errors.New("operation unavailable in this CLI release"))
	}
	public := operations.IsPublic(request.Method, route)
	outgoing := request.Clone(request.Context())
	if !transport.hmac {
		outgoing.Header.Del("Authorization")
		outgoing.Header.Del("KEY")
		outgoing.Header.Del("SIGN")
		outgoing.Header.Del("Timestamp")
		if !public {
			if !transport.oauth {
				return nil, exitcode.New(4, fmt.Errorf("authentication required: run gate-cli cex auth login"))
			}
			token, err := oauth2.EnsureTokenValid(request.Context(), transport.resource)
			if err != nil {
				return nil, err
			}
			outgoing.Header.Set("Authorization", "Bearer "+token.AccessToken)
		}
	}
	started := time.Now()
	response, err := transport.base.RoundTrip(outgoing)
	if transport.debug {
		// 不输出query、Header、body或底层错误文本；它们都可能携带凭据或业务敏感字段。
		status := 0
		if response != nil {
			status = response.StatusCode
		}
		fmt.Fprintf(os.Stderr, "[debug] API method=%s status=%d duration_ms=%d\n", request.Method, status, time.Since(started).Milliseconds())
	}
	if err != nil && request.Method != "GET" && request.Method != "HEAD" {
		return response, fmt.Errorf("write outcome unknown; reconcile before submitting again: %w", err)
	}
	if response != nil && response.Body != nil {
		response.Body = &boundedAPIResponse{ReadCloser: response.Body, remaining: maxAPIResponseBytes}
		normalizeChaseResponse(response, request.Method, route)
		normalizeInsuranceResponse(response, request.Method, route)
	}
	// This adapter also classifies a missing body on its four confirmed routes;
	// other products retain their existing response and error contracts.
	normalizeFuturesResponse(response, request.Method, route)
	return response, err
}

// A full public currency catalog exceeds 2MiB; keep SDK reads bounded at 8MiB.
const maxAPIResponseBytes int64 = 8 * 1024 * 1024

// boundedAPIResponse 在SDK的ReadAll之前限制解压后响应；越界返回错误，绝不把截断内容当成功。
type boundedAPIResponse struct {
	io.ReadCloser
	remaining int64
}

func (body *boundedAPIResponse) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if int64(len(p)) > body.remaining+1 {
		p = p[:body.remaining+1]
	}
	n, err := body.ReadCloser.Read(p)
	body.remaining -= int64(n)
	if body.remaining < 0 {
		return 0, errors.New("API response exceeds 8MiB limit")
	}
	return n, err
}
