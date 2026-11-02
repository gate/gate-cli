package client

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/gate/gate-cli/internal/oauth2"
	"github.com/gate/gate-cli/internal/operations"
)

type retirementRoundTripper func(*http.Request) (*http.Response, error)

func (f retirementRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// TestRetiredOperationsNeverSend exercises the real common transport before
// auth/refresh and send. No real credentials, OAuth issuer or API are contacted.
func TestRetiredOperationsNeverSend(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	oauth2.ResetHomeCacheForTest()
	t.Cleanup(oauth2.ResetHomeCacheForTest)
	raw, err := os.ReadFile("../operations/interface.json")
	if err != nil {
		t.Fatal(err)
	}
	var catalog struct {
		Operations []operations.Operation `json:"operations"`
	}
	if err := json.Unmarshal(raw, &catalog); err != nil {
		t.Fatal(err)
	}
	variable := regexp.MustCompile(`\{[^/{}]+\}`)
	for _, mode := range []string{"anonymous", "hmac", "oauth-old-scopes"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			transport := &apiTransport{resource: "https://api.example.test/api/v4", hmac: mode == "hmac", oauth: mode == "oauth-old-scopes",
				base: retirementRoundTripper(func(*http.Request) (*http.Response, error) {
					calls++
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("[]"))}, nil
				})}
			blocked := 0
			for _, op := range catalog.Operations {
				path := variable.ReplaceAllString(op.Path, "synthetic")
				disabled, err := operations.IsDisabled(op.Method, path)
				if err != nil {
					t.Fatal(err)
				}
				if !disabled {
					continue
				}
				req, err := http.NewRequest(op.Method, "https://api.example.test"+path, nil)
				if err != nil {
					t.Fatal(err)
				}
				resp, err := transport.RoundTrip(req)
				if err == nil || !strings.Contains(err.Error(), "unavailable in this CLI release") || resp != nil {
					t.Fatalf("%s %s reached auth/send: response=%v err=%v", op.Method, path, resp, err)
				}
				blocked++
			}
			if blocked != 81 || calls != 0 {
				t.Fatalf("blocked=%d sent=%d want81/0", blocked, calls)
			}
			// The same boundary still forwards retained public market operations.
			for _, path := range []string{"/api/v4/spot/currencies", "/api/v4/earn/uni/currencies"} {
				req, err := http.NewRequest("GET", "https://api.example.test"+path, nil)
				if err != nil {
					t.Fatal(err)
				}
				resp, err := transport.RoundTrip(req)
				if err != nil {
					t.Fatal(err)
				}
				if err := resp.Body.Close(); err != nil {
					t.Fatal(err)
				}
			}
			if calls != 2 {
				t.Fatalf("retained sends=%d want2", calls)
			}
		})
	}
}
