package auth

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

type failedOutput struct{}

func (failedOutput) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

// TestAuthIOFailures 关闭输出时不能继续浏览器登录；取消/控制字符/不支持的输入有界失败。
func TestAuthIOFailures(t *testing.T) {
	if err := announceAuthorizeURL(failedOutput{}, io.Discard, "https://example.invalid", "auto", false, false); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal("announcement ignored stdout failure", err)
	}
	if err := announceAuthorizeURL(io.Discard, failedOutput{}, "https://example.invalid", "manual", true, false); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal("announcement ignored stderr failure", err)
	}
	if _, err := (&inputSession{}).readLine(context.Background(), 100); !errors.Is(err, io.EOF) {
		t.Fatal(err)
	}
	if _, err := (&inputSession{in: io.LimitReader(strings.NewReader("x"), 1)}).readLine(context.Background(), 100); err == nil {
		t.Fatal("unsupported reader blocked")
	}
	for _, value := range []string{"\x03", "\x04"} {
		if _, err := (&inputSession{in: strings.NewReader(value)}).readLine(context.Background(), 100); err == nil {
			t.Fatal("control character accepted")
		}
	}
	for _, input := range []io.Reader{bytes.NewBufferString("ab\bX\n"), bytes.NewReader([]byte("ab\x7fX\n"))} {
		line, err := (&inputSession{in: input}).readLine(context.Background(), 100)
		if err != nil || line != "aX" {
			t.Fatal("terminal edit failed", line, err)
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := (&inputSession{in: strings.NewReader("code\n")}).readLine(cancelled, 100); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := openBrowser(cancelled, "file:///synthetic"); err == nil {
		t.Fatal("unsafe browser scheme accepted")
	}
	if err := openBrowser(cancelled, "https://example.invalid"); err == nil {
		t.Fatal("cancelled browser launcher succeeded")
	}
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.SetIn(strings.NewReader(" code \n"))
	if line, err := readAuthCode(cmd, &inputSession{in: cmd.InOrStdin()}); err != nil || line != "code" {
		t.Fatal("manual whitespace handling", line, err)
	}
}
