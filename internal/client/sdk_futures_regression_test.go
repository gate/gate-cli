package client

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gate/gate-cli/internal/config"
	gateapi "github.com/gate/gateapi-go/v7"
)

// TestSDK161FuturesPositionCompatibility 升级SDK和共享传输后，单/双向仓位仍使用各自协议及同一模式快照。
func TestSDK161FuturesPositionCompatibility(t *testing.T) {
	for _, dual := range []bool{false, true} {
		t.Run(fmt.Sprint(dual), func(t *testing.T) {
			var accounts atomic.Int32
			var fail atomic.Bool
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if strings.HasSuffix(r.URL.Path, "/accounts") {
					accounts.Add(1)
					if _, err := fmt.Fprintf(w, `{"in_dual_mode":%t}`, dual); err != nil {
						t.Error(err)
					}
					return
				}
				if strings.Contains(r.URL.Path, "dual_comp") != dual {
					t.Error("wrong position mode endpoint")
				}
				if fail.Load() {
					w.WriteHeader(503)
					if _, err := w.Write([]byte(`{"label":"UNAVAILABLE","message":"synthetic"}`)); err != nil {
						t.Error(err)
					}
					return
				}
				result := `{"contract":"SYNTH_USDT","size":"1"}`
				if dual {
					result = "[" + result + "]"
				}
				if _, err := w.Write([]byte(result)); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			c, err := New(&config.Config{BaseURL: server.URL, APIKey: "synthetic", APISecret: "synthetic"}, "futures/position/list")
			if err != nil {
				t.Fatal(err)
			}
			positions, _, err := c.GetFuturesPosition("usdt", "SYNTH_USDT")
			if err != nil || len(positions) != 1 || positions[0].Contract != "SYNTH_USDT" {
				t.Fatal("position decode", err)
			}
			positions, _, err = c.UpdateFuturesPositionLeverage("usdt", "SYNTH_USDT", "2", &gateapi.UpdatePositionLeverageOpts{})
			if err != nil || len(positions) != 1 {
				t.Fatal("leverage decode", err)
			}
			if accounts.Load() != 1 {
				t.Fatal("repeated account snapshot")
			}
			fail.Store(true)
			if _, _, err := c.GetFuturesPosition("usdt", "SYNTH_USDT"); err == nil {
				t.Fatal("query failed silently")
			}
			if _, _, err := c.UpdateFuturesPositionLeverage("usdt", "SYNTH_USDT", "2", nil); err == nil {
				t.Fatal("write failed silently")
			}
		})
	}
}
