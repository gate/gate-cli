package oauth2

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validMetadataJSON() []byte {
	return []byte(`{
  "issuer": "https://example.invalid/oauth",
  "authorization_endpoint": "https://example.invalid/oauth/pkce/oauth/authorize",
  "token_endpoint": "https://example.invalid/oauth/pkce/oauth/token",
  "registration_endpoint": "https://example.invalid/oauth/pkce/oauth/register",
  "revocation_endpoint": "https://example.invalid/oauth/pkce/oauth/revoke"
}`)
}

func stubServer(t *testing.T, body []byte, status int) *httptest.Server {
	t.Helper()
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(bytes.ReplaceAll(body, []byte("https://example.invalid"), []byte(server.URL)))
	}))
	return server
}

func isolateHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	ResetHomeCacheForTest()
	return home
}

func TestDiscoveryMetadata_Validate_AllFieldsRequired(t *testing.T) {
	cases := []struct {
		name    string
		md      DiscoveryMetadata
		wantErr string
	}{
		{"missing issuer", DiscoveryMetadata{
			AuthorizationEndpoint: "a", TokenEndpoint: "t", RegistrationEndpoint: "r", RevocationEndpoint: "v",
		}, "issuer"},
		{"missing token_endpoint", DiscoveryMetadata{
			Issuer: "i", AuthorizationEndpoint: "a", RegistrationEndpoint: "r", RevocationEndpoint: "v",
		}, "token_endpoint"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.md.validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}

	// Happy path: all endpoints must be well-formed URLs same-origin as the
	// issuer (review — reject MITM-crafted metadata pointing endpoints at a
	// different host).
	ok := DiscoveryMetadata{
		Issuer:                "https://example.invalid",
		AuthorizationEndpoint: "https://example.invalid/authorize",
		TokenEndpoint:         "https://example.invalid/token",
		RegistrationEndpoint:  "https://example.invalid/register",
		RevocationEndpoint:    "https://example.invalid/revoke",
	}
	assert.NoError(t, ok.validate())
}

func TestDiscoveryMetadata_Validate_IssuerOriginMismatch(t *testing.T) {
	// token_endpoint on a different host than issuer must be rejected.
	md := DiscoveryMetadata{
		Issuer:                "https://example.invalid",
		AuthorizationEndpoint: "https://example.invalid/authorize",
		TokenEndpoint:         "https://attacker.invalid/token",
		RegistrationEndpoint:  "https://example.invalid/register",
		RevocationEndpoint:    "https://example.invalid/revoke",
	}
	err := md.validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "token_endpoint")
	assert.Contains(t, err.Error(), "mismatch")
}

func TestValidateDiscoveryURL_SchemeAllowlist(t *testing.T) {
	cases := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"https ok", "https://example.com/.well-known/oauth-authorization-server", false},
		{"http loopback ok", "http://127.0.0.1:8080/.well-known/oauth-authorization-server", false},
		{"http localhost ok", "http://localhost/.well-known/oauth-authorization-server", false},
		{"http public ok", "http://example.com/.well-known/oauth-authorization-server", false},
		{"http dev host ok", "http://dev.example.test/.well-known/oauth-authorization-server", false},
		{"file scheme rejected", "file:///etc/passwd", true},
		{"ftp rejected", "ftp://example.com/", true},
		{"missing host rejected", "https:///.well-known/foo", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateDiscoveryURL(tc.url)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFetchDiscovery_HappyPath(t *testing.T) {
	srv := stubServer(t, validMetadataJSON(), 200)
	defer srv.Close()

	md, err := fetchDiscovery(context.Background(), srv.URL)
	require.NoError(t, err)
	assert.Equal(t, srv.URL+"/oauth", md.Issuer)
	assert.Equal(t, srv.URL+"/oauth/pkce/oauth/authorize", md.AuthorizationEndpoint)
}

func TestFetchDiscovery_Non2xxIsError(t *testing.T) {
	srv := stubServer(t, []byte(`not found`), 404)
	defer srv.Close()

	_, err := fetchDiscovery(context.Background(), srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestFetchDiscovery_MalformedJSONIsError(t *testing.T) {
	srv := stubServer(t, []byte(`not json`), 200)
	defer srv.Close()

	_, err := fetchDiscovery(context.Background(), srv.URL)
	require.Error(t, err)
}

func TestFetchDiscovery_IncompleteMetadataIsError(t *testing.T) {
	srv := stubServer(t, []byte(`{"issuer":"x"}`), 200)
	defer srv.Close()

	_, err := fetchDiscovery(context.Background(), srv.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required fields")
}

func TestCacheRoundTrip(t *testing.T) {
	home := isolateHome(t)
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".gate-cli"), 0700))

	md := DiscoveryMetadata{
		Issuer: "https://example.invalid/oauth", AuthorizationEndpoint: "a",
		TokenEndpoint: "t", RegistrationEndpoint: "r", RevocationEndpoint: "v",
	}
	require.NoError(t, writeDiscoveryCache("https://example.invalid/.well-known/x", md))

	got, has, err := readDiscoveryCache()
	require.NoError(t, err)
	require.True(t, has)
	assert.Equal(t, "https://example.invalid/.well-known/x", got.DiscoveryURL)
	assert.Equal(t, md, got.Metadata)
	assert.WithinDuration(t, time.Now().UTC(), got.FetchedAt, 5*time.Second)

	// Cache file permissions should be 0600, consistent with tokens.yaml.
	path := filepath.Join(home, ".gate-cli", "oauth2_discovery.json")
	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestCache_MatchesURL_AndFresh(t *testing.T) {
	now := time.Now().UTC()
	c := &discoveryCache{DiscoveryURL: "u", FetchedAt: now}
	assert.True(t, c.matchesURL("u"))
	assert.False(t, c.matchesURL("other"))
	assert.True(t, c.fresh(now.Add(1*time.Hour)))
	assert.False(t, c.fresh(now.Add(discoveryCacheTTL+time.Second)))

	var nilC *discoveryCache
	assert.False(t, nilC.matchesURL("u"))
	assert.False(t, nilC.fresh(now))
}

func TestResolveDiscovery_FreshCacheHitSkipsNetwork(t *testing.T) {
	home := isolateHome(t)
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".gate-cli"), 0700))

	// review 残留修复后缓存读取也走 validate()，因此 cache seed 必须是合法的
	// 同源 metadata，否则 resolveDiscovery 会拒绝复用缓存并尝试网络回源。
	md := DiscoveryMetadata{
		Issuer:                "https://cached.invalid",
		AuthorizationEndpoint: "https://cached.invalid/authorize",
		TokenEndpoint:         "https://cached.invalid/token",
		RegistrationEndpoint:  "https://cached.invalid/register",
		RevocationEndpoint:    "https://cached.invalid/revoke",
	}
	require.NoError(t, writeDiscoveryCache("https://cached.invalid/.well-known/x", md))

	// Point at a would-be 500 server: the fresh cache must prevent the fetch.
	srv := stubServer(t, []byte(`boom`), 500)
	defer srv.Close()

	var stderr bytes.Buffer
	got, stale, err := resolveDiscovery(context.Background(), "https://cached.invalid/.well-known/x", &stderr)
	require.NoError(t, err)
	assert.False(t, stale)
	assert.Equal(t, "https://cached.invalid", got.Issuer)
	assert.Empty(t, stderr.String())
}

func TestResolveDiscovery_URLMismatchRefetches(t *testing.T) {
	home := isolateHome(t)
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".gate-cli"), 0700))

	// Seed cache for URL-A with fully-qualified same-origin endpoints so the
	// post-review cache validator accepts it.
	cachedMD := DiscoveryMetadata{
		Issuer:                "https://a.invalid",
		AuthorizationEndpoint: "https://a.invalid/authorize",
		TokenEndpoint:         "https://a.invalid/token",
		RegistrationEndpoint:  "https://a.invalid/register",
		RevocationEndpoint:    "https://a.invalid/revoke",
	}
	require.NoError(t, writeDiscoveryCache("https://a.invalid/.well-known/x", cachedMD))

	// Ask for URL-B — must fetch fresh from srv, not use the URL-A cache.
	srv := stubServer(t, validMetadataJSON(), 200)
	defer srv.Close()

	var stderr bytes.Buffer
	got, stale, err := resolveDiscovery(context.Background(), srv.URL, &stderr)
	require.NoError(t, err)
	assert.False(t, stale)
	assert.Equal(t, srv.URL+"/oauth", got.Issuer, "must have fetched fresh, not returned the A-url cache")

	c, has, _ := readDiscoveryCache()
	require.True(t, has)
	assert.Equal(t, srv.URL, c.DiscoveryURL)
}

func TestResolveDiscovery_NetworkFailWithMatchingStaleCacheFallsBack(t *testing.T) {
	home := isolateHome(t)
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".gate-cli"), 0700))

	// Seed a matching cache entry with valid same-origin endpoints, then age
	// it past TTL by editing fetched_at. Stale fallback path also validates.
	staleMD := DiscoveryMetadata{
		Issuer:                "http://127.0.0.1:1",
		AuthorizationEndpoint: "http://127.0.0.1:1/authorize",
		TokenEndpoint:         "http://127.0.0.1:1/token",
		RegistrationEndpoint:  "http://127.0.0.1:1/register",
		RevocationEndpoint:    "http://127.0.0.1:1/revoke",
	}
	require.NoError(t, writeDiscoveryCache("http://127.0.0.1:1/unreachable", staleMD))
	path := filepath.Join(home, ".gate-cli", "oauth2_discovery.json")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var c discoveryCache
	require.NoError(t, json.Unmarshal(raw, &c))
	c.FetchedAt = time.Now().Add(-48 * time.Hour)
	data, _ := json.MarshalIndent(&c, "", "  ")
	require.NoError(t, os.WriteFile(path, data, 0600))

	var stderr bytes.Buffer
	got, stale, err := resolveDiscovery(context.Background(), "http://127.0.0.1:1/unreachable", &stderr)
	require.NoError(t, err)
	assert.True(t, stale, "should report stale-cache-used when network failed")
	assert.Equal(t, "http://127.0.0.1:1", got.Issuer)
	assert.Contains(t, stderr.String(), "using cached metadata")
}

func TestResolveDiscovery_NetworkFailWithoutCacheReturnsError(t *testing.T) {
	_ = isolateHome(t)

	// Unreachable URL + no cache → must return error (no silent fallback).
	var stderr bytes.Buffer
	_, _, err := resolveDiscovery(context.Background(), "http://127.0.0.1:1/unreachable", &stderr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no usable cache")
}

func TestResolveDiscovery_ForceRefreshBypassesFreshCache(t *testing.T) {
	home := isolateHome(t)
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".gate-cli"), 0700))
	t.Setenv(discoveryRefreshEnv, "1")

	cached := DiscoveryMetadata{
		Issuer: "https://cached.invalid", AuthorizationEndpoint: "a",
		TokenEndpoint: "t", RegistrationEndpoint: "r", RevocationEndpoint: "v",
	}
	srv := stubServer(t, validMetadataJSON(), 200)
	defer srv.Close()
	require.NoError(t, writeDiscoveryCache(srv.URL, cached))

	var stderr bytes.Buffer
	got, stale, err := resolveDiscovery(context.Background(), srv.URL, &stderr)
	require.NoError(t, err)
	assert.False(t, stale)
	assert.Equal(t, srv.URL+"/oauth", got.Issuer, "force refresh should pull from srv, not cache")
}

func TestEndpointsFromMetadata_UsesFixedDisplayCallback(t *testing.T) {
	md := &DiscoveryMetadata{
		Issuer:                "https://example.invalid/oauth/", // trailing slash must be trimmed
		AuthorizationEndpoint: "https://example.invalid/oauth/pkce/oauth/authorize",
		TokenEndpoint:         "https://example.invalid/oauth/pkce/oauth/token",
		RegistrationEndpoint:  "https://example.invalid/oauth/pkce/oauth/register",
		RevocationEndpoint:    "https://example.invalid/oauth/pkce/oauth/revoke",
	}
	ep := endpointsFromMetadata(md)
	assert.Equal(t, "https://example.invalid/oauth", ep.BaseURL)
	assert.Equal(t, "https://www.gate.com/cli-oauth-callback", ep.DisplayCallbackURL)
	assert.Equal(t, md.AuthorizationEndpoint, ep.AuthorizationEndpoint)
}

func TestLoadEndpoints_EndToEnd(t *testing.T) {
	home := isolateHome(t)
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".gate-cli"), 0700))
	srv := stubServer(t, validMetadataJSON(), 200)
	defer srv.Close()

	ensureActiveDiscoveryURL(t, srv.URL)

	var stderr bytes.Buffer
	ep, err := LoadEndpoints(&stderr)
	require.NoError(t, err)
	assert.Equal(t, srv.URL+"/oauth", ep.BaseURL)
	assert.Equal(t, "https://www.gate.com/cli-oauth-callback", ep.DisplayCallbackURL)
}

func TestResolveDiscoveryURL_DefaultWhenUnconfigured(t *testing.T) {
	_ = isolateHome(t)
	// No config.yaml at all.
	got := resolveDiscoveryURL()
	assert.Equal(t, DefaultDiscoveryURL, got)
}

// TestResolveDiscoveryURL_DefaultWhenOAuth2SectionMissing 覆盖缺少配置字段的场景：
// config.yaml 存在但没有 oauth2 段（或 discovery_url 为空）时，应回退到生产默认。
func TestResolveDiscoveryURL_DefaultWhenOAuth2SectionMissing(t *testing.T) {
	home := isolateHome(t)
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".gate-cli"), 0700))
	cfgYAML := []byte("default_profile: sample\ndefault_settle: usdt\n")
	require.NoError(t, os.WriteFile(filepath.Join(home, ".gate-cli", "config.yaml"), cfgYAML, 0600))
	got := resolveDiscoveryURL()
	assert.Equal(t, DefaultDiscoveryURL, got)
}

// TestFetchDiscovery_CrossOriginRedirectRefused covers the RFC 8414 §5 threat:
// an attacker 301s the entrypoint to a host they control; validate() cannot
// help because the redirected metadata is internally self-consistent. The
// CheckRedirect guard must refuse the cross-origin hop before we ever parse.
func TestFetchDiscovery_CrossOriginRedirectRefused(t *testing.T) {
	attacker := stubServer(t, validMetadataJSON(), 200)
	defer attacker.Close()

	entry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, attacker.URL, http.StatusMovedPermanently)
	}))
	defer entry.Close()

	_, err := fetchDiscovery(context.Background(), entry.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cross-origin redirect refused")
}

// TestFetchDiscovery_SameOriginRedirectAllowed ensures ordinary canonicalization
// (e.g. trailing-slash redirect) still works within the same origin.
func TestFetchDiscovery_SameOriginRedirectAllowed(t *testing.T) {
	var srv *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/redir", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, srv.URL+"/meta", http.StatusMovedPermanently)
	})
	mux.HandleFunc("/meta", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(bytes.ReplaceAll(validMetadataJSON(), []byte("https://example.invalid"), []byte(srv.URL)))
	})
	srv = httptest.NewServer(mux)
	defer srv.Close()

	md, err := fetchDiscovery(context.Background(), srv.URL+"/redir")
	require.NoError(t, err)
	assert.Equal(t, srv.URL+"/oauth", md.Issuer)
}
