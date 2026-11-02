package auth

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
)

// inputSession 由一次登录独占；自动读者退出后才交给手动读者，不需全局状态或锁。
// afterCR只跨阶段记录一个待跳过的LF，不预读、不等待LF，也不保存授权码。
type inputSession struct {
	in      io.Reader
	afterCR bool
}

// readLine 将File交给平台取消机制；其他可关闭reader在取消后回收读协程。
func (input *inputSession) readLine(ctx context.Context, max int) (string, error) {
	in := input.in
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if in == nil {
		return "", io.EOF
	}
	if file, ok := in.(*os.File); ok {
		return input.readFileLine(ctx, file, max)
	}
	switch in.(type) {
	case *strings.Reader, *bytes.Reader, *bytes.Buffer:
		return input.scanLine(ctx, in, max)
	}
	closer, ok := in.(io.ReadCloser)
	if !ok {
		return "", fmt.Errorf("input does not support cancellation")
	}
	type result struct {
		line string
		err  error
	}
	done := make(chan result, 1)
	go func() { line, err := input.scanLine(ctx, in, max); done <- result{line, err} }()
	select {
	case got := <-done:
		return got.line, got.err
	case <-ctx.Done():
		// 此reader属于本次登录输入；关闭它使Read退出，并回收协程后才返回。
		closeErr := closer.Close()
		<-done
		if closeErr != nil {
			return "", fmt.Errorf("cancel input: %w", closeErr)
		}
		return "", ctx.Err()
	}
}

// scanLine 处理raw终端退格/Ctrl-C，授权码最大4096字节，始终不回显。
func (input *inputSession) scanLine(ctx context.Context, in io.Reader, max int) (string, error) {
	line := make([]byte, 0, 128)
	var one [1]byte
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		n, err := in.Read(one[:])
		if n > 0 {
			// 仅吞掉上一次CR紧邻的LF；单CR后的首个code字节必须正常处理。
			pairedLF := input.afterCR && one[0] == '\n'
			input.afterCR = false
			if pairedLF {
				if err != nil {
					return "", err
				}
				continue
			}
			switch one[0] {
			case '\r':
				input.afterCR = true
				return string(line), nil
			case '\n':
				return string(line), nil
			case 3:
				return "", context.Canceled
			case 4:
				return "", io.EOF
			case 8, 127:
				if len(line) > 0 {
					line = line[:len(line)-1]
				}
			default:
				if len(line) >= max {
					return "", fmt.Errorf("authorization input exceeds %d bytes", max)
				}
				line = append(line, one[0])
			}
		}
		if err != nil {
			return "", err
		}
	}
}
