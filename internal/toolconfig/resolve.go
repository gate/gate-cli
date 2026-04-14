package toolconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultHTTPTimeout = 60 * time.Second

// ResolveOptions controls endpoint resolution for intel facade commands.
type ResolveOptions struct {
	Backend string
}

// ResolvedEndpoint is the finalized endpoint config used by MCP client.
type ResolvedEndpoint struct {
	Backend      string
	BaseURL      string
	BearerToken  string
	ExtraHeaders map[string]string
	Timeout      time.Duration
}

// Resolve resolves endpoint config from environment variables.
func Resolve(opts ResolveOptions) (*ResolvedEndpoint, error) {
	backend := strings.ToLower(strings.TrimSpace(opts.Backend))
	if backend == "" {
		return nil, fmt.Errorf("intel backend is required")
	}

	envKey := "GATE_INTEL_" + strings.ToUpper(backend) + "_MCP_URL"
	baseURL := strings.TrimSpace(os.Getenv(envKey))
	if baseURL == "" {
		return nil, fmt.Errorf("intel endpoint URL is required for %s command", backend)
	}

	tokenKey := "GATE_INTEL_" + strings.ToUpper(backend) + "_BEARER_TOKEN"
	bearer := strings.TrimSpace(os.Getenv(tokenKey))
	if bearer == "" {
		bearer = strings.TrimSpace(os.Getenv("GATE_INTEL_BEARER_TOKEN"))
	}

	extraHeaders, err := parseExtraHeaders(os.Getenv("GATE_INTEL_EXTRA_HEADERS"))
	if err != nil {
		return nil, err
	}

	timeout, err := parseTimeoutEnv(os.Getenv("GATE_INTEL_HTTP_TIMEOUT"))
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
		out[key] = fmt.Sprint(v)
	}
	return out, nil
}

func parseTimeoutEnv(raw string) (time.Duration, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return defaultHTTPTimeout, nil
	}

	if d, err := time.ParseDuration(v); err == nil {
		return d, nil
	}

	seconds, err := strconv.Atoi(v)
	if err != nil || seconds <= 0 {
		return 0, fmt.Errorf("invalid GATE_INTEL_HTTP_TIMEOUT: %q", raw)
	}
	return time.Duration(seconds) * time.Second, nil
}
