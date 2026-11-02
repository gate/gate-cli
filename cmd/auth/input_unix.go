//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package auth

import (
	"context"
	"io"
	"os"

	"github.com/gate/gate-cli/internal/terminalio"
)

// readinessReader 由当前协程轮询可读事件；取消后不关闭共享stdin、不留下阻塞读协程。
type readinessReader struct {
	ctx  context.Context
	file *os.File
}

func (r readinessReader) Read(p []byte) (int, error) {
	// Preserve the original private wrapper and scanner contract.
	return terminalio.Read(r.ctx, r.file, p)
}

func (input *inputSession) readFileLine(ctx context.Context, file *os.File, max int) (string, error) {
	return input.scanLine(ctx, io.Reader(readinessReader{ctx, file}), max)
}
