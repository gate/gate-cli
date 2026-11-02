package client

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/gate/gate-cli/internal/output"
	"github.com/stretchr/testify/require"
)

type chaseTestBody struct {
	io.Reader
	closed   int
	closeErr error
}

func (b *chaseTestBody) Close() error { b.closed++; return b.closeErr }

// Check every deployed envelope/error boundary before generated SDK decoding.
func TestNormalizeChaseResponseBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, raw string
		bad       bool
	}{
		{"flat", `{"orders":[]}`, false}, {"wrapped", `{"code":0,"data":{"orders":[]}}`, false},
		{"business rejection", `{"code":-1,"message":"rejected","data":{"id":"partial"}}`, true},
		{"string code", `{"code":"0","data":{"orders":[]}}`, true},
		{"null code", `{"code":null,"data":{"orders":[]}}`, true},
		{"fractional code", `{"code":0.5,"data":{"orders":[]}}`, true},
		{"missing data", `{"code":0}`, true}, {"null data", `{"code":0,"data":null}`, true},
		{"array data", `{"code":0,"data":[]}`, true}, {"missing orders", `{"code":0,"data":{}}`, true},
		{"null orders", `{"orders":null}`, true}, {"object orders", `{"orders":{}}`, true},
		{"null item", `{"orders":[null]}`, true}, {"empty item", `{"orders":[{}]}`, true},
		{"malformed", `{`, true}, {"null", `null`, true},
		{"wrong message type", `{"code":-1,"message":[],"data":{}}`, true},
		{"wrong label type", `{"code":-1,"label":[],"data":{}}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := &chaseTestBody{Reader: strings.NewReader(tc.raw)}
			response := &http.Response{StatusCode: 200, Header: make(http.Header), Body: source}
			normalizeChaseResponse(response, "GET", "/api/v4/futures/usdt/autoorder/v1/chase/list")
			body, err := io.ReadAll(response.Body)
			require.NoError(t, response.Body.Close())
			require.Equal(t, 1, source.closed)
			if tc.bad {
				require.Error(t, err)
				var detail *output.GateError
				require.True(t, errors.As(err, &detail))
				require.Equal(t, 200, detail.Status)
			} else {
				require.NoError(t, err)
				require.JSONEq(t, `{"orders":[]}`, string(body))
				require.EqualValues(t, len(body), response.ContentLength)
			}
		})
	}
}
func TestChaseOperationResultShapes(t *testing.T) {
	for _, tc := range []struct {
		method, leaf, body string
		ok                 bool
	}{
		{"POST", "create", `{"id":"9223372036854775807"}`, true}, {"POST", "create", `{"id":""}`, false},
		{"POST", "create", `{"id":123}`, false}, {"GET", "detail", `{"order":{"id":"123"}}`, true},
		{"GET", "detail", `{"order":{}}`, false}, {"POST", "stop", `{"order":{"id":"123"}}`, true},
		{"POST", "stop", `{"order":[]}`, false}, {"POST", "stop_all", `{"orders":[]}`, true},
	} {
		response := &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"code":0,"data":` + tc.body + `}`))}
		normalizeChaseResponse(response, tc.method, "/api/v4/futures/usdt/autoorder/v1/chase/"+tc.leaf)
		_, err := io.ReadAll(response.Body)
		require.NoError(t, response.Body.Close())
		require.Equal(t, tc.ok, err == nil, tc.leaf+tc.body)
	}
}
func TestChaseNormalizationDoesNotChangeOtherProtocols(t *testing.T) {
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/api/v4/spot/orders/1", 200}, {"GET", "/api/v4/futures/usdt/trail_orders", 200},
		{"POST", "/api/v4/bot/portfolio/stop", 200}, {"GET", "/api/v4/futures/usdt/autoorder/v1/chase/list", 403},
	} {
		raw := `{"code":-1,"message":"preserved"}`
		body := &chaseTestBody{Reader: strings.NewReader(raw)}
		r := &http.Response{StatusCode: tc.status, Body: body}
		normalizeChaseResponse(r, tc.method, tc.path)
		require.Same(t, body, r.Body)
		require.Equal(t, 0, body.closed)
		b, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, raw, string(b))
		require.NoError(t, r.Body.Close())
	}
}
func TestChaseBodyFailureIsClosedAndReported(t *testing.T) {
	for _, source := range []*chaseTestBody{
		{Reader: io.LimitReader(strings.NewReader(`{"orders":[]}`), 1)},
		{Reader: strings.NewReader(`{"orders":[]}`), closeErr: errors.New("close")},
		{Reader: &boundedAPIResponse{ReadCloser: io.NopCloser(strings.NewReader(strings.Repeat("x", 20))), remaining: 10}},
	} {
		r := &http.Response{StatusCode: 200, Header: make(http.Header), Body: source}
		normalizeChaseResponse(r, "GET", "/api/v4/futures/usdt/autoorder/v1/chase/list")
		_, err := io.ReadAll(r.Body)
		require.Error(t, err)
		require.Equal(t, 1, source.closed)
		require.NoError(t, r.Body.Close())
	}
	r := &http.Response{StatusCode: 200}
	normalizeChaseResponse(r, "GET", "/api/v4/futures/usdt/autoorder/v1/chase/list")
	_, err := io.ReadAll(r.Body)
	require.Error(t, err)
	require.NoError(t, r.Body.Close())
}

// Ensure partial recovery facts survive a known business failure unchanged.
func TestChaseBusinessDetailsArePreserved(t *testing.T) {
	r := &http.Response{StatusCode: 200, Header: http.Header{"X-Gate-Trace-Id": []string{"trace"}}, Body: io.NopCloser(strings.NewReader(`{"code":-7,"label":"REJECTED","message":"original","data":{"id":"123"}}`))}
	normalizeChaseResponse(r, "POST", "/api/v4/futures/usdt/autoorder/v1/chase/create")
	_, err := io.ReadAll(r.Body)
	require.NoError(t, r.Body.Close())
	var detail *output.GateError
	require.True(t, errors.As(err, &detail))
	require.Equal(t, "original", detail.Message)
	require.Equal(t, "REJECTED", detail.Label)
	require.Equal(t, -7, *detail.BusinessCode)
	require.Equal(t, "trace", detail.TraceID)
	require.JSONEq(t, `{"id":"123"}`, string(detail.Data))
	require.False(t, detail.Retryable)
	b, marshalErr := json.Marshal(detail)
	require.NoError(t, marshalErr)
	require.Contains(t, string(b), "original")
}
