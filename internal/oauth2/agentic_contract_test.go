package oauth2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// setupTestTokenDir 隔离凭据与进程内缓存，所有测试不访问用户真实配置。
func setupTestTokenDir(t *testing.T) (string, func()) {
	t.Helper()
	home := isolateHome(t)
	if err := os.MkdirAll(filepath.Join(home, ".gate-cli"), 0700); err != nil {
		t.Fatal(err)
	}
	return home, func() { resetHomeCache() }
}
func ensureActiveDiscoveryURL(t *testing.T, issuer string) {
	t.Helper()
	if issuer == "" {
		issuer = DefaultDiscoveryURL
	}
	previous := discoveryURLSource
	discoveryURLSource = func() string { return NormalizeDiscoveryURL(issuer) }
	t.Cleanup(func() { discoveryURLSource = previous })
}
func writeTestToken(t *testing.T, token *TokenData) {
	t.Helper()
	ensureActiveDiscoveryURL(t, token.DiscoveryURL)
	if err := SaveToken(token); err != nil {
		t.Fatal(err)
	}
}

func TestAgenticDefaultScopes(t *testing.T) {
	if DefaultScopes != "base spot_trade futures_trade earn cfd stock" {
		t.Fatalf("default scopes=%q", DefaultScopes)
	}
}

// TestAPIResource 证明受众来自已解析地址，危险或歧义URL在发凭据之前被拒绝。
func TestAPIResource(t *testing.T) {
	for _, tc := range []struct{ base, want string }{
		{"https://api.gateio.ws", "https://api.gateio.ws/api/v4"},
		{"https://api.gateio.ws/", "https://api.gateio.ws/api/v4"},
		{"https://trusted.example.test/prefix", "https://trusted.example.test/prefix/api/v4"},
		{"http://127.0.0.1:12345", "http://127.0.0.1:12345/api/v4"},
		// 测试/预发可使用非回环HTTP，受众仍保留完整协议、主机和端口。
		{"http://10.0.0.1:18080", "http://10.0.0.1:18080/api/v4"},
		{"http://192.0.2.10:18081/", "http://192.0.2.10:18081/api/v4"},
		{"http://api.dev.example.test/prefix", "http://api.dev.example.test/prefix/api/v4"},
		{"http://[2001:db8::1]:18080", "http://[2001:db8::1]:18080/api/v4"},
		{"http://localhost:18080", "http://localhost:18080/api/v4"},
		{"https://user:password@api.gateio.ws", ""}, {"https://api.gateio.ws?token=synthetic", ""},
		{"https://api.gateio.ws#fragment", ""}, {"https://api.gateio.ws/api/v4", ""},
		{"https://api.gateio.ws/a/../b", ""}, {"https://api.gateio.ws/a%2fb", ""},
		{"http://user:password@10.0.0.1:18080", ""}, {"http://10.0.0.1:18080?token=synthetic", ""},
		{"http://10.0.0.1:18080#fragment", ""}, {"http://10.0.0.1:18080/api/v4", ""},
		{"http://10.0.0.1:18080/a/../b", ""}, {"http://10.0.0.1:18080/a%2fb", ""},
		{"ftp://10.0.0.1:18080", ""}, {"file:///tmp/test", ""}, {"//10.0.0.1:18080", ""},
	} {
		t.Run(tc.base, func(t *testing.T) {
			got, err := APIResource(tc.base)
			if tc.want == "" {
				if err == nil {
					t.Errorf("unsafe resource accepted: %q", got)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Errorf("resource=%q err=%v want%q", got, err, tc.want)
			}
		})
	}
}

func TestAgenticResourcePersistenceAndLogoutMetadata(t *testing.T) {
	for _, issuer := range []string{DefaultDiscoveryURL, "https://staging.example.test"} {
		t.Run(issuer, func(t *testing.T) {
			_, cleanup := setupTestTokenDir(t)
			defer cleanup()
			td := &TokenData{ClientID: "synthetic-client", DiscoveryURL: issuer, AccessToken: "synthetic-at", RefreshToken: "synthetic-rt", Resource: "https://api.gateio.ws/api/v4", Scopes: "stock", ExpiresAt: time.Now().Add(time.Hour), RegisteredRedirectURIs: []string{"http://127.0.0.1:18991/callback"}}
			writeTestToken(t, td)
			loaded, err := LoadToken()
			if err != nil {
				t.Fatal(err)
			}
			if loaded == nil || loaded.Resource != td.Resource || loaded.Scopes != "stock" {
				t.Fatalf("resource/grant not preserved: %v", loaded)
			}
			if err := RemoveToken(); err != nil {
				t.Fatal(err)
			}
			loaded, err = LoadToken()
			if err != nil {
				t.Fatal(err)
			}
			if loaded == nil || loaded.Resource != td.Resource || loaded.ClientID != td.ClientID || len(loaded.RegisteredRedirectURIs) != 1 {
				t.Fatalf("logout lost registration provenance: %v", loaded)
			}
			if loaded.AccessToken != "" || loaded.RefreshToken != "" || loaded.Scopes != "" {
				t.Fatal("logout retained credential or grant")
			}
		})
	}
}

func TestAgenticUnboundCredentialNeverRefreshes(t *testing.T) {
	_, cleanup := setupTestTokenDir(t)
	defer cleanup()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) }))
	t.Cleanup(server.Close)
	previous := setActiveEndpointsForTest(&Endpoints{BaseURL: server.URL, TokenEndpoint: server.URL})
	t.Cleanup(func() { setActiveEndpointsForTest(previous) })
	for _, resource := range []string{"", "https://different.example.test/api/v4"} {
		writeTestToken(t, &TokenData{ClientID: "synthetic-client", AccessToken: "synthetic-at", RefreshToken: "synthetic-rt", Resource: resource, ExpiresAt: time.Now().Add(-time.Minute)})
		if _, err := EnsureTokenValid(t.Context(), "https://api.gateio.ws/api/v4"); err == nil {
			t.Error("unbound credential accepted")
		}
	}
	if calls.Load() != 0 {
		t.Errorf("unbound credential sent %d requests", calls.Load())
	}
}

// TestAgenticRefreshFailureBurnsRT 模拟请求提交后的503/坏响应，后续命令不得再次消费旧RT。
func TestAgenticRefreshFailureBurnsRT(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"503", 503, `{"error":"server_error"}`}, {"malformed", 200, `{`},
		{"empty tokens", 200, `{"token_type":"Bearer","expires_in":3600,"scope":"stock"}`},
		{"bad type", 200, `{"access_token":"new","refresh_token":"newrt","token_type":"MAC","expires_in":3600,"scope":"stock"}`},
		{"expiry overflow", 200, `{"access_token":"new","refresh_token":"newrt","token_type":"Bearer","expires_in":9223372036854775807,"scope":"stock"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, cleanup := setupTestTokenDir(t)
			defer cleanup()
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if err := r.ParseForm(); err != nil {
					t.Error(err)
				}
				if r.Form.Get("resource") != "https://api.gateio.ws/api/v4" {
					t.Error("refresh lost resource")
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				if _, err := w.Write([]byte(tc.body)); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			previous := setActiveEndpointsForTest(&Endpoints{BaseURL: server.URL, TokenEndpoint: server.URL})
			t.Cleanup(func() { setActiveEndpointsForTest(previous) })
			td := &TokenData{ClientID: "synthetic-client", DiscoveryURL: server.URL, AccessToken: "synthetic-at", RefreshToken: "one-use-rt", Resource: "https://api.gateio.ws/api/v4", Scopes: "stock", ExpiresAt: time.Now().Add(30 * time.Second)}
			writeTestToken(t, td)
			if _, err := ForceRefresh(t.Context(), td.Resource); err == nil {
				t.Error("invalid/unknown refresh response accepted")
			}
			loaded, err := LoadToken()
			if err != nil {
				t.Fatal(err)
			}
			if loaded == nil || loaded.RefreshToken != "" {
				t.Fatal("old RT remained on disk")
			}
			if _, err := ForceRefresh(t.Context(), td.Resource); err == nil {
				t.Error("second refresh unexpectedly succeeded")
			}
			if calls.Load() != 1 {
				t.Errorf("consumed old RT %d times", calls.Load())
			}
		})
	}
}

func TestAgenticConcurrentRefreshHasSingleConsumer(t *testing.T) {
	_, cleanup := setupTestTokenDir(t)
	defer cleanup()
	var calls atomic.Int32
	entered, release := make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			close(entered)
		}
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(TokenResponse{AccessToken: "new-at", RefreshToken: "new-rt", TokenType: "Bearer", ExpiresIn: 3600, Scope: "stock"}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	previous := setActiveEndpointsForTest(&Endpoints{BaseURL: server.URL, TokenEndpoint: server.URL})
	t.Cleanup(func() { setActiveEndpointsForTest(previous) })
	resource := "https://api.gateio.ws/api/v4"
	writeTestToken(t, &TokenData{ClientID: "synthetic-client", DiscoveryURL: server.URL, AccessToken: "old-at", RefreshToken: "one-use-rt", Resource: resource, Scopes: "stock", ExpiresAt: time.Now().Add(time.Hour)})
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	results := make(chan error, 1)
	go func() { _, err := ForceRefresh(ctx, resource); results <- err }()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("refresh did not start")
	}
	// 第二个强制刷新必须在第一个完成前返回忙/取消，不能消费同一旧RT。
	secondCtx, secondCancel := context.WithTimeout(ctx, 100*time.Millisecond)
	_, secondErr := ForceRefresh(secondCtx, resource)
	secondCancel()
	if secondErr == nil {
		t.Error("in-flight forced refresh must not report an uncommitted success")
	}
	close(release)
	if firstErr := <-results; firstErr != nil {
		t.Fatalf("owning refresh failed: %v", firstErr)
	}
	if calls.Load() != 1 {
		t.Errorf("concurrent RT requests=%d want1", calls.Load())
	}
	loaded, err := LoadToken()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.AccessToken != "new-at" || loaded.RefreshToken != "new-rt" || loaded.Resource != resource {
		t.Fatal("successful refresh did not persist bound new pair")
	}
}

func TestOAuthWritesRejectRedirectAndDCRRetry(t *testing.T) {
	for _, status := range []int{307, 308, 429, 503} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var calls, redirected atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/second" {
					redirected.Add(1)
					w.WriteHeader(500)
					return
				}
				calls.Add(1)
				w.Header().Set("Location", "/second")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				if _, err := w.Write([]byte(`{"error":"server_error","error_description":"synthetic"}`)); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			previous := setActiveEndpointsForTest(&Endpoints{BaseURL: server.URL, RegistrationEndpoint: server.URL, RevocationEndpoint: server.URL, TokenEndpoint: server.URL})
			t.Cleanup(func() { setActiveEndpointsForTest(previous) })
			if _, err := Register(t.Context(), []string{"http://127.0.0.1:18991/callback"}); err == nil {
				t.Error("DCR error accepted")
			}
			if err := RevokeToken(t.Context(), "synthetic-rt", "synthetic-client"); err == nil {
				t.Error("revoke error accepted")
			}
			if calls.Load() != 2 || redirected.Load() != 0 {
				t.Errorf("original=%d redirected=%d want2/0", calls.Load(), redirected.Load())
			}
		})
	}
}

// TestEnsureFreshTokenDoesNotStartBackgroundRefresh 保证一小时AT不会每次命令触发后台轮换。
func TestEnsureFreshTokenDoesNotStartBackgroundRefresh(t *testing.T) {
	_, cleanup := setupTestTokenDir(t)
	defer cleanup()
	resource := "https://api.gateio.ws/api/v4"
	writeTestToken(t, &TokenData{ClientID: "synthetic-client", AccessToken: "old-at", RefreshToken: "old-rt", Resource: resource, Scopes: "stock", ExpiresAt: time.Now().Add(time.Hour)})
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) }))
	t.Cleanup(server.Close)
	previous := setActiveEndpointsForTest(&Endpoints{BaseURL: server.URL, TokenEndpoint: server.URL})
	t.Cleanup(func() { setActiveEndpointsForTest(previous) })
	for range 3 {
		if _, err := EnsureTokenValid(context.Background(), resource); err != nil {
			t.Fatal(err)
		}
	}
	// 文件锁屏障要求所有刷新仅在调用栈同步执行，不靠sleep等待后台任务。
	if err := WithConfigLock(t.Context(), func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 0 {
		t.Errorf("fresh AT triggered %d refreshes", calls.Load())
	}
}

// TestDiscoveryCannotDelegateIssuer 即使metadata内端点互相同源，也不能把整个issuer换成第三方。
func TestDiscoveryCannotDelegateIssuer(t *testing.T) {
	isolateHome(t)
	md := DiscoveryMetadata{Issuer: "https://attacker.invalid", AuthorizationEndpoint: "https://attacker.invalid/authorize", TokenEndpoint: "https://attacker.invalid/token", RegistrationEndpoint: "https://attacker.invalid/register", RevocationEndpoint: "https://attacker.invalid/revoke"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(md); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	if _, _, err := resolveDiscovery(t.Context(), server.URL+"/.well-known/oauth-authorization-server", io.Discard); err == nil {
		t.Fatal("discovery changed trusted issuer origin")
	}
}
func TestTokenResponseRejectsUnknownScope(t *testing.T) {
	_, cleanup := setupTestTokenDir(t)
	defer cleanup()
	for _, scope := range []string{"unknown", "stock admin", "read", "base stock\nadmin"} {
		t.Run(scope, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := json.NewEncoder(w).Encode(TokenResponse{AccessToken: "at", RefreshToken: "rt", TokenType: "Bearer", ExpiresIn: 3600, Scope: scope}); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			previous := setActiveEndpointsForTest(&Endpoints{BaseURL: server.URL, TokenEndpoint: server.URL + "/token"})
			defer setActiveEndpointsForTest(previous)
			if _, err := ExchangeCode(t.Context(), "code", "client", "callback", "verifier", "https://api.gateio.ws/api/v4"); err == nil {
				t.Fatal("unknown permission persisted")
			}
		})
	}
}

// TestOAuthErrorDoesNotEchoServerSecrets 错误响应也是不可信出口，不回显可能包含一次性凭据的描述。
func TestOAuthErrorDoesNotEchoServerSecrets(t *testing.T) {
	body := []byte(`{"error":"invalid_grant","error_description":"synthetic-sensitive-at synthetic-sensitive-rt Cookie=synthetic"}`)
	err := extractOAuth2Error(body, 400)
	if strings.Contains(err.Error(), "synthetic") {
		t.Fatal("OAuth error leaked remote response details")
	}
}
