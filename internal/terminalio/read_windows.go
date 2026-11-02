//go:build windows

package terminalio

import (
	"context"
	"errors"
	"io"
	"os"
	"runtime"
	"time"

	"golang.org/x/sys/windows"
)

var cancelSynchronousIO = windows.NewLazySystemDLL("kernel32.dll").NewProc("CancelSynchronousIo")

// Windows控制台同步Read绑定OS线程，取消时只中断该线程的IO，不关闭共享stdin句柄。
func WithReader(ctx context.Context, file *os.File, consume func(io.Reader) error) (resultErr error) {
	// 不启动无法取消的同步读；旧Windows缺少该API时先明确失败。
	if err := cancelSynchronousIO.Find(); err != nil {
		return err
	}
	type result struct {
		err error
	}
	done := make(chan result, 1)
	ready := make(chan windows.Handle, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		var thread windows.Handle
		err := windows.DuplicateHandle(windows.CurrentProcess(), windows.CurrentThread(), windows.CurrentProcess(), &thread, 0, false, windows.DUPLICATE_SAME_ACCESS)
		ready <- thread
		if err != nil {
			done <- result{err}
			return
		}
		err = consume(file)
		done <- result{err}
	}()
	thread := <-ready
	if thread != 0 {
		defer func() { resultErr = errors.Join(resultErr, windows.CloseHandle(thread)) }()
	}
	select {
	case got := <-done:
		return got.err
	case <-ctx.Done():
		// Read可能尚未进入内核，重复取消直到线程退出，防止取消先于Read的竞争。
		ticker := time.NewTicker(25 * time.Millisecond)
		defer ticker.Stop()
		var cancelErr error
		for {
			if thread != 0 {
				cancelled, _, callErr := cancelSynchronousIO.Call(uintptr(thread))
				// ERROR_NOT_FOUND说明Read尚未提交或已结束，继续回收屏障；其他错误保留并报告。
				if cancelled == 0 && !errors.Is(callErr, windows.ERROR_NOT_FOUND) && cancelErr == nil {
					cancelErr = callErr
				}
			}
			select {
			case <-done:
				return errors.Join(ctx.Err(), cancelErr)
			case <-ticker.C:
			}
		}
	}
}
