//go:build agentic_e2e

package agentic_test

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/config"
)

// TestCFDAdditionalSDKQueries 覆盖CLI未新增叶子的两个查询及第五个int64路径ID，直接使用公共SDK。
func TestCFDAdditionalSDKQueries(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/api/v4/tradfi/orders/log/9223372036854775807" && r.URL.Path != "/api/v4/tradfi/symbols/commissions" {
			t.Errorf("unexpected SDK path=%s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"code":0,"data":{}}`)); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	c, err := client.New(&config.Config{BaseURL: server.URL, APIKey: "synthetic", APISecret: "synthetic"}, "tradfi/sdk")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.TradFiAPI.QueryOrderLog(t.Context(), 9223372036854775807); err != nil {
		t.Fatal(err)
	}
	if _, _, err := c.TradFiAPI.QuerySymbolCommissions(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatal("missing SDK query")
	}
}
