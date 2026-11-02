package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	gateapi "github.com/gate/gateapi-go/v7"
)

// Exercise the released SDK directly: application response workarounds must not
// conceal an incompatible envelope, a lost business code or a numeric timestamp.
func TestSDK178TrailNativeContracts(t *testing.T) {
	for _, operation := range []string{"update", "stop", "change_log"} {
		for _, mode := range []string{"normal", "empty", "business", "malformed", "400", "503", "numeric-time"} {
			if mode == "numeric-time" && operation != "change_log" {
				continue
			}
			t.Run(operation+"/"+mode, func(t *testing.T) {
				var calls atomic.Int32
				status := http.StatusOK
				if mode == "400" {
					status = http.StatusBadRequest
				}
				if mode == "503" {
					status = http.StatusServiceUnavailable
				}
				peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					wantMethod := http.MethodPost
					if operation == "change_log" {
						wantMethod = http.MethodGet
					}
					if r.Method != wantMethod || r.URL.Path != "/api/v4/futures/usdt/autoorder/v1/trail/"+operation {
						t.Errorf("wrong native route: %s %s", r.Method, r.URL.Path)
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(status)
					if status != http.StatusOK {
						_, _ = fmt.Fprint(w, `{"label":"FIXTURE_REJECTED","message":"rejected"}`)
						return
					}
					if mode == "malformed" {
						_, _ = fmt.Fprint(w, `{"data":`)
						return
					}
					data := `{"order":{"id":"9007199254740993","amount":"0.01","updated_at":"1793627143","status":"open"}}`
					if operation == "stop" {
						data = strings.Replace(data, `"open"`, `"finished"`, 1)
					}
					if operation == "change_log" {
						data = `{"change_log":[{"updated_at":"1793627143","amount":"0.01","price_type":1}]}`
					}
					if mode == "numeric-time" {
						data = strings.Replace(data, `"1793627143"`, `1793627143`, 1)
					}
					code := 0
					if mode == "empty" || mode == "business" {
						data = `{}`
					}
					if mode == "business" {
						code = -1
					}
					_, _ = fmt.Fprintf(w, `{"code":%d,"message":"fixture","data":%s,"timestamp":1793627143000,"future_field":true}`, code, data)
				}))
				t.Cleanup(peer.Close)
				cfg := gateapi.NewConfiguration()
				cfg.BasePath = peer.URL + "/api/v4"
				cfg.HTTPClient = &http.Client{Timeout: 2 * time.Second}
				api := gateapi.NewAPIClient(cfg).FuturesApi
				var value any
				var response *http.Response
				var err error
				switch operation {
				case "stop":
					value, response, err = api.StopTrailOrder(t.Context(), "usdt", gateapi.StopTrailOrder{Id: 9007199254740993})
				case "update":
					value, response, err = api.UpdateTrailOrder(t.Context(), "usdt", gateapi.UpdateTrailOrder{Id: 9007199254740993, ActivationPrice: "39075.6"})
				case "change_log":
					value, response, err = api.GetTrailOrderChangeLog(t.Context(), "usdt", 9007199254740993, nil)
				}
				if calls.Load() != 1 || response == nil || response.StatusCode != status {
					t.Fatalf("transport changed: calls=%d response=%v", calls.Load(), response)
				}
				failure := status != 200 || mode == "malformed" || mode == "numeric-time"
				if (err != nil) != failure {
					t.Fatalf("decode error=%v want failure=%v", err, failure)
				}
				if failure {
					return
				}
				raw, err := json.Marshal(value)
				if err != nil {
					t.Fatal(err)
				}
				var envelope struct {
					Code      int    `json:"code"`
					Message   string `json:"message"`
					Timestamp int64  `json:"timestamp"`
					Data      struct {
						Order struct {
							ID        string `json:"id"`
							UpdatedAt string `json:"updated_at"`
							Status    string `json:"status"`
						} `json:"order"`
						ChangeLog []struct {
							UpdatedAt string `json:"updated_at"`
							Amount    string `json:"amount"`
						} `json:"change_log"`
					} `json:"data"`
				}
				if err := json.Unmarshal(raw, &envelope); err != nil {
					t.Fatal(err)
				}
				wantCode := 0
				if mode == "business" {
					wantCode = -1
				}
				if envelope.Code != wantCode || envelope.Message != "fixture" || envelope.Timestamp != 1793627143000 {
					t.Fatalf("SDK discarded response envelope: %s", raw)
				}
				if mode == "normal" {
					if operation == "change_log" {
						if len(envelope.Data.ChangeLog) != 1 || envelope.Data.ChangeLog[0].UpdatedAt != "1793627143" || envelope.Data.ChangeLog[0].Amount != "0.01" {
							t.Fatalf("log content lost: %s", raw)
						}
					} else {
						wantStatus := "open"
						if operation == "stop" {
							wantStatus = "finished"
						}
						if envelope.Data.Order.ID != "9007199254740993" || envelope.Data.Order.UpdatedAt != "1793627143" || envelope.Data.Order.Status != wantStatus {
							t.Fatalf("order content lost: %s", raw)
						}
					}
				} else if envelope.Data.Order.ID != "" || len(envelope.Data.ChangeLog) != 0 {
					t.Fatalf("invented data: %s", raw)
				}
			})
		}
	}
}

// An already-cancelled request must not reach the order service or be retried.
func TestSDK178TrailCancellation(t *testing.T) {
	var calls atomic.Int32
	peer := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
	t.Cleanup(peer.Close)
	cfg := gateapi.NewConfiguration()
	cfg.BasePath = peer.URL + "/api/v4"
	cfg.HTTPClient = &http.Client{Timeout: time.Second}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, _, err := gateapi.NewAPIClient(cfg).FuturesApi.GetTrailOrderChangeLog(ctx, "usdt", 1, nil)
	if !errors.Is(err, context.Canceled) || calls.Load() != 0 {
		t.Fatalf("cancellation changed: calls=%d error=%v", calls.Load(), err)
	}
}
