package contracts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SiblingRepository supports both normal and .worktrees checkouts without
// adding a product setting or creating symlinks outside the test workspace.
func SiblingRepository(source, name, sentinel string) (string, error) {
	if name == "" || name == "." || name == ".." || filepath.Base(name) != name {
		return "", fmt.Errorf("invalid sibling repository name")
	}
	clean := filepath.Clean(sentinel)
	if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid repository sentinel")
	}
	workspace := filepath.Clean(filepath.Join(filepath.Dir(source), "../../.."))
	for _, candidate := range []string{filepath.Join(workspace, name), filepath.Join(workspace, ".worktrees", name)} {
		info, err := os.Stat(filepath.Join(candidate, clean))
		if err == nil && info.Mode().IsRegular() {
			return candidate, nil
		}
		if err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("inspect test dependency: %w", err)
		}
	}
	return "", fmt.Errorf("test dependency %s with %s not found under %s or its .worktrees directory", name, clean, workspace)
}
