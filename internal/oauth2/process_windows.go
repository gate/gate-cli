//go:build windows

package oauth2

import (
	"errors"

	"golang.org/x/sys/windows"
)

// processAlive Windows不支持Signal(0)；用查询句柄检查退出，禁止把所有死进程永久当作活跃。
func processAlive(pid int) (alive bool) {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return !errors.Is(err, windows.ERROR_INVALID_PARAMETER)
	}
	defer func() {
		// 无法确认句柄释放同样按活跃处理，不在异常环境接管授权。
		if err := windows.CloseHandle(handle); err != nil {
			alive = true
		}
	}()
	state, err := windows.WaitForSingleObject(handle, 0)
	return err != nil || state != windows.WAIT_OBJECT_0
}
