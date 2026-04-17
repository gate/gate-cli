package toolconfig

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gate/gate-cli/internal/config"
)

const defaultHTTPTimeout = 60 * time.Second

var (
	headerNamePattern = regexp.MustCompile(`^[A-Za-z0-9-]+$`)
	deniedHeaders     = map[string]struct{}{
		"authorization": {},
		"host":          {},
		"content-length": {},
	}
)

// ResolveOptions controls endpoint resolution for intel facade commands.
type ResolveOptions struct {
	Backend   string
	IntelFile config.IntelFile // optional defaults from ~/.gate-cli/config.yaml "intel"
}

// ResolvedEndpoint is the finalized endpoint config used by MCP client.
type ResolvedEndpoint struct {
	Backend      string
	BaseURL      string
	BearerToken  string
	ExtraHeaders map[string]string
	Timeout      time.Duration
}

// Resolve resolves endpoint config. Precedence: non-empty env, then config file intel.*,
// then config.DefaultIntel*MCPURL (public QC). Bearer / headers / timeout: env overrides file only.
func Resolve(opts ResolveOptions) (*ResolvedEndpoint, error) {
	backend := strings.ToLower(strings.TrimSpace(opts.Backend))
	if backend == "" {
		return nil, fmt.Errorf("intel backend is required")
	}
	if backend != "news" && backend != "info" {
		return nil, fmt.Errorf("intel endpoint URL is required for %s command", backend)
	}

	file := opts.IntelFile
	envKey := "GATE_INTEL_" + strings.ToUpper(backend) + "_MCP_URL"
	baseURL := strings.TrimSpace(os.Getenv(envKey))
	if baseURL == "" {
		switch backend {
		case "news":
			baseURL = strings.TrimSpace(file.NewsMCPURL)
		case "info":
			baseURL = strings.TrimSpace(file.InfoMCPURL)
		}
	}
	if baseURL == "" {
		switch backend {
		case "news":
			baseURL = config.DefaultIntelNewsMCPURL
		case "info":
			baseURL = config.DefaultIntelInfoMCPURL
		}
	}
	if err := validateBaseURL(baseURL); err != nil {
		return nil, err
	}

	tokenKey := "GATE_INTEL_" + strings.ToUpper(backend) + "_BEARER_TOKEN"
	bearer := strings.TrimSpace(os.Getenv(tokenKey))
	if bearer == "" {
		bearer = strings.TrimSpace(os.Getenv("GATE_INTEL_BEARER_TOKEN"))
	}
	if bearer == "" {
		switch backend {
		case "news":
			bearer = strings.TrimSpace(file.NewsBearerToken)
		case "info":
			bearer = strings.TrimSpace(file.InfoBearerToken)
		}
	}
	if bearer == "" {
		bearer = strings.TrimSpace(file.BearerToken)
	}

	extraHeaders, err := resolveExtraHeaders(os.Getenv("GATE_INTEL_EXTRA_HEADERS"), file.ExtraHeaders)
	if err != nil {
		return nil, err
	}

	timeout, err := resolveHTTPTimeout(os.Getenv("GATE_INTEL_HTTP_TIMEOUT"), file.HTTPTimeout)
	if err != nil {
		return nil, err
	}

	return &ResolvedEndpoint{
		Backend:      backend,
		BaseURL:      baseURL,
		BearerToken:  bearer,
		ExtraHeaders: extraHeaders,
		Timeout:      timeout,
	}, nil
}

func resolveExtraHeaders(envRaw string, fileHeaders map[string]string) (map[string]string, error) {
	if strings.TrimSpace(envRaw) != "" {
		return parseExtraHeaders(envRaw)
	}
	return cloneStringMap(fileHeaders), nil
}

func cloneStringMap(m map[string]string) map[string]string {
	if len(m) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func resolveHTTPTimeout(envRaw, fileRaw string) (time.Duration, error) {
	if strings.TrimSpace(envRaw) != "" {
		return parseNonEmptyIntelTimeout(envRaw, "GATE_INTEL_HTTP_TIMEOUT")
	}
	if strings.TrimSpace(fileRaw) != "" {
		return parseNonEmptyIntelTimeout(fileRaw, "intel.http_timeout")
	}
	return defaultHTTPTimeout, nil
}

func parseNonEmptyIntelTimeout(raw, field string) (time.Duration, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return 0, fmt.Errorf("internal: empty timeout for %s", field)
	}
	if d, err := time.ParseDuration(v); err == nil {
		return d, nil
	}
	seconds, err := strconv.Atoi(v)
	if err != nil || seconds <= 0 {
		return 0, fmt.Errorf("invalid %s: %q", field, raw)
	}
	return time.Duration(seconds) * time.Second, nil
}

func parseExtraHeaders(raw string) (map[string]string, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]string{}, nil
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, fmt.Errorf("invalid GATE_INTEL_EXTRA_HEADERS: %w", err)
	}

	out := make(map[string]string, len(decoded))
	for k, v := range decoded {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		if !headerNamePattern.MatchString(key) {
			return nil, fmt.Errorf("invalid GATE_INTEL_EXTRA_HEADERS key: %q", key)
		}
		lk := strings.ToLower(key)
		if _, denied := deniedHeaders[lk]; denied {
			return nil, fmt.Errorf("GATE_INTEL_EXTRA_HEADERS key is not allowed: %q", key)
		}
		val := fmt.Sprint(v)
		if strings.ContainsAny(val, "\r\n") {
			return nil, fmt.Errorf("invalid GATE_INTEL_EXTRA_HEADERS value for %q", key)
		}
		out[key] = val
	}
	return out, nil
}

func validateBaseURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("invalid intel endpoint url: %w", err)
	}
	if u.Host == "" {
		return fmt.Errorf("invalid intel endpoint url: missing host")
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme == "https" {
		return nil
	}
	if scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1") {
		return nil
	}
	return fmt.Errorf("invalid intel endpoint url: scheme must be https (or localhost http)")
}

