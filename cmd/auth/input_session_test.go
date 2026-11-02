package auth

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// TestLoginInputHandoff 同一登录会话跨自动/手动阶段共享换行状态，不预读授权码。
func TestLoginInputHandoff(t *testing.T) {
	for _, newline := range []string{"\r\n", "\n", "\r"} {
		t.Run(strings.ReplaceAll(strings.ReplaceAll(newline, "\r", "CR"), "\n", "LF"), func(t *testing.T) {
			input := &inputSession{in: strings.NewReader(newline + "synthetic-code" + newline)}
			if !input.watchEnter(context.Background()) {
				t.Fatal("Enter not recognized")
			}
			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())
			code, err := readAuthCode(cmd, input)
			if err != nil || code != "synthetic-code" {
				t.Fatalf("manual code=%q err=%v", code, err)
			}
		})
	}
}

// TestLoginInputSplitCRLF 模拟CR先到、LF与code后到；首阶段不能等待尚未到达的LF。
func TestLoginInputSplitCRLF(t *testing.T) {
	r, w := io.Pipe()
	t.Cleanup(func() { _ = r.Close(); _ = w.Close() })
	input := &inputSession{in: r}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	done := make(chan bool, 1)
	go func() { done <- input.watchEnter(ctx) }()
	if _, err := io.WriteString(w, "\r"); err != nil {
		t.Fatal(err)
	}
	if !<-done {
		t.Fatal("CR waited for LF")
	}
	writeDone := make(chan error, 1)
	go func() { _, err := io.WriteString(w, "\nsynthetic-code\r\n"); writeDone <- err }()
	line, err := input.readLine(ctx, 4096)
	if err != nil || line != "synthetic-code" {
		t.Errorf("manual code=%q err=%v", line, err)
	}
	// 行读取不预读尾部LF，关闭测试管道回收仍在发送该字节的写协程。
	if err := r.Close(); err != nil {
		t.Error(err)
	}
	<-writeDone
}
