package client

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gate/gate-cli/internal/config"
	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/stretchr/testify/require"
)

// Reject ambiguous or incomplete error envelopes without exposing arbitrary DTO fields.
func TestDecodeBusinessError(t *testing.T) {
	for _, tc := range []struct {
		name, raw, label, message string
	}{
		{"upper", `{"Label":"BAD_REQUEST","Message":"meme token not found"}`, "BAD_REQUEST", "meme token not found"},
		{"lower", `{"label":"INVALID_ARGUMENT","message":"bad chain"}`, "INVALID_ARGUMENT", "bad chain"},
		{"mixed-and-extra", `{"LaBeL":" BAD_REQUEST ","mEsSaGe":" original ","private_debug":{"secret":"do not print"},"data":[1]}`, " BAD_REQUEST ", " original "},
		{"detail-not-message", `{"label":"BAD_REQUEST","message":"original","detail":"do not substitute"}`, "BAD_REQUEST", "original"},
		{"missing-label", `{"message":"bad"}`, "", ""},
		{"missing-message", `{"label":"BAD_REQUEST"}`, "", ""},
		{"empty-label", `{"label":" ","message":"bad"}`, "", ""},
		{"empty-message", `{"label":"BAD_REQUEST","message":"\t"}`, "", ""},
		{"null-label", `{"label":null,"message":"bad"}`, "", ""},
		{"null-message", `{"label":"BAD_REQUEST","message":null}`, "", ""},
		{"numeric-label", `{"label":1,"message":"bad"}`, "", ""},
		{"object-message", `{"label":"BAD_REQUEST","message":{}}`, "", ""},
		{"duplicate-label", `{"label":"BAD_REQUEST","Label":"OTHER","message":"bad"}`, "", ""},
		{"duplicate-message", `{"label":"BAD_REQUEST","message":"bad","Message":"other"}`, "", ""},
		{"same-key-duplicate", `{"label":"BAD_REQUEST","label":"BAD_REQUEST","message":"bad"}`, "", ""},
		{"nested", `{"data":{"label":"BAD_REQUEST","message":"bad"}}`, "", ""},
		{"array", `[{"label":"BAD_REQUEST","message":"bad"}]`, "", ""},
		{"null", `null`, "", ""}, {"empty", ``, "", ""}, {"html", `<html>failure</html>`, "", ""},
		{"truncated", `{"label":"BAD_REQUEST","message":"bad"`, "", ""},
		{"trailing-object", `{"label":"BAD_REQUEST","message":"bad"} {}`, "", ""},
		{"trailing-garbage", `{"label":"BAD_REQUEST","message":"bad"} x`, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := []byte(tc.raw)
			before := bytes.Clone(raw)
			label, message, ok := decodeBusinessError(raw)
			require.Equal(t, tc.label != "", ok)
			require.Equal(t, tc.label, label)
			require.Equal(t, tc.message, message)
			require.Equal(t, before, raw, "parser must not modify the SDK body")
		})
	}
	const raw = `{"label":"BAD_REQUEST","message":"bad"}`
	for _, extra := range []int{0, 1} {
		body := []byte(raw + strings.Repeat(" ", int(maxAPIResponseBytes)-len(raw)+extra))
		_, _, ok := decodeBusinessError(body)
		require.Equal(t, extra == 0, ok, "exact transport bound and one byte over")
	}
}

// Obtain genuine SDK failures, including their private raw body, at the HTTP peer.
func TestParseBusinessErrorSDK(t *testing.T) {
	for _, raw := range []string{
		`{"Label":"BAD_REQUEST","Message":"meme token not found","private_debug":"secret-sentinel"}`,
		`{"label":"BAD_REQUEST","message":"meme token not found"}`,
		`{"LaBeL":"BAD_REQUEST","mEsSaGe":"meme token not found"}`,
		`{"private_debug":"secret-sentinel"}`,
		`{"Label":"BAD_REQUEST","Message":"meme token not found"} {}`,
	} {
		t.Run(raw, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Gate-Trace-Id", "business-error-trace")
				_, writeErr := io.WriteString(w, raw)
				if writeErr != nil {
					t.Error(writeErr)
				}
			}))
			t.Cleanup(server.Close)
			c, err := New(&config.Config{BaseURL: server.URL}, "alpha/market/tickers")
			require.NoError(t, err)
			_, response, sdkErr := c.AlphaAPI.ListAlphaTickers(t.Context(), nil)
			require.Error(t, sdkErr)
			var generic gateapi.GenericOpenAPIError
			require.True(t, errors.As(sdkErr, &generic))
			before := bytes.Clone(generic.Body())
			for _, failure := range []error{generic, &generic, fmt.Errorf("wrapped: %w", generic), fmt.Errorf("wrapped pointer: %w", &generic)} {
				detail := ParseGateError(failure, response, "GET", "/alpha/tickers", "")
				_, _, valid := decodeBusinessError([]byte(raw))
				if valid {
					require.Equal(t, "BAD_REQUEST", detail.Label)
					require.Equal(t, "meme token not found", detail.Message)
				} else {
					require.Equal(t, "INVALID_RESPONSE", detail.Label)
					require.Contains(t, detail.Message, "cannot decode API response:")
				}
				require.NotContains(t, detail.Message, "secret-sentinel")
				require.Equal(t, 200, detail.Status)
				require.Equal(t, "business-error-trace", detail.TraceID)
				require.Equal(t, "GET", detail.Request.Method)
				require.Equal(t, "/alpha/tickers", detail.Request.URL)
			}
			require.Equal(t, before, generic.Body())
			// Non-2xx and absent responses retain the existing generic body behavior.
			for _, resp := range []*http.Response{nil, {StatusCode: 400}} {
				detail := ParseGateError(generic, resp, "GET", "/alpha/tickers", "")
				require.Empty(t, detail.Label)
				require.Equal(t, raw, detail.Message)
			}
		})
	}
}

// Wrapped SDK structured errors keep SDK Detail semantics; nil SDK pointers stay safe.
func TestParseGateErrorWrappedAndNilSDK(t *testing.T) {
	api := gateapi.GateAPIError{Label: "INVALID_PARAM", Message: "original", Detail: "specific detail"}
	for _, err := range []error{api, &api, fmt.Errorf("wrapped: %w", api), fmt.Errorf("wrapped: %w", &api)} {
		detail := ParseGateError(err, &http.Response{StatusCode: 400}, "POST", "/test", "")
		require.Equal(t, "INVALID_PARAM", detail.Label)
		require.Equal(t, "specific detail", detail.Message)
	}
	for _, err := range []error{nil, (*gateapi.GateAPIError)(nil), (*gateapi.GenericOpenAPIError)(nil)} {
		detail := ParseGateError(err, nil, "GET", "/test", "")
		require.Equal(t, 500, detail.Status)
		require.Equal(t, "unknown API error", detail.Message)
	}
}
