package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gate/gate-cli/internal/config"
	"github.com/gate/gate-cli/internal/output"
	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/stretchr/testify/require"
)

// SDK decoding remains real: only Gate's HTTP peer is replaced. A well-formed
// business rejection must not become a successful zero-valued SDK result.
func TestFuturesResponseSDK(t *testing.T) {
	for _, operation := range []string{"trail-create", "trail-list", "price-create", "price-update"} {
		for _, raw := range []string{
			`{"code":-1,"message":"Invalid Fund password","data":{}}`,
			`{"code":1013,"Label":"AUTO_INVALID_PARAM_INITIAL_TIF","Message":"original failure"}`,
			`{"code":1077,"label":"APIOrderNotSupportUpdateTouchOrder","message":"original failure"}`,
			`{"Label":"BAD_REQUEST","Message":"original failure"}`,
		} {
			// A bare -1 is confirmed for Trail, not a new Price Trigger contract.
			if strings.HasPrefix(operation, "price") && strings.Contains(raw, "Fund password") {
				continue
			}
			t.Run(operation+raw, func(t *testing.T) {
				var calls atomic.Int32
				peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("X-Gate-Trace-Id", "fixture-trace")
					_, _ = io.WriteString(w, raw)
				}))
				t.Cleanup(peer.Close)
				c, err := New(&config.Config{BaseURL: peer.URL, APIKey: "fixture-key", APISecret: "fixture-secret"}, "futures")
				require.NoError(t, err)
				_, response, callErr := futuresFixtureCall(t.Context(), c, operation)
				require.Error(t, callErr, "business rejection decoded as success")
				failure := CommandError(callErr, response, "POST", "/fixture", "")
				var detail *output.GateError
				require.True(t, errors.As(failure, &detail))
				require.Equal(t, 200, detail.Status)
				require.Equal(t, "fixture-trace", detail.TraceID)
				var expected struct {
					Code           *int
					Label, Message string
				}
				require.NoError(t, json.Unmarshal([]byte(raw), &expected))
				require.Equal(t, expected.Code, detail.BusinessCode)
				require.Equal(t, expected.Label, detail.Label)
				require.Equal(t, expected.Message, detail.Message)
				output.FillAgentErrorConvergence(detail)
				if operation != "trail-list" {
					require.False(t, detail.Retryable)
				}
				require.EqualValues(t, 1, calls.Load())
			})
		}
	}
}

// Replay the documented string codes through the real SDK, including every
// adapter route. A success identity or private extension cannot hide a rejection.
func TestFuturesResponseStringCodesSDK(t *testing.T) {
	for _, tc := range []struct {
		name, raw, label, message string
		code                      int
	}{
		{"original-1013", `{"code":"1013","label":"AUTO_INVALID_PARAM_INITIAL_TIF","message":"invalid argument: market order tif must ioc"}`, "AUTO_INVALID_PARAM_INITIAL_TIF", "invalid argument: market order tif must ioc", 1013},
		{"original-1014", `{"code":"1014","label":"AUTO_INVALID_PARAM_INITIAL_SIZE","message":"invalid argument: initial.size plan close long position size must less zero"}`, "AUTO_INVALID_PARAM_INITIAL_SIZE", "invalid argument: initial.size plan close long position size must less zero", 1014},
		{"uppercase-and-identity", `{"Code":"1077","Label":"APIOrderNotSupportUpdateTouchOrder","Message":" original rejection ","id":9007199254740993,"data":{"id":"9007199254740993","token":"secret-canary"},"private_debug":"secret-canary"}`, "APIOrderNotSupportUpdateTouchOrder", " original rejection ", 1077},
		{"negative", `{"code":"-1","label":"BAD_REQUEST","message":"original rejection"}`, "BAD_REQUEST", "original rejection", -1},
		{"minimum", `{"code":"-2147483648","label":"BAD_REQUEST","message":"original rejection"}`, "BAD_REQUEST", "original rejection", -2147483648},
		{"maximum", `{"code":"2147483647","label":"BAD_REQUEST","message":"original rejection"}`, "BAD_REQUEST", "original rejection", 2147483647},
	} {
		for _, operation := range []string{"trail-create", "trail-list", "price-create", "price-update"} {
			t.Run(tc.name+"/"+operation, func(t *testing.T) {
				var calls atomic.Int32
				peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("X-Gate-Trace-Id", "string-code-trace")
					_, _ = io.WriteString(w, tc.raw)
				}))
				t.Cleanup(peer.Close)
				c, err := New(&config.Config{BaseURL: peer.URL, APIKey: "fixture-key", APISecret: "fixture-secret"}, "futures")
				require.NoError(t, err)
				_, response, callErr := futuresFixtureCall(t.Context(), c, operation)
				require.Error(t, callErr)
				var detail *output.GateError
				require.ErrorAs(t, CommandError(callErr, response, "POST", "/fixture", ""), &detail)
				require.Equal(t, tc.label, detail.Label)
				require.Equal(t, tc.message, detail.Message)
				require.NotNil(t, detail.BusinessCode)
				require.Equal(t, tc.code, *detail.BusinessCode)
				require.Equal(t, 200, detail.Status)
				require.Equal(t, "string-code-trace", detail.TraceID)
				output.FillAgentErrorConvergence(detail)
				if operation != "trail-list" {
					require.False(t, detail.Retryable)
					require.Contains(t, detail.SuggestedNextAction, "query")
				}
				encoded, err := json.Marshal(detail)
				require.NoError(t, err)
				require.NotContains(t, string(encoded), "secret-canary")
				require.EqualValues(t, 1, calls.Load())
			})
		}
	}
}

// String support is restricted to canonical nonzero int32 error codes. Invalid
// values and ambiguous fields must retain the existing protocol rejection.
func TestFuturesResponseStringCodeProtocol(t *testing.T) {
	const route = "/api/v4/futures/usdt/price_orders"
	for _, rawCode := range []string{`"0"`, `"-0"`, `""`, `" "`, `" 1013"`, `"1013 "`, `"+1013"`, `"01013"`, `"-01"`, `"1.0"`, `"1e3"`, `"future"`, `"2147483648"`, `"-2147483649"`, `"9223372036854775808"`, `null`, `true`, `[]`, `{}`} {
		t.Run(rawCode, func(t *testing.T) {
			raw := `{"code":` + rawCode + `,"label":"BAD_REQUEST","message":"original rejection","id":1,"private_debug":"secret-canary"}`
			response, body := futuresTransportResponse(t, "POST", route, raw, 200, nil)
			_, err := io.ReadAll(response.Body)
			require.NoError(t, response.Body.Close())
			require.Equal(t, 1, body.closes)
			var detail *output.GateError
			require.ErrorAs(t, err, &detail)
			require.Equal(t, "INVALID_RESPONSE", detail.Label)
			require.Nil(t, detail.BusinessCode)
			require.NotContains(t, detail.Error(), "secret-canary")
		})
	}
	for _, raw := range []string{
		`{"code":"1013","Code":0,"label":"BAD_REQUEST","message":"original"}`,
		`{"code":0,"Code":"1013","label":"BAD_REQUEST","message":"original"}`,
		`{"code":"1013","label":"BAD_REQUEST","Label":"OTHER","message":"original"}`,
		`{"code":"1013","label":"BAD_REQUEST","message":"original","Message":"other"}`,
	} {
		t.Run(raw, func(t *testing.T) {
			response, _ := futuresTransportResponse(t, "POST", route, raw, 200, nil)
			_, err := io.ReadAll(response.Body)
			require.NoError(t, response.Body.Close())
			var detail *output.GateError
			require.ErrorAs(t, err, &detail)
			require.Equal(t, "INVALID_RESPONSE", detail.Label)
		})
	}
}

func futuresFixtureCall(ctx context.Context, c *Client, operation string) (any, *http.Response, error) {
	switch operation {
	case "trail-create":
		return c.FuturesAPI.CreateTrailOrder(ctx, "usdt", gateapi.CreateTrailOrder{Contract: "SYNTH_USDT", Amount: "1", PriceOffset: "1%"})
	case "trail-list":
		return c.FuturesAPI.GetTrailOrders(ctx, "usdt", nil)
	case "price-create":
		return c.FuturesAPI.CreatePriceTriggeredOrder(ctx, "usdt", gateapi.FuturesPriceTriggeredOrder{})
	default:
		return c.FuturesAPI.UpdatePriceTriggeredOrder(ctx, "usdt", "pending-2147483648", gateapi.FuturesUpdatePriceTriggeredOrder{OrderId: 9007199254740993})
	}
}

type futuresRoundTripper func(*http.Request) (*http.Response, error)

func (f futuresRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type futuresReadCloser struct {
	io.Reader
	closes   int
	closeErr error
}

func (b *futuresReadCloser) Close() error { b.closes++; return b.closeErr }

// Use the existing transport entry point so this suite can run RED before the
// new private response adapter exists; it also verifies the real route guard.
func futuresTransportResponse(t *testing.T, method, route, raw string, status int, body *futuresReadCloser) (*http.Response, *futuresReadCloser) {
	t.Helper()
	if body == nil {
		body = &futuresReadCloser{Reader: strings.NewReader(raw)}
	}
	transport := &apiTransport{resource: "https://fixture.test/api/v4", hmac: true, base: futuresRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: status, Body: body, Header: http.Header{"X-Gate-Trace-Id": []string{"fixture-trace"}}}, nil
	})}
	req, err := http.NewRequestWithContext(t.Context(), method, "https://fixture.test"+route, nil)
	require.NoError(t, err)
	response, err := transport.RoundTrip(req)
	require.NoError(t, err)
	return response, body
}

func TestFuturesResponseProtocol(t *testing.T) {
	const create = "/api/v4/futures/usdt/autoorder/v1/trail/create"
	const list = "/api/v4/futures/usdt/autoorder/v1/trail/list"
	const price = "/api/v4/futures/usdt/price_orders"
	for _, tc := range []struct {
		name, method, route, raw string
		bad                      bool
	}{
		{"create-success", "POST", create, `{"code":0,"message":"ok","data":{"id":"9007199254740993"}} `, false},
		{"create-unknown-extension", "POST", create, `{"code":0,"extra":{"anything":1}}`, false},
		{"empty-list", "GET", list, `{"orders":[]}`, false},
		{"flat-code-200", "POST", price, `{"code":200,"message":"ok","id":9007199254740993}`, false},
		{"flat-code-string-extension", "POST", price, `{"code":"future","id":1}`, false},
		{"list-string-extension-without-label", "GET", list, `{"code":"-1","message":"future","orders":[]}`, false},
		{"flat-partial-error", "POST", price, `{"label":"future","id":1}`, false},
		{"unknown-empty", "POST", price, `{}`, false}, {"unknown-null", "POST", price, `null`, false},
		{"malformed-sdk-path", "POST", price, `{`, false}, {"trailing-sdk-path", "POST", price, `{"label":"BAD","message":"bad"} {}`, false},
		{"create-failure", "POST", create, `{"code":-1,"message":"original"}`, true},
		{"missing-message", "POST", create, `{"code":-1}`, true},
		{"empty-message", "POST", create, `{"code":-1,"message":""}`, true},
		{"null-message", "POST", create, `{"code":-1,"message":null}`, true},
		{"code-minimum", "POST", create, `{"code":-2147483648,"message":"original"}`, true},
		{"code-maximum", "POST", create, `{"code":2147483647,"message":"original"}`, true},
		{"list-missing-message", "GET", list, `{"code":-1}`, true},
		{"code-string", "POST", create, `{"code":"0"}`, true}, {"code-null", "POST", create, `{"code":null}`, true},
		{"create-string-error-without-label", "POST", create, `{"code":"-1","message":"original"}`, true},
		{"code-fraction", "POST", create, `{"code":0.5}`, true}, {"code-overflow", "POST", create, `{"code":2147483648}`, true},
		{"code-underflow", "POST", create, `{"code":-2147483649}`, true},
		{"duplicate-code", "POST", create, `{"code":-1,"Code":0,"message":"original"}`, true},
		{"duplicate-list-code", "GET", list, `{"code":-1,"code":0,"message":"original"}`, true},
		{"duplicate-label", "POST", price, `{"label":"BAD","Label":"OTHER","message":"original"}`, true},
		{"duplicate-message", "POST", price, `{"label":"BAD","message":"original","Message":"other"}`, true},
		{"bad-error-code", "POST", price, `{"label":"BAD","message":"original","code":null}`, true},
		{"conflicting-success", "POST", price, `{"label":"BAD","message":"original","code":0}`, true},
		{"bad-message-type", "POST", create, `{"code":-1,"message":[],"label":"BAD"}`, true},
		{"bad-label-type", "GET", list, `{"code":-1,"message":"original","label":{}}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response, original := futuresTransportResponse(t, tc.method, tc.route, tc.raw, 200, nil)
			raw, err := io.ReadAll(response.Body)
			require.NoError(t, response.Body.Close())
			require.Equal(t, 1, original.closes)
			if !tc.bad {
				require.NoError(t, err)
				require.Equal(t, tc.raw, string(raw))
				return
			}
			require.Error(t, err)
			var detail *output.GateError
			require.True(t, errors.As(err, &detail))
			require.Equal(t, 200, detail.Status)
			require.Equal(t, "fixture-trace", detail.TraceID)
			require.Equal(t, tc.route, detail.Request.URL)
			if tc.name == "empty-message" || tc.name == "missing-message" {
				require.Equal(t, "API returned a business rejection", detail.Message)
			}
			if tc.name == "code-minimum" || tc.name == "code-maximum" {
				want := int(-2147483648)
				if tc.name == "code-maximum" {
					want = 2147483647
				}
				require.NotNil(t, detail.BusinessCode)
				require.Equal(t, want, *detail.BusinessCode)
				require.Empty(t, detail.Label)
			}
		})
	}
}

// SDK v178 declares the stop/update/log envelopes; their nonzero-code paths
// are covered by TestSDK178. This table retains only untouched contracts.
func TestFuturesResponseScope(t *testing.T) {
	for _, tc := range []struct {
		method, route string
		status        int
	}{
		{"POST", "/api/v4/futures/usdt/autoorder/v1/trail/stop_all", 200},
		{"GET", "/api/v4/futures/usdt/autoorder/v1/trail/detail", 200},
		{"GET", "/api/v4/futures/usdt/price_orders/1", 200},
		{"POST", "/api/v4/bot/portfolio/stop", 200},
		{"POST", "/api/v4/futures/usdt/price_orders/extra", 200},
		{"POST", "/api/v4/futures/usdt/price_orders", 400},
		{"POST", "/api/v4/futures/usdt/price_orders", 403},
		{"POST", "/api/v4/futures/usdt/price_orders", 503},
	} {
		t.Run(tc.method+tc.route+http.StatusText(tc.status), func(t *testing.T) {
			const raw = `{"code":-1,"message":"unchanged"}`
			response, original := futuresTransportResponse(t, tc.method, tc.route, raw, tc.status, nil)
			require.Zero(t, original.closes)
			data, err := io.ReadAll(response.Body)
			require.NoError(t, err)
			require.Equal(t, raw, string(data))
			require.NoError(t, response.Body.Close())
		})
	}
}

func TestFuturesResponsePartialIdentity(t *testing.T) {
	for _, id := range []string{`"9007199254740993"`, `1`, `null`, `""`, `{}`} {
		raw := `{"code":-1,"message":" original ","data":{"id":` + id + `,"token":"secret-canary"},"private_debug":"secret-canary"}`
		response, _ := futuresTransportResponse(t, "POST", "/api/v4/futures/usdt/autoorder/v1/trail/create", raw, 200, nil)
		_, err := io.ReadAll(response.Body)
		require.NoError(t, response.Body.Close())
		var detail *output.GateError
		require.True(t, errors.As(err, &detail))
		require.Equal(t, " original ", detail.Message)
		if strings.HasPrefix(id, `"900`) {
			require.JSONEq(t, `{"id":"9007199254740993"}`, string(detail.Data))
		} else {
			require.Empty(t, detail.Data)
		}
		encoded, marshalErr := json.Marshal(detail)
		require.NoError(t, marshalErr)
		require.NotContains(t, string(encoded), "secret-canary")
		output.FillAgentErrorConvergence(detail)
		require.False(t, detail.Retryable)
	}
}

type futuresBrokenReader struct{}

func (futuresBrokenReader) Read([]byte) (int, error) { return 0, errors.New("private-read-canary") }

func TestFuturesResponseIO(t *testing.T) {
	const route = "/api/v4/futures/usdt/autoorder/v1/trail/create"
	for _, body := range []*futuresReadCloser{
		{Reader: futuresBrokenReader{}},
		{Reader: strings.NewReader(`{"code":0}`), closeErr: errors.New("private-close-canary")},
		{Reader: strings.NewReader(`{"code":0}` + strings.Repeat(" ", int(maxAPIResponseBytes)))},
	} {
		response, _ := futuresTransportResponse(t, "POST", route, "", 200, body)
		_, err := io.ReadAll(response.Body)
		require.Error(t, err)
		require.Equal(t, 1, body.closes)
		require.NotContains(t, err.Error(), "private-")
		require.NoError(t, response.Body.Close())
	}
	raw := `{"code":0}`
	raw += strings.Repeat(" ", int(maxAPIResponseBytes)-len(raw))
	response, body := futuresTransportResponse(t, "POST", route, raw, 200, nil)
	data, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, raw, string(data))
	require.NoError(t, response.Body.Close())
	require.Equal(t, 1, body.closes)
}

// Deterministic interleaving exposes cross-request error/identity contamination;
// no global mutable fixture, random sleep or live account is involved.
func TestFuturesResponseConcurrent(t *testing.T) {
	for _, fields := range []string{`"code":-1`, `"code":"-1","label":"BAD_REQUEST"`} {
		t.Run(fields, func(t *testing.T) { testFuturesResponseConcurrent(t, fields) })
	}
}

// Exercise the same controlled interleaving for both code representations;
// response-local parsing must not exchange another request's cause or identity.
func testFuturesResponseConcurrent(t *testing.T, codeFields string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	entered := make(chan string, 2)
	releases := map[string]chan struct{}{"first": make(chan struct{}), "second": make(chan struct{})}
	peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Contract string }
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		entered <- req.Contract
		select {
		case <-releases[req.Contract]:
		case <-r.Context().Done():
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Gate-Trace-Id", req.Contract)
		_, _ = io.WriteString(w, `{`+codeFields+`,"message":"`+req.Contract+`","data":{"id":"`+req.Contract+`"}}`)
	}))
	t.Cleanup(peer.Close)
	c, err := NewWithContext(ctx, &config.Config{BaseURL: peer.URL, APIKey: "fixture", APISecret: "fixture"}, "futures")
	require.NoError(t, err)
	type result struct {
		name string
		err  error
	}
	done := make(chan result, 2)
	for _, name := range []string{"first", "second"} {
		go func() {
			_, _, err := c.FuturesAPI.CreateTrailOrder(ctx, "usdt", gateapi.CreateTrailOrder{Contract: name})
			done <- result{name, err}
		}()
	}
	for range 2 {
		select {
		case <-entered:
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	for _, name := range []string{"second", "first"} {
		close(releases[name])
		var r result
		select {
		case r = <-done:
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
		var detail *output.GateError
		if r.name != name || !errors.As(r.err, &detail) {
			t.Errorf("missing scoped error for %s: %v", name, r.err)
			continue
		}
		require.Equal(t, name, detail.Message)
		require.Equal(t, name, detail.TraceID)
		require.JSONEq(t, `{"id":"`+name+`"}`, string(detail.Data))
	}
}

func TestFuturesResponseContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	ready := make(chan struct{})
	var calls atomic.Int32
	peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = io.WriteString(w, `{`)
		w.(http.Flusher).Flush()
		close(ready)
		<-r.Context().Done()
	}))
	t.Cleanup(peer.Close)
	c, err := NewWithContext(ctx, &config.Config{BaseURL: peer.URL, APIKey: "fixture", APISecret: "fixture"}, "futures")
	require.NoError(t, err)
	done := make(chan error, 1)
	go func() { _, _, err := futuresFixtureCall(ctx, c, "trail-create"); done <- err }()
	select {
	case <-ready:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	cancel()
	select {
	case err := <-done:
		require.Error(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("canceled response reader did not exit")
	}
	require.EqualValues(t, 1, calls.Load())
}

func TestFuturesResponseMissingBody(t *testing.T) {
	transport := &apiTransport{resource: "https://fixture.test/api/v4", hmac: true, base: futuresRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header)}, nil
	})}
	req, err := http.NewRequestWithContext(t.Context(), "POST", "https://fixture.test/api/v4/futures/usdt/price_orders", nil)
	require.NoError(t, err)
	response, err := transport.RoundTrip(req)
	require.NoError(t, err)
	require.NotNil(t, response.Body)
	_, err = io.ReadAll(response.Body)
	require.Error(t, err)
	require.NoError(t, response.Body.Close())
}
