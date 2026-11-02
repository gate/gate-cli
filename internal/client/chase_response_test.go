package client_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/antihax/optional"
	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/config"
	"github.com/gate/gate-cli/internal/output"
	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/stretchr/testify/require"
)

// The deployed chase service wraps results and business errors inside HTTP 200.
func TestChaseResponseEnvelope(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		wantError  bool
	}{
		{"wrapped success", `{"code":0,"message":"ok","data":{"orders":[{"id":"9223372036854775807"}]}}`, false},
		{"HTTP200 rejection", `{"code":-1,"message":"Invalid request"}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Gate-Trace-Id", "fixture-trace")
				_, _ = w.Write([]byte(tc.body))
			}))
			defer peer.Close()
			c, err := client.New(&config.Config{BaseURL: peer.URL, APIKey: "fixture-key", APISecret: "fixture-secret"}, "futures/chase/list")
			require.NoError(t, err)
			result, resp, err := c.FuturesAPI.GetChaseOrders(c.Context(), "usdt", 1, &gateapi.GetChaseOrdersOpts{IsFinished: optional.NewBool(false)})
			if tc.wantError {
				require.Error(t, err)
				wrapped := client.CommandError(err, resp, "GET", "/api/v4/futures/usdt/autoorder/v1/chase/list", "")
				var detail *output.GateError
				require.True(t, errors.As(wrapped, &detail))
				require.Equal(t, 200, detail.Status)
				require.Equal(t, "Invalid request", detail.Message)
				require.Equal(t, -1, *detail.BusinessCode)
				require.Equal(t, "fixture-trace", detail.TraceID)
			} else {
				require.NoError(t, err)
				require.Len(t, result.Orders, 1)
				require.Equal(t, "9223372036854775807", result.Orders[0].Id)
			}
		})
	}
}
