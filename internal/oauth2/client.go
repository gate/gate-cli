package oauth2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DCRRequest is the Dynamic Client Registration request body (RFC 7591).
type DCRRequest struct {
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	Scope                   string   `json:"scope"`
}

// DCRResponse is the DCR response body.
type DCRResponse struct {
	ClientID       string   `json:"client_id"`
	ClientIDIssued int64    `json:"client_id_issued_at"`
	RedirectURIs   []string `json:"redirect_uris,omitempty"`
}

// TokenResponse is the token endpoint response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// oauthResponseBodyLimit caps any response body we read from the authorization
// server. DCR/token responses are normally <2 KiB; 64 KiB is generous enough
// for worst-case error payloads while still bounding memory.
const oauthResponseBodyLimit = 64 << 10

// httpClient uses an explicit Transport so that connection / TLS / header
// timeouts are independently bounded. The total request timeout (30s) is still
// enforced by http.Client.Timeout.
var httpClient = &http.Client{
	// 写请求收到3xx按结果未确认处理；不得跨站或同站重放。
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	Timeout:       30 * time.Second,
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          20, MaxIdleConnsPerHost: 4, MaxConnsPerHost: 8,
	},
}

// Register 每次用户动作只发一次DCR；响应丢失不能靠自动重试制造重复注册。
func Register(ctx context.Context, redirectURIs []string) (*DCRResponse, error) {
	body := DCRRequest{ClientName: ClientName, RedirectURIs: redirectURIs, GrantTypes: []string{"authorization_code", "refresh_token"}, TokenEndpointAuthMethod: "none", Scope: DefaultScopes}
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	resp, err := oauthWrite(ctx, ActiveEndpoints().RegistrationEndpoint, "application/json", raw)
	if err != nil {
		return nil, err
	}
	result, _, err := handleDCRResponse(resp)
	if err != nil {
		return nil, err
	}
	if result.ClientID == "" || len(result.ClientID) > 256 {
		return nil, fmt.Errorf("invalid DCR client ID")
	}
	return result, nil
}

// oauthWrite 是所有一次性OAuth写请求的唯一发送边界，校验受信issuer并禁止任何重定向。
func oauthWrite(ctx context.Context, endpoint, contentType string, body []byte) (*http.Response, error) {
	base, baseErr := url.Parse(ActiveEndpoints().BaseURL)
	target, err := url.Parse(endpoint)
	if err != nil || baseErr != nil || base.Host == "" || target.Host != base.Host || target.Scheme != base.Scheme || target.User != nil || target.RawQuery != "" || target.Fragment != "" || (target.Scheme != "https" && target.Scheme != "http") {
		return nil, fmt.Errorf("OAuth endpoint does not match trusted issuer")
	}
	prefix := strings.TrimRight(base.Path, "/")
	if prefix != "" && target.Path != prefix && !strings.HasPrefix(target.Path, prefix+"/") {
		return nil, fmt.Errorf("OAuth endpoint escapes issuer prefix")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return httpClient.Do(req)
}

// handleDCRResponse reads the body and classifies the status into success / retryable / terminal.
// Callers must not use resp after this returns; its body is always closed here.
func handleDCRResponse(resp *http.Response) (*DCRResponse, bool, error) {
	defer closeOAuthBody(resp.Body)

	respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, oauthResponseBodyLimit+1))
	if len(respBody) > oauthResponseBodyLimit {
		return nil, false, fmt.Errorf("DCR response too large")
	}
	if readErr != nil {
		return nil, true, fmt.Errorf("DCR response read error: %w", readErr)
	}

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		var result DCRResponse
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, false, fmt.Errorf("DCR response parse error: %w", err)
		}
		return &result, false, nil
	}

	retryable := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
	return nil, retryable, fmt.Errorf("DCR failed (HTTP %d): %w", resp.StatusCode, extractOAuth2Error(respBody, resp.StatusCode))
}

// BuildAuthorizeURL constructs the authorization URL with PKCE parameters,
// honoring the active Endpoints (overridable via oauth2.yaml).
func BuildAuthorizeURL(clientID, redirectURI, state, codeChallenge, scope, resource string) string {
	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {clientID},
		"redirect_uri":          {redirectURI},
		"state":                 {state},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
		"scope":                 {scope},
		"resource":              {resource},
	}
	return ActiveEndpoints().AuthorizationEndpoint + "?" + params.Encode()
}

// ExchangeCode exchanges an authorization code for tokens.
func ExchangeCode(ctx context.Context, code, clientID, redirectURI, codeVerifier, resource string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {clientID},
		"redirect_uri":  {redirectURI},
		"code_verifier": {codeVerifier},
		"resource":      {resource},
	}
	return doTokenRequestCtx(ctx, data)
}

// RefreshToken exchanges a refresh token for new tokens.
func RefreshToken(ctx context.Context, refreshToken, clientID, resource string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"resource":      {resource},
		"client_id":     {clientID},
	}
	return doTokenRequestCtx(ctx, data)
}

// RevokeToken revokes an access or refresh token.
func RevokeToken(ctx context.Context, token, clientID string) error {
	data := url.Values{"token": {token}, "client_id": {clientID}}
	resp, err := oauthWrite(ctx, ActiveEndpoints().RevocationEndpoint, "application/x-www-form-urlencoded", []byte(data.Encode()))
	if err != nil {
		return fmt.Errorf("revoke request: %w", err)
	}
	defer closeOAuthBody(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("revoke was not confirmed (HTTP %d)", resp.StatusCode)
	}
	return nil
}

// OAuth2ServerError represents a structured error from the OAuth2 token/DCR endpoint.
// Callers can use errors.As to inspect ErrorCode without string matching.
type OAuth2ServerError struct {
	ErrorCode   string // e.g. "invalid_grant", "invalid_client"
	Description string
	StatusCode  int
}

func (e *OAuth2ServerError) Error() string {
	if e.Description != "" {
		return e.ErrorCode + ": " + e.Description
	}
	return e.ErrorCode
}

// extractOAuth2Error parses the standard error/error_description from an OAuth2 error response.
// Always returns an *OAuth2ServerError so callers can use errors.As to recover
// the HTTP status and a recognized protocol code. Remote descriptions stay private.
func extractOAuth2Error(body []byte, statusCode int) error {
	var parsed struct {
		Error       string `json:"error"`
		Description string `json:"error_description"`
	}
	if json.Unmarshal(body, &parsed) == nil {
		// 只显示稳定协议码；服务端描述可能回显AT/RT/验证材料，不能进入stderr。
		switch parsed.Error {
		case "invalid_grant", "invalid_client", "invalid_request", "invalid_scope", "invalid_target", "invalid_redirect_uri", "invalid_client_metadata", "unsupported_grant_type", "unsupported_response_type", "access_denied", "temporarily_unavailable", "server_error":
			return &OAuth2ServerError{ErrorCode: parsed.Error, StatusCode: statusCode}
		}
	}
	// Fallback: keep the status code so upstream errors.As still works, and
	// truncate the raw body so server internals can't leak into stderr.

	return &OAuth2ServerError{
		ErrorCode:   "server_error",
		Description: "Authorization server returned an invalid error response",
		StatusCode:  statusCode,
	}
}

// doTokenRequestCtx is the ctx-aware token request path. When ctx carries a
// deadline it bounds connection + body read independent of httpClient.Timeout.
func doTokenRequestCtx(ctx context.Context, data url.Values) (*TokenResponse, error) {
	resp, err := oauthWrite(ctx, ActiveEndpoints().TokenEndpoint, "application/x-www-form-urlencoded", []byte(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer closeOAuthBody(resp.Body)

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, oauthResponseBodyLimit+1))
	if len(body) > oauthResponseBodyLimit {
		return nil, fmt.Errorf("token response too large")
	}
	if readErr != nil {
		return nil, fmt.Errorf("token response read error: %w", readErr)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed (HTTP %d): %w", resp.StatusCode, extractOAuth2Error(body, resp.StatusCode))
	}

	var result TokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("token response parse error: %w", err)
	}
	// 防止空凭据、错误Token类型或time.Duration溢出被保存为有效授权。
	if result.AccessToken == "" || result.RefreshToken == "" || !strings.EqualFold(result.TokenType, "Bearer") || result.ExpiresIn <= 0 || result.ExpiresIn > int64((1<<63-1)/int64(time.Second)) || !validGrantedScopes(result.Scope) {
		return nil, fmt.Errorf("invalid token response fields")
	}
	return &result, nil
}

// validGrantedScopes 只接受已确认的六项；read/asset是base在服务端展开的历史执行别名。
func validGrantedScopes(raw string) bool {
	if raw == "" || len(raw) > 4096 || strings.ContainsAny(raw, "\r\n\t") {
		return false
	}
	scopes := strings.Fields(raw)
	base := false
	for _, scope := range scopes {
		if scope == "base" {
			base = true
		}
	}
	for _, scope := range scopes {
		if strings.Contains(" "+DefaultScopes+" ", " "+scope+" ") {
			continue
		}
		if base && (scope == "read" || scope == "asset") {
			continue
		}
		return false
	}
	return true
}

// closeOAuthBody 已完整读取的HTTP响应关闭失败只记分类日志，不把已提交Token重新当作可重试操作。
func closeOAuthBody(body io.Closer) {
	if err := body.Close(); err != nil {
		log.Print("OAuth response body close failed")
	}
}
