package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gate/gate-cli/internal/config"
	"github.com/gate/gate-cli/internal/output"
)

// TestFuturesUnknownModeStopsDependentCalls 模式未知时不得查询猜测的仓位端点或提交杠杆写操作。
func TestFuturesUnknownModeStopsDependentCalls(t *testing.T) {
	for _, status := range []int{401, 403, 503, 200} {
		for _, write := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/write=%t", status, write), func(t *testing.T) {
				var downstream atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					if r.URL.Path == "/api/v4/futures/usdt/accounts" {
						w.Header().Set("X-Gate-Trace-Id", "synthetic-mode-trace")
						w.WriteHeader(status)
						body := `{"label":"MODE_UNAVAILABLE","message":"cannot determine position mode"}`
						if status == 200 {
							body = `not-json`
						}
						if _, err := w.Write([]byte(body)); err != nil {
							t.Error(err)
						}
						return
					}
					downstream.Add(1)
					if _, err := w.Write([]byte(`{"contract":"SYNTH_USDT","size":"0"}`)); err != nil {
						t.Error(err)
					}
				}))
				defer server.Close()
				c, err := New(&config.Config{BaseURL: server.URL, APIKey: "synthetic", APISecret: "synthetic"}, "futures/position")
				if err != nil {
					t.Fatal(err)
				}
				var callErr error
				if write {
					_, _, callErr = c.UpdateFuturesPositionLeverage("usdt", "SYNTH_USDT", "2", nil)
				} else {
					_, _, callErr = c.GetFuturesPosition("usdt", "SYNTH_USDT")
				}
				if callErr == nil || downstream.Load() != 0 {
					t.Fatalf("unknown mode continued: err=%v downstream=%d", callErr, downstream.Load())
				}
				if status != 200 {
					// 多步骤调用的统一错误入口必须保留真正失败的accounts请求，而非覆盖成仓位路径。
					failure := CommandError(callErr, nil, "POST", "/synthetic-later-operation", "")
					var detail *output.GateError
					if !errors.As(failure, &detail) || detail.Status != status || detail.TraceID != "synthetic-mode-trace" || detail.Label != "MODE_UNAVAILABLE" || detail.Request.URL != "/api/v4/futures/usdt/accounts" {
						t.Fatalf("mode error context lost: %+v", detail)
					}
				}
			})
		}
	}
}

// TestFuturesModeCancellationStopsWrite 确认超时/取消也停止后续写，而非只处理HTTP拒绝。
func TestFuturesModeCancellationStopsWrite(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		t.Run(fmt.Sprint(cancelled), func(t *testing.T) {
			var downstream atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/v4/futures/usdt/accounts" {
					<-r.Context().Done()
					return
				}
				downstream.Add(1)
			}))
			defer server.Close()
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Millisecond)
			defer cancel()
			if cancelled {
				cancel()
			}
			c, err := NewWithContext(ctx, &config.Config{BaseURL: server.URL, APIKey: "synthetic", APISecret: "synthetic"}, "futures/position")
			if err != nil {
				t.Fatal(err)
			}
			_, _, err = c.UpdateFuturesPositionLeverage("usdt", "SYNTH_USDT", "2", nil)
			if err == nil || downstream.Load() != 0 {
				t.Fatalf("cancelled mode query continued: err=%v writes=%d", err, downstream.Load())
			}
		})
	}
}

// TestFuturesModeFailureIsNotCached 临时失败不能污染模式缓存，后续显式调用可重新取得真实模式。
func TestFuturesModeFailureIsNotCached(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if calls.Add(1) == 1 {
			w.WriteHeader(503)
			if _, err := w.Write([]byte(`{"label":"MODE_UNAVAILABLE","message":"synthetic"}`)); err != nil {
				t.Error(err)
			}
			return
		}
		if _, err := w.Write([]byte(`{"in_dual_mode":true}`)); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	c, err := New(&config.Config{BaseURL: server.URL, APIKey: "synthetic", APISecret: "synthetic"}, "futures/position")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.IsDualMode("usdt"); err == nil {
		t.Fatal("mode query failure hidden")
	}
	for i := 0; i < 2; i++ {
		dual, err := c.IsDualMode("usdt")
		if err != nil || !dual {
			t.Fatalf("failed query poisoned cache: dual=%t err=%v", dual, err)
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("expected failed query and one successful query: %d", calls.Load())
	}
}

// TestFuturesModeConcurrentCache 固定8个调用者共享客户端，验证缓存读写无竞态，不引入后台任务。
func TestFuturesModeConcurrentCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"in_dual_mode":true}`)); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	c, err := NewWithContext(t.Context(), &config.Config{BaseURL: server.URL, APIKey: "synthetic", APISecret: "synthetic"}, "futures/position")
	if err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 8)
	for i := 0; i < 8; i++ {
		go func() {
			dual, err := c.IsDualMode("usdt")
			if err == nil && !dual {
				err = fmt.Errorf("confirmed dual mode lost")
			}
			results <- err
		}()
	}
	for i := 0; i < 8; i++ {
		if err := <-results; err != nil {
			t.Error(err)
		}
	}
}
