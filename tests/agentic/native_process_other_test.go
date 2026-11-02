//go:build agentic_e2e && !(darwin || linux)

package agentic_test

import (
	"fmt"
	"os/exec"
)

// Docker remains the default on platforms without this native process owner.
func nativeProcessGroup(_ *exec.Cmd) error {
	return fmt.Errorf("native fixture requires Darwin or Linux; use the existing Docker fixture")
}
func signalNativeGroup(_ *exec.Cmd, _ bool) error {
	return fmt.Errorf("native fixture is unavailable on this platform")
}
