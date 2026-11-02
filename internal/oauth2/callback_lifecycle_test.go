package oauth2

import (
	"context"
	"testing"
	"time"
)

// TestCallbackCloseBeforeServe 覆盖取消与Serve启动的竞争，Close之后不得重新占用端口。
func TestCallbackCloseBeforeServe(t *testing.T) {
	for i := 0; i < 50; i++ {
		s, err := NewCallbackServer()
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			defer close(done)
			_, _ = s.WaitForCallback(ctx) // 该竞争测试只验证退出；取消/先Close均是预期错误。
		}()
		s.Close()
		cancel()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("callback goroutine leaked")
		}
	}
}
