package oauth2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gate/gate-cli/internal/config"
)

// DiscoveryMetadata is the subset of RFC 8414 OAuth 2.0 Authorization Server
// Metadata that gate-cli needs to drive its PKCE flow. Other metadata fields
// (scopes_supported, grant_types_supported, etc.) are intentionally ignored —
// they are informational and we don't want the discovery doc to become a
// source of silent behavior drift.
type DiscoveryMetadata struct {
	Issuer                string `json:"issuer"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	RegistrationEndpoint  string `json:"registration_endpoint"`
	RevocationEndpoint    string `json:"revocation_endpoint"`
}

// validate enforces the minimum set of URLs required to run the PKCE flow
// and their origin binding to the issuer. Missing fields surface as a clear
// "discovery document is incomplete" error; mismatched origins surface as
// a "origin mismatch" error so that a hostile / compromised metadata
// document cannot redirect token exchange to an attacker-controlled host
// (review, defense against DNS hijack / MITM downgrade).
func (m *DiscoveryMetadata) validate() error {
	var missing []string
	if strings.TrimSpace(m.Issuer) == "" {
		missing = append(missing, "issuer")
	}
	if strings.TrimSpace(m.AuthorizationEndpoint) == "" {
		missing = append(missing, "authorization_endpoint")
	}
	if strings.TrimSpace(m.TokenEndpoint) == "" {
		missing = append(missing, "token_endpoint")
	}
	if strings.TrimSpace(m.RegistrationEndpoint) == "" {
		missing = append(missing, "registration_endpoint")
	}
	if strings.TrimSpace(m.RevocationEndpoint) == "" {
		missing = append(missing, "revocation_endpoint")
	}
	if len(missing) > 0 {
		return fmt.Errorf("discovery document missing required fields: %s", strings.Join(missing, ", "))
	}

	issuerU, err := url.Parse(m.Issuer)
	if err != nil || (issuerU.Scheme != "https" && issuerU.Scheme != "http") || issuerU.Host == "" || issuerU.User != nil || issuerU.RawQuery != "" || issuerU.Fragment != "" {
		return fmt.Errorf("invalid issuer URL %q", m.Issuer)
	}
	// RFC 8414 §3.3: all endpoint URLs SHOULD be same-origin as the issuer.
	// We enforce it: the CLI refuses to talk to a token_endpoint on a different
	// host/scheme than the issuer advertises.
	for _, ep := range []struct{ name, raw string }{
		{"authorization_endpoint", m.AuthorizationEndpoint},
		{"token_endpoint", m.TokenEndpoint},
		{"registration_endpoint", m.RegistrationEndpoint},
		{"revocation_endpoint", m.RevocationEndpoint},
	} {
		u, err := url.Parse(ep.raw)
		if err != nil || u.Scheme == "" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return fmt.Errorf("invalid %s URL %q", ep.name, ep.raw)
		}
		if !strings.EqualFold(u.Scheme, issuerU.Scheme) || !strings.EqualFold(u.Host, issuerU.Host) {
			return fmt.Errorf("%s origin (%s://%s) mismatches issuer origin (%s://%s)",
				ep.name, u.Scheme, u.Host, issuerU.Scheme, issuerU.Host)
		}
	}
	return nil
}

// validateFor 绑定调用方配置的发现站点，metadata不能通过自声明issuer把凭据导向第三方。
func (m *DiscoveryMetadata) validateFor(discovery string) error {
	if err := m.validate(); err != nil {
		return err
	}
	trusted, err := url.Parse(discovery)
	if err != nil {
		return err
	}
	issuer, _ := url.Parse(m.Issuer)
	if !strings.EqualFold(trusted.Scheme, issuer.Scheme) || !strings.EqualFold(trusted.Host, issuer.Host) {
		return fmt.Errorf("discovery issuer origin mismatch")
	}
	return nil
}

// validateDiscoveryURL enforces a scheme allow-list on the discovery URL
// itself: both https and http are accepted for any host so operators can
// point at internal dev environments served over plain HTTP. file://, ftp://
// and other non-HTTP schemes are still rejected to keep obviously dangerous
// URL shapes out of the fetch path. The stronger SSRF guard (metadata
// same-origin validation and cross-origin redirect refusal) remains in place.
func validateDiscoveryURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid discovery URL %q: %w", raw, err)
	}
	scheme := strings.ToLower(u.Scheme)
	if u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("discovery URL %q has no host", raw)
	}
	switch scheme {
	case "https", "http":
		return nil
	default:
		return fmt.Errorf("discovery URL scheme %q not allowed (must be https or http)", u.Scheme)
	}
}

// sameOriginRedirect rejects redirects that cross scheme or host relative to
// the original request (first entry in via). Up to 5 same-origin redirects
// are allowed to accommodate ordinary HTTPS canonicalization; beyond that
// the chain is treated as suspicious and refused.
func sameOriginRedirect(req *http.Request, via []*http.Request) error {
	if len(via) == 0 {
		return nil
	}
	if len(via) >= 5 {
		return fmt.Errorf("too many redirects (%d) fetching discovery", len(via))
	}
	orig := via[0].URL
	if !strings.EqualFold(req.URL.Scheme, orig.Scheme) || !strings.EqualFold(req.URL.Host, orig.Host) {
		return fmt.Errorf("cross-origin redirect refused: %s://%s → %s://%s",
			orig.Scheme, orig.Host, req.URL.Scheme, req.URL.Host)
	}
	return nil
}

// discoveryCache is the on-disk shape of ~/.gate-cli/oauth2_discovery.json.
// FetchedAt uses RFC 3339 via time.Time's default JSON marshalling.
type discoveryCache struct {
	DiscoveryURL string            `json:"discovery_url"`
	FetchedAt    time.Time         `json:"fetched_at"`
	Metadata     DiscoveryMetadata `json:"metadata"`
}

const (
	// discoveryCacheTTL is how long a cached discovery document stays fresh.
	// 24h means day-over-day CLI use hits the cache exactly once.
	discoveryCacheTTL = 24 * time.Hour

	// discoveryHTTPTimeout bounds how long a single discovery request waits.
	// 5s matches typical well-known fetch latency with headroom for TLS.
	discoveryHTTPTimeout = 5 * time.Second

	// discoveryRefreshEnv, when set to any non-empty value, forces a
	// live fetch and bypasses the cache (even fresh cache entries).
	discoveryRefreshEnv = "GATE_CLI_DISCOVERY_REFRESH"
)

// discoveryCachePath returns ~/.gate-cli/oauth2_discovery.json.
func discoveryCachePath() (string, error) {
	home, err := resolveHome()
	if err != nil {
		return "", fmt.Errorf("oauth2 discovery cache path: %w", err)
	}
	return filepath.Join(home, ".gate-cli", "oauth2_discovery.json"), nil
}

// readDiscoveryCache returns the cached discovery document (regardless of
// freshness) and whether it was present at all. A missing file is not an
// error — first-run users have no cache.
func readDiscoveryCache() (*discoveryCache, bool, error) {
	path, err := discoveryCachePath()
	if err != nil {
		return nil, false, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("read discovery cache: %w", err)
	}
	var c discoveryCache
	if err := json.Unmarshal(data, &c); err != nil {
		// Treat a malformed cache as "no cache" rather than a fatal error — we
		// don't want a corrupted file to brick every subsequent CLI call.
		return nil, false, nil
	}
	return &c, true, nil
}

// writeDiscoveryCache persists the metadata keyed by discoveryURL to disk with
// 0600 perms so subsequent runs can hit a warm cache. Tmp + rename so a
// partial crash never leaves a half-written JSON in place.
func writeDiscoveryCache(discoveryURL string, md DiscoveryMetadata) error {
	path, err := discoveryCachePath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}
	// Review: refuse to overwrite a symlink — Lstat does not follow links.
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to write through symlink: %s", path)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("lstat cache: %w", err)
	}
	c := discoveryCache{
		DiscoveryURL: discoveryURL,
		FetchedAt:    time.Now().UTC(),
		Metadata:     md,
	}
	data, err := json.MarshalIndent(&c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal discovery cache: %w", err)
	}

	// 缓存和凭据统一复用配置文件的原子替换，避免重复维护落盘协议。
	return config.WriteAtomic(path, data, 0600)
}

// matchesURL reports whether the cache entry keys to discoveryURL. Used to
// distinguish "cache for this environment" from "cache for another env".
func (c *discoveryCache) matchesURL(discoveryURL string) bool {
	return c != nil && c.DiscoveryURL == discoveryURL
}

// fresh reports whether the cache entry is within TTL. Stale-but-matching
// entries can still serve as a fallback when the network is down.
func (c *discoveryCache) fresh(now time.Time) bool {
	if c == nil {
		return false
	}
	return now.Sub(c.FetchedAt) < discoveryCacheTTL
}

// fetchDiscovery does a single HTTP GET for the RFC 8414 metadata document at
// discoveryURL. Returns the parsed metadata on success. The 5s timeout is
// generous enough for slow TLS handshakes but short enough to not make the
// CLI feel sluggish on a broken network.
//
// Redirects: a cross-origin redirect is refused. RFC 8414 §5 identifies a
// hostile redirect as an attack vector — if an attacker can 301 the
// well-known URL to evil.com, they fully control the metadata payload and
// validate() cannot help (issuer/endpoint are all self-declared). Same-origin
// redirects are allowed so operators can use internal 301 aliases.
func fetchDiscovery(ctx context.Context, discoveryURL string) (*DiscoveryMetadata, error) {
	if err := validateDiscoveryURL(discoveryURL); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build discovery request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", ClientName)

	client := &http.Client{
		Timeout:       discoveryHTTPTimeout,
		CheckRedirect: sameOriginRedirect,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", discoveryURL, err)
	}
	defer closeOAuthBody(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GET %s returned HTTP %d", discoveryURL, resp.StatusCode)
	}

	// Cap the read so a pathological server can't OOM the CLI. Real metadata
	// docs are a few KB at most.
	body, err := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if err != nil {
		return nil, fmt.Errorf("read discovery body: %w", err)
	}
	if len(body) > 1<<20 {
		return nil, fmt.Errorf("discovery body too large")
	}
	var md DiscoveryMetadata
	if err := json.Unmarshal(body, &md); err != nil {
		return nil, fmt.Errorf("parse discovery JSON: %w", err)
	}
	if err := md.validateFor(discoveryURL); err != nil {
		return nil, err
	}
	return &md, nil
}

// resolveDiscovery produces the DiscoveryMetadata used to populate
// activeEndpoints, handling cache hit/miss/stale/network-failure. Behavior:
//
//  1. GATE_CLI_DISCOVERY_REFRESH non-empty → skip cache, go straight to fetch
//  2. Fresh cache matching discoveryURL → use it, no network
//  3. Stale or URL-mismatched cache → attempt fetch
//     - Fetch OK → use fetched, refresh cache
//     - Fetch fails AND stale cache for SAME URL exists → use stale cache
//     + warn on stderr (second return value true)
//     - Fetch fails AND no cache for this URL → return error (no silent
//     fallback to hardcoded values, per explicit design choice)
func resolveDiscovery(ctx context.Context, discoveryURL string, stderr io.Writer) (*DiscoveryMetadata, bool, error) {
	cached, hasCache, _ := readDiscoveryCache()
	forceRefresh := strings.TrimSpace(os.Getenv(discoveryRefreshEnv)) != ""
	now := time.Now().UTC()

	// review 残留修复：缓存路径也必须走 validate()。攻击者若能写入
	// ~/.gate-cli/oauth2_discovery.json（供应链 / 已登录的恶意进程），可以
	// 注入一个跨源 token_endpoint 的 metadata，绕过 fetch 路径的同源校验。
	// 以下两个返回点（fresh hit 与 stale fallback）都在返回前做 validate。
	if !forceRefresh && hasCache && cached.matchesURL(discoveryURL) && cached.fresh(now) {
		md := cached.Metadata
		if err := md.validateFor(discoveryURL); err != nil {
			log.New(stderr, "", 0).Printf("Warning: cached OAuth2 discovery failed validation (%v); re-fetching\n", err)
		} else {
			return &md, false, nil
		}
	}

	md, fetchErr := fetchDiscovery(ctx, discoveryURL)
	if fetchErr == nil {
		if err := writeDiscoveryCache(discoveryURL, *md); err != nil {
			// Cache write failure is non-fatal; next call will simply re-fetch.
			log.New(stderr, "", 0).Printf("Warning: could not update OAuth2 discovery cache: %v\n", err)
		}
		return md, false, nil
	}

	// Fetch failed. If we have a cache entry for this exact URL — even stale —
	// use it so the CLI remains usable offline. Mismatched-URL caches are NOT
	// reused: the user explicitly configured a different environment.
	if hasCache && cached.matchesURL(discoveryURL) {
		md := cached.Metadata
		if err := md.validateFor(discoveryURL); err != nil {
			// Tampered / broken cache is useless as a fallback — fall through
			// to the hard-fail branch below.
			log.New(stderr, "", 0).Printf("Warning: stale cached metadata failed validation (%v); cannot fall back\n", err)
		} else {
			log.New(stderr, "", 0).Printf("Warning: OAuth2 discovery refresh failed (%v); using cached metadata from %s\n",
				fetchErr, cached.FetchedAt.Local().Format(time.RFC3339))
			return &md, true, nil
		}
	}

	return nil, false, fmt.Errorf(
		"OAuth2 discovery failed and no usable cache for %s: %w\n"+
			"  Set GATE_CLI_DISCOVERY_REFRESH=1 and retry when network is back,\n"+
			"  or check connectivity to the fixed production OAuth discovery endpoint",
		discoveryURL, fetchErr,
	)
}
