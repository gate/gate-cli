package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/iotest"
	"time"

	"github.com/gate/gate-cli/internal/config"
	"github.com/gate/gate-cli/internal/output"
	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Replay a synthetic fixture based on the actual response structure through the
// released SDK. IDs, account, times and suborder text are invented; field types and
// cancellation semantics remain representative. The interface carrier keeps
// SDK178 compiling so the regression fails on data loss.
func TestSDK179TrailStopAllNativeContracts(t *testing.T) {
	fixture, err := os.ReadFile("testdata/trail_stop_all.json")
	require.NoError(t, err)
	large := strings.ReplaceAll(string(fixture), `"71001"`, `"9007199254740993"`)
	multiple := strings.Replace(string(fixture), `"orders": [`, `"orders": [{"id":"9007199254740993","amount":"0.01","status_code":"cancelled"},`, 1)
	for _, tc := range []struct {
		name, raw, wantID    string
		status, code, orders int
		fail                 bool
	}{
		{"synthetic-cancellation", string(fixture), "71001", 200, 0, 1, false},
		{"large-string-id", large, "9007199254740993", 200, 0, 1, false},
		{"multiple-orders", multiple, "9007199254740993", 200, 0, 2, false},
		{"above-int64-string-id", strings.ReplaceAll(string(fixture), `"71001"`, `"18446744073709551615"`), "18446744073709551615", 200, 0, 1, false},
		{"empty", `{"code":0,"message":"ok","data":{"orders":[]},"timestamp":1700000001837}`, "", 200, 0, 0, false},
		{"business-code-retained", `{"code":1077,"message":"fixture rejection","data":{},"timestamp":1700000001837}`, "", 200, 1077, 0, false},
		{"http400", `{"label":"BAD_REQUEST","message":"fixture rejection"}`, "", 400, 0, 0, true},
		{"http503", `{"label":"SERVER_ERROR","message":"fixture rejection"}`, "", 503, 0, 0, true},
		{"malformed", `{"data":`, "", 200, 0, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/api/v4/futures/usdt/autoorder/v1/trail/stop_all", r.URL.Path)
				assert.Empty(t, r.URL.RawQuery)
				raw, readErr := io.ReadAll(io.LimitReader(r.Body, 4097))
				assert.NoError(t, readErr)
				assert.Equal(t, "{\"contract\":\"BTC_USDT\"}\n", string(raw))
				assert.Equal(t, "sdk179-key", r.Header.Get("KEY"))
				assert.NotEmpty(t, r.Header.Get("SIGN"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.raw)
			}))
			t.Cleanup(peer.Close)
			cfg := gateapi.NewConfiguration()
			cfg.BasePath, cfg.Key, cfg.Secret = peer.URL+"/api/v4", "sdk179-key", "sdk179-secret"
			cfg.HTTPClient = &http.Client{Timeout: 2 * time.Second}
			value, response, callErr := gateapi.NewAPIClient(cfg).FuturesApi.StopAllTrailOrders(t.Context(), "usdt", gateapi.StopAllTrailOrders{Contract: "BTC_USDT"})
			require.EqualValues(t, 1, calls.Load())
			require.NotNil(t, response)
			require.Equal(t, tc.status, response.StatusCode)
			if tc.fail {
				require.Error(t, callErr)
				return
			}
			require.NoError(t, callErr)
			raw, encodeErr := json.Marshal(any(value))
			require.NoError(t, encodeErr)
			var got struct {
				Code      int   `json:"code"`
				Timestamp int64 `json:"timestamp"`
				Data      struct {
					Orders []struct {
						ID         string `json:"id"`
						Amount     string `json:"amount"`
						StatusCode string `json:"status_code"`
					} `json:"orders"`
				} `json:"data"`
			}
			require.NoError(t, json.Unmarshal(raw, &got))
			require.Equal(t, tc.code, got.Code)
			var expected struct {
				Timestamp int64 `json:"timestamp"`
			}
			require.NoError(t, json.Unmarshal([]byte(tc.raw), &expected))
			require.Equal(t, expected.Timestamp, got.Timestamp, "SDK discarded or changed envelope time: %s", raw)
			require.Len(t, got.Data.Orders, tc.orders)
			if tc.orders != 0 {
				require.Equal(t, tc.wantID, got.Data.Orders[0].ID)
				require.Equal(t, "cancelled", got.Data.Orders[0].StatusCode)
				require.Equal(t, "0.01", got.Data.Orders[0].Amount)
				if tc.orders == 2 {
					require.Equal(t, "71001", got.Data.Orders[1].ID)
				}
			}
		})
	}
}

// Reuse the governed response transport. Native successful bytes are untouched;
// newly declared stop-all business failures cannot become a zero CLI exit.
func TestTrailStopAll179BusinessBoundary(t *testing.T) {
	for _, tc := range []struct {
		name, raw, label string
		fail             bool
	}{
		{"success", `{"code":0,"message":"ok","label":"future","data":{"orders":[]},"timestamp":1700000001837,"unknown":true}`, "", false},
		{"business-without-label", `{"code":1077,"message":"fixture rejection","data":{},"private_debug":"sdk179-private-canary"}`, "", true},
		{"business-with-label", `{"code":-1,"label":"TRAIL_REJECTED","message":"fixture rejection","data":{},"private_debug":"sdk179-private-canary"}`, "TRAIL_REJECTED", true},
		{"ambiguous", `{"code":-1,"Code":0,"message":"fixture rejection"}`, "INVALID_RESPONSE", true},
		{"invalid-zero-string", `{"code":"0","message":"fixture rejection"}`, "INVALID_RESPONSE", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const route = "/api/v4/futures/usdt/autoorder/v1/trail/stop_all"
			response, original := futuresTransportResponse(t, http.MethodPost, route, tc.raw, 200, nil)
			raw, err := io.ReadAll(response.Body)
			require.NoError(t, response.Body.Close())
			if !tc.fail {
				require.NoError(t, err)
				require.Equal(t, tc.raw, string(raw))
				require.Equal(t, 1, original.closes)
				return
			}
			var detail *output.GateError
			require.ErrorAs(t, err, &detail)
			require.Equal(t, tc.label, detail.Label)
			require.Equal(t, 1, original.closes)
			require.NotNil(t, detail.Request)
			require.Equal(t, http.MethodPost, detail.Request.Method)
			require.Equal(t, route, detail.Request.URL)
			require.Equal(t, "fixture-trace", detail.TraceID)
			if strings.HasPrefix(tc.name, "business-") {
				require.NotNil(t, detail.BusinessCode)
			}
			output.FillAgentErrorConvergence(detail)
			require.False(t, detail.Retryable)
			require.NotEmpty(t, detail.SuggestedNextAction)
			encoded, encodeErr := json.Marshal(detail)
			require.NoError(t, encodeErr)
			require.NotContains(t, string(encoded), "sdk179-private-canary")
		})
	}
}

// The existing body owner must also cover this newly normalized route. A failed
// write read/close cannot silently turn into a successful empty SDK DTO.
func TestTrailStopAll179BodyOwnership(t *testing.T) {
	const route = "/api/v4/futures/usdt/autoorder/v1/trail/stop_all"
	for _, body := range []*futuresReadCloser{
		{Reader: futuresBrokenReader{}},
		{Reader: strings.NewReader(`{"code":0}`), closeErr: fmt.Errorf("private-close-canary")},
		{Reader: strings.NewReader(`{"code":0}` + strings.Repeat(" ", int(maxAPIResponseBytes)))},
	} {
		response, original := futuresTransportResponse(t, http.MethodPost, route, "", 200, body)
		_, err := io.ReadAll(response.Body)
		require.Error(t, err)
		require.Equal(t, 1, original.closes)
		require.NoError(t, response.Body.Close())
		require.NotContains(t, err.Error(), "private-")
	}

	raw := `{"code":0}`
	raw += strings.Repeat(" ", int(maxAPIResponseBytes)-len(raw))
	response, original := futuresTransportResponse(t, http.MethodPost, route, raw, 200, nil)
	bounded, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, raw, string(bounded))
	require.NoError(t, response.Body.Close())
	require.Equal(t, 1, original.closes)
}

// Cancellation while reading a successful HTTP body must retain its safe cause,
// close the network body once, and never be mistaken for a business rejection.
func TestTrailStopAll179BodyReadCauses(t *testing.T) {
	for _, cause := range []error{context.Canceled, context.DeadlineExceeded} {
		t.Run(cause.Error(), func(t *testing.T) {
			body := &futuresReadCloser{Reader: iotest.ErrReader(fmt.Errorf("private-peer-canary: %w", cause))}
			response, original := futuresTransportResponse(t, http.MethodPost, "/api/v4/futures/usdt/autoorder/v1/trail/stop_all", "", 200, body)
			_, err := io.ReadAll(response.Body)
			require.ErrorIs(t, err, cause)
			require.NotContains(t, err.Error(), "private-peer-canary")
			require.Equal(t, 1, original.closes)
			require.NoError(t, response.Body.Close())
			var detail *output.GateError
			require.ErrorAs(t, TrailCommandError(err, response, "stopAllTrailOrders", "usdt", ""), &detail)
			output.FillAgentErrorConvergence(detail)
			require.False(t, detail.Retryable)
			require.Contains(t, detail.Message, "outcome unknown")
			require.Contains(t, detail.SuggestedNextAction, "query")
			require.NotContains(t, detail.Message, "private-peer-canary")
		})
	}
}

// Instrument reads rather than sleeping: cancellation begins only after the
// real HTTP peer flushed 200 and the production normalizer started body I/O.
type trail179ObservedBody struct {
	io.ReadCloser
	entered chan struct{}
	once    sync.Once
	closes  atomic.Int32
}

func (body *trail179ObservedBody) Read(p []byte) (int, error) {
	body.once.Do(func() { close(body.entered) })
	return body.ReadCloser.Read(p)
}

func (body *trail179ObservedBody) Close() error {
	body.closes.Add(1)
	return body.ReadCloser.Close()
}

func TestSDK179TrailStopAllBodyReadCancellation(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		t.Run(fmt.Sprintf("deadline=%t", deadline), func(t *testing.T) {
			var calls atomic.Int32
			peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				_, _ = io.Copy(io.Discard, r.Body)
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"code":0,"data":{"orders":[`)
				w.(http.Flusher).Flush()
				<-r.Context().Done()
			}))
			t.Cleanup(peer.Close)
			ctx, cancel := context.WithCancel(t.Context())
			if deadline {
				cancel()
				ctx, cancel = context.WithTimeout(t.Context(), 3*time.Second)
			}
			defer cancel()
			readStarted := make(chan struct{})
			network := http.DefaultTransport.(*http.Transport).Clone()
			t.Cleanup(network.CloseIdleConnections)
			var observed *trail179ObservedBody
			cfg := gateapi.NewConfiguration()
			cfg.BasePath, cfg.Key, cfg.Secret = peer.URL+"/api/v4", "fixture-key", "fixture-secret"
			cfg.HTTPClient = &http.Client{Timeout: 5 * time.Second, Transport: &apiTransport{resource: cfg.BasePath, hmac: true,
				base: futuresRoundTripper(func(r *http.Request) (*http.Response, error) {
					response, err := network.RoundTrip(r)
					if response != nil && response.Body != nil {
						observed = &trail179ObservedBody{ReadCloser: response.Body, entered: readStarted}
						response.Body = observed
					}
					return response, err
				})}}
			result := make(chan error, 1)
			go func() {
				_, _, err := gateapi.NewAPIClient(cfg).FuturesApi.StopAllTrailOrders(ctx, "usdt", gateapi.StopAllTrailOrders{Contract: "BTC_USDT"})
				result <- err
			}()
			select {
			case <-readStarted:
			case <-time.After(2 * time.Second):
				cancel()
				t.Fatal("normalizer did not begin reading the response body")
			}
			want := error(context.Canceled)
			if deadline {
				want = context.DeadlineExceeded
			} else {
				cancel()
			}
			select {
			case err := <-result:
				require.ErrorIs(t, err, want)
				require.NotNil(t, observed)
				require.EqualValues(t, 1, observed.closes.Load())
			case <-time.After(5 * time.Second):
				cancel()
				t.Fatal("body cancellation did not return")
			}
			require.EqualValues(t, 1, calls.Load())
		})
	}
}

// Component contexts cover application cancellation, unlike merely killing a
// CLI child. Cancel/timeout once a peer has received the write; never replay it.
func TestSDK179TrailStopAllCancellation(t *testing.T) {
	for _, mode := range []string{"before-send", "in-flight", "deadline"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int32
			entered := make(chan struct{}, 1)
			peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				// Drain the small request so net/http can observe a disconnected client.
				_, _ = io.Copy(io.Discard, r.Body)
				entered <- struct{}{}
				<-r.Context().Done()
			}))
			t.Cleanup(peer.Close)
			ctx := t.Context()
			var cancel context.CancelFunc
			if mode == "deadline" {
				ctx, cancel = context.WithTimeout(ctx, 3*time.Second)
			} else {
				ctx, cancel = context.WithCancel(ctx)
			}
			defer cancel()
			if mode == "before-send" {
				cancel()
			}
			c, err := NewWithContext(ctx, &config.Config{BaseURL: peer.URL, APIKey: "fixture-key", APISecret: "fixture-secret"}, "futures/trail/stop-all")
			require.NoError(t, err)
			result := make(chan error, 1)
			go func() {
				_, _, err := c.FuturesAPI.StopAllTrailOrders(c.Context(), "usdt", gateapi.StopAllTrailOrders{Contract: "BTC_USDT"})
				result <- err
			}()
			if mode != "before-send" {
				select {
				case <-entered:
				case <-time.After(2 * time.Second):
					cancel()
					t.Fatal("write did not reach peer")
				}
				if mode == "in-flight" {
					cancel()
				}
			}
			select {
			case err := <-result:
				expected := context.Canceled
				if mode == "deadline" {
					expected = context.DeadlineExceeded
				}
				require.True(t, errors.Is(err, expected), "lost context: %v", err)
			case <-time.After(5 * time.Second):
				cancel()
				t.Fatal("cancelled write did not return")
			}
			want := int32(1)
			if mode == "before-send" {
				want = 0
			}
			require.Equal(t, want, calls.Load())
		})
	}
}
