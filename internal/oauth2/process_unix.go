//go:build !windows

package oauth2

import (
	"errors"
	"os"
	"syscall"
)

// processAlive 只有明确不存在才允许恢复，PID复用或无权限均失败关闭。
func processAlive(pid int) (alive bool) {
	process, err := os.FindProcess(pid)
	if err != nil {
		return !errors.Is(err, os.ErrProcessDone)
	}
	defer func() {
		if err := process.Release(); err != nil {
			alive = true
		}
	}() // 释放失败同样保守拒绝接管。
	err = process.Signal(syscall.Signal(0))
	// 只有明确不存在才允许恢复；无权限、平台不支持及PID复用均失败关闭。
	return err == nil || (!errors.Is(err, syscall.ESRCH) && !errors.Is(err, os.ErrProcessDone))
}
