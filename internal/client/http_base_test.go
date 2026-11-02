package client

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gate/gate-cli/internal/config"
	"github.com/gate/gate-cli/internal/oauth2"
)

// TestSDKNonLoopbackHTTP exercises the real SDK/auth boundary with a private HTTP
// authority. Only dialing is redirected to the local fixture; credentials never
// leave this machine, while the HTTP Host, URL and resource remain unchanged.
func TestSDKNonLoopbackHTTP(t *testing.T) {
	const base = "http://10.0.0.1:18080"
	const key, secret = "synthetic-http-key", "synthetic-http-secret"
	t.Setenv("HOME", t.TempDir())
	oauth2.ResetHomeCacheForTest()
	t.Cleanup(oauth2.ResetHomeCacheForTest)
	var public, signed, bearer, redirected atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "10.0.0.1:18080" {
			t.Error("fixture lost the configured private HTTP authority")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v4/spot/currencies":
			public.Add(1)
			if r.Header.Get("Authorization") != "" || r.Header.Get("KEY") != "" {
				t.Error("anonymous HTTP request sent credentials")
			}
		case "/api/v4/spot/accounts":
			if r.Header.Get("KEY") != "" {
				signed.Add(1)
				bodyHash := sha512.Sum512(nil)
				message := strings.Join([]string{r.Method, r.URL.Path, r.URL.RawQuery, hex.EncodeToString(bodyHash[:]), r.Header.Get("Timestamp")}, "\n")
				signature := hmac.New(sha512.New, []byte(secret))
				_, _ = signature.Write([]byte(message)) // hash.Hash.Write never returns an error.
				if r.Header.Get("KEY") != key || r.Header.Get("Timestamp") == "" || !hmac.Equal([]byte(r.Header.Get("SIGN")), []byte(hex.EncodeToString(signature.Sum(nil)))) || r.Header.Get("Authorization") != "" {
					t.Error("HTTP request did not preserve the HMAC contract")
				}
			} else {
				bearer.Add(1)
				if r.Header.Get("Authorization") != "Bearer synthetic-http-at" || r.Header.Get("SIGN") != "" || r.Header.Get("Timestamp") != "" {
					t.Error("HTTP request did not preserve the Bearer contract")
				}
			}
		case "/redirect-host/api/v4/spot/currencies", "/redirect-scheme/api/v4/spot/currencies":
			redirected.Add(1)
			target := "http://other.example.test/api/v4/spot/currencies"
			if strings.HasPrefix(r.URL.Path, "/redirect-scheme/") {
				target = "https://10.0.0.1:18080/api/v4/spot/currencies"
			}
			w.Header().Set("Location", target)
			w.WriteHeader(http.StatusFound)
			return
		default:
			t.Error("unexpected SDK path")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if _, err := io.WriteString(w, "[]"); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)

	// This package's tests are serial. Reuse the production transport boundary and
	// restore it after all requests; no hosts file, external proxy or DNS is changed.
	previous := sharedAPITransport
	transport := previous.Clone()
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		if address != "10.0.0.1:18080" {
			t.Error("request escaped the configured HTTP authority")
			return nil, fmt.Errorf("unexpected fixture destination")
		}
		return (&net.Dialer{Timeout: time.Second}).DialContext(ctx, network, server.Listener.Addr().String())
	}
	sharedAPITransport = transport
	t.Cleanup(func() { sharedAPITransport = previous; transport.CloseIdleConnections() })

	publicClient, err := New(&config.Config{BaseURL: base}, "spot/market/currencies")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := publicClient.SpotAPI.ListCurrencies(t.Context()); err != nil {
		t.Fatal(err)
	}
	keyClient, err := New(&config.Config{BaseURL: base, APIKey: key, APISecret: secret}, "spot/account")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := keyClient.SpotAPI.ListSpotAccounts(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	token := &oauth2.TokenData{ClientID: "synthetic-http-client", Resource: base + "/api/v4", AccessToken: "synthetic-http-at", ExpiresAt: time.Now().Add(time.Hour), Scopes: "base"}
	if err := oauth2.SaveToken(token); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{BaseURL: base, OAuth2: config.OAuth2File{Token: &config.TokenEntry{ClientID: token.ClientID, Resource: token.Resource, AccessToken: token.AccessToken}}}
	bearerClient, err := New(cfg, "spot/account")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := bearerClient.SpotAPI.ListSpotAccounts(t.Context(), nil); err != nil {
		t.Fatal(err)
	}

	// Allowing HTTP must not let a credential issued for another scheme, host or
	// port reach the server. Check the wire count, not only the returned error.
	for _, resource := range []string{"https://10.0.0.1:18080/api/v4", "http://10.0.0.2:18080/api/v4", "http://10.0.0.1:18081/api/v4"} {
		token.Resource = resource
		if err := oauth2.SaveToken(token); err != nil {
			t.Fatal(err)
		}
		if _, _, err := bearerClient.SpotAPI.ListSpotAccounts(t.Context(), nil); err == nil {
			t.Error("mismatched resource reached the HTTP API")
		}
	}
	for _, prefix := range []string{"/redirect-host", "/redirect-scheme"} {
		c, err := New(&config.Config{BaseURL: base + prefix}, "spot/market/currencies")
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := c.SpotAPI.ListCurrencies(t.Context()); err == nil || !strings.Contains(err.Error(), "API redirect escaped trusted origin") {
			t.Errorf("HTTP redirect did not preserve the origin boundary: %v", err)
		}
	}
	if public.Load() != 1 || signed.Load() != 1 || bearer.Load() != 1 || redirected.Load() != 2 {
		t.Fatalf("wire counts public/signed/bearer/redirect = %d/%d/%d/%d", public.Load(), signed.Load(), bearer.Load(), redirected.Load())
	}
}
