package client

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gate/gate-cli/internal/oauth2"

	"github.com/gate/gate-cli/internal/config"
	gateapi "github.com/gate/gateapi-go/v7"
)

// TestSDKWriteNeverFollowsRedirect 防止同站跳转把已处理的交易再次提交。
func TestSDKWriteNeverFollowsRedirect(t *testing.T) {
	for _, status := range []int{301, 302, 303, 307, 308} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var first, redirected atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/v4/futures/usdt/orders" {
					first.Add(1)
					w.Header().Set("Location", "/api/v4/futures/usdt/replayed")
					w.WriteHeader(status)
					return
				}
				redirected.Add(1)
				w.Header().Set("Content-Type", "application/json")
				if _, err := w.Write([]byte(`{"id":1}`)); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			c, err := New(&config.Config{BaseURL: server.URL, APIKey: "synthetic", APISecret: "synthetic"}, "futures/order/create")
			if err != nil {
				t.Fatal(err)
			}
			_, _, err = c.FuturesAPI.CreateFuturesOrder(context.Background(), "usdt", gateapi.FuturesOrder{Contract: "BTC_USDT", Size: "1"}, nil)
			if err == nil {
				t.Error("write redirect must return an error, not a successful trade")
			}
			if first.Load() != 1 || redirected.Load() != 0 {
				t.Errorf("write requests=%d redirected requests=%d; want 1/0", first.Load(), redirected.Load())
			}
		})
	}
}

// TestSDKBearerIsPrivateAndBound 验证真实SDK请求：公开匿名，私有带受众绑定Token，换站不发送。
func TestSDKBearerIsPrivateAndBound(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	oauth2.ResetHomeCacheForTest()
	defer oauth2.ResetHomeCacheForTest()
	var public, private atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v4/spot/currencies" {
			public.Add(1)
			if r.Header.Get("Authorization") != "" {
				t.Error("public request sent bearer")
			}
		} else {
			private.Add(1)
			if r.Header.Get("Authorization") != "Bearer synthetic-bound-at" {
				t.Error("private request lost bearer")
			}
			if r.Header.Get("KEY") != "" {
				t.Error("mixed credentials")
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`[]`)); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	token := &oauth2.TokenData{ClientID: "synthetic", AccessToken: "synthetic-bound-at", RefreshToken: "synthetic-rt", Resource: server.URL + "/api/v4", ExpiresAt: time.Now().Add(time.Hour)}
	if err := oauth2.SaveToken(token); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(config.Options{})
	if err != nil {
		t.Fatal(err)
	}
	cfg.BaseURL = server.URL
	c, err := New(cfg, "spot/account")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.SpotAPI.ListCurrencies(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.SpotAPI.ListSpotAccounts(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	for _, resource := range []string{"", "https://other.example.test/api/v4"} {
		token.Resource = resource
		if err := oauth2.SaveToken(token); err != nil {
			t.Fatal(err)
		}
		if _, _, err := c.SpotAPI.ListSpotAccounts(t.Context(), nil); err == nil {
			t.Fatal("unbound token reached API")
		}
	}
	if public.Load() != 1 || private.Load() != 1 {
		t.Errorf("requests public=%d private=%d want1/1", public.Load(), private.Load())
	}
}

func TestSDKDebugDoesNotDumpCredentials(t *testing.T) {
	var captured bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&captured)
	defer log.SetOutput(previous)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()
	c, err := New(&config.Config{BaseURL: server.URL, APIKey: "synthetic-sensitive-key", APISecret: "synthetic-sensitive-secret", Debug: true}, "spot/account")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.SpotAPI.ListSpotAccounts(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(captured.Bytes(), []byte("synthetic-sensitive-key")) {
		t.Fatal("SDK debug dumped API credential")
	}
}

// TestAPIRejectsAmbiguousPath 验证发送前拒绝路径归一化绕过，不能先把Bearer发出去。
func TestAPIRejectsAmbiguousPath(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(200) }))
	defer server.Close()
	transport := &apiTransport{resource: server.URL + "/api/v4", hmac: true, base: http.DefaultTransport}
	for _, path := range []string{"/api/v4/../private", "/api/v4/%2e%2e/private", "/api/v4/stock%2fusers/assets", "/api/v4//stock/users/assets", "/api/v4/stock/./users/assets"} {
		req, err := http.NewRequest("GET", server.URL+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := transport.RoundTrip(req)
		if response != nil {
			if err := response.Body.Close(); err != nil {
				t.Fatal(err)
			}
		}
		if err == nil {
			t.Errorf("ambiguous path sent: %s", path)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("unsafe wire requests=%d", calls.Load())
	}
}

// TestAPIResponseReadBound 验证实际SDK读取前即受统一边界限制；大响应不能无界占用内存。
func TestAPIResponseReadBound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, strings.Repeat(" ", int(maxAPIResponseBytes)+1)); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	c, err := New(&config.Config{BaseURL: server.URL}, "stock/market/exchanges")
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = c.StockAPI.QueryStockExchanges(t.Context())
	if err == nil || !strings.Contains(err.Error(), "response exceeds") {
		t.Fatalf("expected response bound error: %v", err)
	}
}
