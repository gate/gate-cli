package client

import (
	"errors"
	"github.com/gate/gate-cli/internal/config"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"testing/iotest"
)

// Both generated SDK methods share InsuranceRecord; decimal lexemes must remain exact.
func TestInsuranceNumericBalanceSDK(t *testing.T) {
	for _, kind := range []string{"futures", "delivery"} {
		t.Run(kind, func(t *testing.T) {
			for _, tc := range []struct {
				name, balance, want string
				invalid             bool
			}{
				{"number", "307314616.15009", "307314616.15009", false}, {"negative", "-42542.804042324", "-42542.804042324", false},
				{"precision", "9007199254740993.123456789012345678", "9007199254740993.123456789012345678", false},
				{"string", `"1021.210000"`, "1021.210000", false}, {"zero", "0", "0", false}, {"exponent", "1.23e-18", "1.23e-18", false},
				{"boolean", "true", "", true}, {"null", "null", "", true}, {"object", "{}", "", true}, {"missing", "", "", true},
			} {
				t.Run(tc.name, func(t *testing.T) {
					raw := `[{"t":1793232000`
					if tc.balance != "" {
						raw += `,"b":` + tc.balance
					}
					raw += `}]`
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						if _, err := io.WriteString(w, raw); err != nil {
							t.Error(err)
						}
					}))
					t.Cleanup(server.Close)
					c, err := New(&config.Config{BaseURL: server.URL}, kind+"/market/insurance")
					if err != nil {
						t.Fatal(err)
					}
					var got string
					if kind == "futures" {
						rows, _, callErr := c.FuturesAPI.ListFuturesInsuranceLedger(t.Context(), "usdt", nil)
						err = callErr
						if len(rows) == 1 {
							got = rows[0].B
						}
					} else {
						rows, _, callErr := c.DeliveryAPI.ListDeliveryInsuranceLedger(t.Context(), "usdt", nil)
						err = callErr
						if len(rows) == 1 {
							got = rows[0].B
						}
					}
					if (err != nil) != tc.invalid {
						t.Errorf("invalid=%v error=%v", tc.invalid, err)
					}
					if !tc.invalid && got != tc.want {
						t.Errorf("balance=%q want=%q", got, tc.want)
					}
				})
			}
		})
	}
}

// Exercise SDK ReadAll, rather than only the reader helper, at the exact byte boundary.
func TestMarketResponseSizeSDK(t *testing.T) {
	for _, size := range []int{2*1024*1024 + 1, 8 * 1024 * 1024, 8*1024*1024 + 1} {
		t.Run(strconv.Itoa(size), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if _, err := io.WriteString(w, "[]"+strings.Repeat(" ", size-2)); err != nil && size <= 8*1024*1024 {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			c, err := New(&config.Config{BaseURL: server.URL}, "spot/market/currencies")
			if err != nil {
				t.Fatal(err)
			}
			_, _, err = c.SpotAPI.ListCurrencies(t.Context())
			if size <= 8*1024*1024 && err != nil {
				t.Fatalf("valid response rejected: %v", err)
			}
			if size > 8*1024*1024 && (err == nil || !strings.Contains(err.Error(), "response exceeds")) {
				t.Fatalf("expected size rejection, got %v", err)
			}
		})
	}
}

// Adapter failures must close the original body and must not alter unrelated APIs.
func TestInsuranceAdapterBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, raw string
		bad       bool
	}{
		{"empty-array", "[]", false}, {"top-null", "null", true}, {"object", "{}", true}, {"malformed", "[", true},
		{"null-record", "[null]", true}, {"array-balance", `[{"b":[]}]`, true}, {"empty-balance", `[{"b":""}]`, true},
		{"trailing-document", `[{"b":1}] []`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := &chaseTestBody{Reader: strings.NewReader(tc.raw)}
			response := &http.Response{StatusCode: 200, Header: make(http.Header), Body: source}
			normalizeInsuranceResponse(response, "GET", "/api/v4/futures/usdt/insurance")
			_, err := io.ReadAll(response.Body)
			if (err != nil) != tc.bad {
				t.Errorf("bad=%v error=%v", tc.bad, err)
			}
			if err := response.Body.Close(); err != nil {
				t.Fatal(err)
			}
			if source.closed != 1 {
				t.Errorf("source closes=%d want1", source.closed)
			}
		})
	}
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/api/v4/spot/currencies", 200}, {"POST", "/api/v4/futures/usdt/insurance", 200},
		{"GET", "/api/v4/futures/usdt/insurance/extra", 200}, {"GET", "/api/v4/futures/usdt/insurance", 400},
	} {
		source := &chaseTestBody{Reader: strings.NewReader(`{"label":"preserve"}`)}
		response := &http.Response{StatusCode: tc.status, Body: source}
		normalizeInsuranceResponse(response, tc.method, tc.path)
		if response.Body != source || source.closed != 0 {
			t.Errorf("changed unrelated response: %+v", tc)
		}
		if err := response.Body.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGenericSDKDecodeCauseIsPreserved(t *testing.T) {
	const raw = `{"private_debug":"must-not-be-printed"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := io.WriteString(w, raw); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	c, err := New(&config.Config{BaseURL: server.URL}, "spot/market/currencies")
	if err != nil {
		t.Fatal(err)
	}
	_, response, err := c.SpotAPI.ListCurrencies(t.Context())
	if err == nil {
		t.Fatal("object decoded as currency array")
	}
	detail := ParseGateError(err, response, "GET", "/api/v4/spot/currencies", "")
	if !strings.Contains(detail.Message, "cannot unmarshal") || strings.Contains(detail.Message, "must-not-be-printed") || detail.Status != 200 || detail.Label != "INVALID_RESPONSE" {
		t.Errorf("diagnostic=%+v", detail)
	}
}

// I/O failure and nil bodies must stay failures, with no leaked response stream.
func TestInsuranceAdapterIOFailures(t *testing.T) {
	for _, source := range []*chaseTestBody{
		{Reader: iotest.ErrReader(errors.New("synthetic read failure"))},
		{Reader: strings.NewReader("[]"), closeErr: errors.New("synthetic close failure")},
	} {
		response := &http.Response{StatusCode: 200, Body: source}
		normalizeInsuranceResponse(response, "GET", "/api/v4/delivery/usdt/insurance")
		if _, err := io.ReadAll(response.Body); err == nil {
			t.Error("body failure appeared successful")
		}
		if err := response.Body.Close(); err != nil {
			t.Fatal(err)
		}
		if source.closed != 1 {
			t.Errorf("source closes=%d want1", source.closed)
		}
	}
	response := &http.Response{StatusCode: 200}
	normalizeInsuranceResponse(response, "GET", "/api/v4/futures/usdt/insurance")
	if _, err := io.ReadAll(response.Body); err == nil {
		t.Error("missing body appeared successful")
	}
	if err := response.Body.Close(); err != nil {
		t.Fatal(err)
	}
	normalizeInsuranceResponse(nil, "GET", "/api/v4/futures/usdt/insurance")
}
