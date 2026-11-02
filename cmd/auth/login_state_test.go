//go:build agentic_e2e

package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gate/gate-cli/internal/oauth2"
	"github.com/spf13/cobra"
)

// TestLoginLocalReuseAndExpiredRefresh 本地有效凭据短路；过期凭据通过真实HTTP刷新并保留服务端最终scope。
func TestLoginLocalReuseAndExpiredRefresh(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	oauth2.ResetHomeCacheForTest()
	defer oauth2.ResetHomeCacheForTest()
	var tokens atomic.Int32
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/.well-known/oauth-authorization-server" {
			if err := json.NewEncoder(w).Encode(map[string]string{"issuer": server.URL, "authorization_endpoint": server.URL + "/authorize", "token_endpoint": server.URL + "/token", "registration_endpoint": server.URL + "/register", "revocation_endpoint": server.URL + "/revoke"}); err != nil {
				t.Error(err)
			}
			return
		}
		if r.URL.Path != "/token" {
			t.Errorf("unexpected network request %s", r.URL.Path)
			w.WriteHeader(500)
			return
		}
		tokens.Add(1)
		if err := json.NewEncoder(w).Encode(oauth2.TokenResponse{AccessToken: "new-at", RefreshToken: "new-rt", TokenType: "Bearer", Scope: "stock", ExpiresIn: 3600}); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	t.Setenv("GATE_BASE_URL", server.URL)
	if err := os.MkdirAll(filepath.Join(home, ".gate-cli"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLI_AGENTIC_TEST_ISSUER", server.URL)

	token := &oauth2.TokenData{ClientID: "synthetic", DiscoveryURL: server.URL, AccessToken: "old-at", RefreshToken: "old-rt", Scopes: "base stock", Resource: server.URL + "/api/v4", ExpiresAt: time.Now().Add(time.Hour)}
	command := func() *cobra.Command {
		c := &cobra.Command{}
		c.SetContext(context.Background())
		c.SetOut(&bytes.Buffer{})
		c.Flags().Bool("manual", true, "")
		c.Flags().Bool("force", false, "")
		c.Flags().String("scope", "", "")
		return c
	}
	if err := oauth2.SaveToken(token); err != nil {
		t.Fatal(err)
	}
	if err := runLogin(command(), nil); err != nil {
		t.Fatal(err)
	}
	if tokens.Load() != 0 {
		t.Fatal("local reuse refreshed")
	}
	token.ExpiresAt = time.Now().Add(-time.Hour)
	if err := oauth2.SaveToken(token); err != nil {
		t.Fatal(err)
	}
	if err := runLogin(command(), nil); err != nil {
		t.Fatal(err)
	}
	actual, err := oauth2.LoadToken()
	if err != nil || actual.Scopes != "stock" || actual.AccessToken != "new-at" || tokens.Load() != 1 {
		t.Fatal("expired login lost final scope or repeated refresh", err)
	}
	// 非法scope必须在读取凭据/发现之前报错；地址错误也不能发网络请求。
	bad := command()
	if err := bad.Flags().Set("scope", "unknown"); err != nil {
		t.Fatal(err)
	}
	if err := runLogin(bad, nil); err == nil {
		t.Fatal("invalid scope accepted")
	}
	t.Setenv("GATE_BASE_URL", "https://user:password@example.invalid")
	if err := runLogin(command(), nil); err == nil {
		t.Fatal("unsafe destination accepted")
	}
}
