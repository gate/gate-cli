package auth

import (
	"context"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// TestStdinCancellationJoinsReader 保证取消后内层读协程也退出，而非仅返回外层select。
func TestStdinCancellationJoinsReader(t *testing.T) {
	before := runtime.NumGoroutine()
	for i := 0; i < 20; i++ {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Millisecond)
		(&inputSession{in: r}).watchEnter(ctx)
		cancel()
		if err := r.Close(); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
	}
	deadline := time.Now().Add(time.Second)
	for runtime.NumGoroutine() > before+2 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if runtime.NumGoroutine() > before+2 {
		t.Fatal("stdin readers leaked")
	}
}

// TestManualInputCancellation 对未结束的管道取消，不得卡住登录或把部分code发出去。
func TestManualInputCancellation(t *testing.T) {
	r, w := io.Pipe()
	defer func() {
		if err := r.Close(); err != nil {
			t.Error(err)
		}
	}()
	defer func() {
		if err := w.Close(); err != nil {
			t.Error(err)
		}
	}()
	ctx, cancel := context.WithCancel(context.Background())
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)
	cmd.SetIn(r)
	done := make(chan error, 1)
	go func() { _, err := readAuthCode(cmd, &inputSession{in: cmd.InOrStdin()}); done <- err }()
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancelled input succeeded")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("manual read ignored cancellation")
	}
}
func TestManualInputBound(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.SetIn(strings.NewReader(strings.Repeat("x", 4097) + "\n"))
	if _, err := readAuthCode(cmd, &inputSession{in: cmd.InOrStdin()}); err == nil {
		t.Fatal("oversized code accepted")
	}
}
