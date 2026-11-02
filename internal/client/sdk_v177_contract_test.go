package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gateapi "github.com/gate/gateapi-go/v7"
)

// Call the published SDK directly so a CLI transport workaround cannot conceal
// a regression in the authoritative string-ID/time response decoder.
func TestSDK177TrailQuotedFields(t *testing.T) {
	const order = `{"id":"9007199254740993","user_id":"1001","user":"1002","suborder_id":"9223372036854775807","created_at":"1793613819","activated_at":"0","finished_at":"0","create_time":"1793613819","active_time":"0","finish_time":"0","updated_at":"1793613820","contract":"SYNTH_USDT","amount":"1","unknown":true}`
	for _, operation := range []string{"detail", "list"} {
		t.Run(operation, func(t *testing.T) {
			peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || r.URL.Path != "/api/v4/futures/usdt/autoorder/v1/trail/"+operation {
					t.Errorf("unexpected route: %s %s", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				data := `{"order":` + order + `}`
				if operation == "list" {
					data = `{"orders":[` + order + `]}`
				}
				_, _ = fmt.Fprint(w, `{"code":0,"message":"ok","data":`+data+`,"timestamp":1793582374966743}`)
			}))
			t.Cleanup(peer.Close)
			cfg := gateapi.NewConfiguration()
			cfg.BasePath = peer.URL + "/api/v4"
			cfg.HTTPClient = &http.Client{Timeout: 2 * time.Second}
			api := gateapi.NewAPIClient(cfg).FuturesApi
			var result any
			var err error
			if operation == "list" {
				result, _, err = api.GetTrailOrders(t.Context(), "usdt", nil)
			} else {
				result, _, err = api.GetTrailOrderDetail(t.Context(), "usdt", 9007199254740993)
			}
			if err != nil {
				t.Fatal(err)
			}
			raw, err := json.Marshal(result)
			if err != nil {
				t.Fatal(err)
			}
			var envelope struct {
				Data struct {
					Order  map[string]any   `json:"order"`
					Orders []map[string]any `json:"orders"`
				} `json:"data"`
			}
			if err := json.Unmarshal(raw, &envelope); err != nil {
				t.Fatal(err)
			}
			got := envelope.Data.Order
			if operation == "list" {
				if len(envelope.Data.Orders) != 1 {
					t.Fatalf("orders lost: %s", raw)
				}
				got = envelope.Data.Orders[0]
			}
			var expected map[string]any
			if err := json.Unmarshal([]byte(order), &expected); err != nil {
				t.Fatal(err)
			}
			for _, field := range []string{"id", "user_id", "user", "suborder_id", "created_at", "activated_at", "finished_at", "create_time", "active_time", "finish_time", "updated_at"} {
				if got[field] != expected[field] {
					t.Errorf("%s=%#v, want exact string %#v", field, got[field], expected[field])
				}
			}
		})
	}
}
