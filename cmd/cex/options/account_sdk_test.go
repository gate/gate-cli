package options

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	gateapi "github.com/gate/gateapi-go/v7"
)

// Exercise the generated SDK HTTP decoder, rather than a hand-written mirror DTO.
// Only the upstream response is synthetic; no account or trading service is used.
func TestOptionsAccountNotionalLimitSDK(t *testing.T) {
	cases := []struct {
		name, field, want string
		status            int
		wantError         bool
	}{
		{"reported_string", `,"position_notional_limit":"10000000"`, "10000000", 200, false},
		{"zero_string", `,"position_notional_limit":"0"`, "0", 200, false},
		{"wide_string", `,"position_notional_limit":"9223372036854775808"`, "9223372036854775808", 200, false},
		{"decimal_string", `,"position_notional_limit":"10000000.123456789012345678"`, "10000000.123456789012345678", 200, false},
		{"empty_string", `,"position_notional_limit":""`, "", 200, false},
		{"optional_absent", "", "", 200, false},
		{"integer_mismatch", `,"position_notional_limit":10000000`, "", 200, true},
		{"fraction_mismatch", `,"position_notional_limit":1.5`, "", 200, true},
		{"boolean_mismatch", `,"position_notional_limit":true`, "", 200, true},
		{"object_mismatch", `,"position_notional_limit":{}`, "", 200, true},
		{"bad_request", "", "", 400, true},
		{"server_error", "", "", 500, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var hits int32
			peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&hits, 1)
				if r.Method != http.MethodGet || r.URL.Path != "/api/v4/options/accounts" {
					t.Errorf("wrong request: %s %s", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				body := fmt.Sprintf(`{"user":666,"short_enabled":true,"orders_limit":10,"total":"1.123456789012345678"%s}`, tc.field)
				if tc.status != 200 {
					body = `{"label":"INVALID_PARAM_VALUE","message":"fixture rejection"}`
				}
				if _, err := io.WriteString(w, body); err != nil {
					t.Error(err)
				}
			}))
			defer peer.Close()
			cfg := gateapi.NewConfiguration()
			cfg.BasePath = peer.URL + "/api/v4"
			cfg.HTTPClient = &http.Client{Timeout: 2 * time.Second}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			account, response, err := gateapi.NewAPIClient(cfg).OptionsApi.ListOptionsAccount(ctx)
			if (err != nil) != tc.wantError || response == nil || response.StatusCode != tc.status {
				t.Fatalf("response=%v error=%v", response, err)
			}
			if atomic.LoadInt32(&hits) != 1 {
				t.Fatal("query was lost or retried")
			}
			if !tc.wantError && (account.PositionNotionalLimit != tc.want || account.User != 666 || !account.ShortEnabled || account.OrdersLimit != 10 || account.Total != "1.123456789012345678") {
				t.Fatalf("response fields changed: %+v", account)
			}
		})
	}
}
