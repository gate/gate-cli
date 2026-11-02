//go:build !windows

package config

import (
	"fmt"
	"os"
)

// replaceFile 保持同目录原子替换，随后由公共写者同步父目录。
func replaceFile(from, to string) error { return os.Rename(from, to) }

// SyncDirectory 同步POSIX目录项并传播持久化错误。
func SyncDirectory(dir string) error {
	file, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("open parent directory: %w", err)
	}
	syncErr := file.Sync()
	closeErr := file.Close()
	if syncErr != nil {
		return fmt.Errorf("sync parent directory: %w", syncErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close parent directory: %w", closeErr)
	}
	return nil
}
