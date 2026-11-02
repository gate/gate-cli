//go:build windows

package config

import (
	"os"

	"golang.org/x/sys/windows"
)

// replaceFile 使用Windows写穿透替换；POSIX目录fsync不能直接套用到只读目录句柄。
func replaceFile(from, to string) error {
	source, err := windows.UTF16PtrFromString(from)
	if err != nil {
		return err
	}
	target, err := windows.UTF16PtrFromString(to)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(source, target, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}

// SyncDirectory Windows无等价的可移植目录fsync；原子写走上面的写穿透替换。
// 删除记录的掉电持久性由文件系统保证；若旧记录重现，摘要恢复逻辑仍失败关闭。
func SyncDirectory(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return &os.PathError{Op: "sync directory", Path: dir, Err: windows.ERROR_DIRECTORY}
	}
	return nil
}
