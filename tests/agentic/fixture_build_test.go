//go:build agentic_e2e

package agentic_test

import (
	"os/exec"
	"runtime"
	"time"
)

// The fixture build owns its compiler descendants and their output pipes. Match
// the existing issuer lifecycle so cancellation cannot leave a linker running.
func runFixtureBuild(cmd *exec.Cmd) ([]byte, error) {
	cmd.WaitDelay = 2 * time.Second
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		if err := nativeProcessGroup(cmd); err != nil {
			return nil, err
		}
		cmd.Cancel = func() error { return signalNativeGroup(cmd, true) }
	}
	var log nativeDiagnostics
	cmd.Stdout, cmd.Stderr = &log, &log
	err := cmd.Run()
	return []byte(log.String()), err
}
