//go:build agentic_e2e && (darwin || linux)

package agentic_test

import (
	"errors"
	"os/exec"
	"syscall"
)

// Each native fixture owns a group, including workers inheriting output pipes.
func nativeProcessGroup(command *exec.Cmd) error {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return nil
}

func signalNativeGroup(command *exec.Cmd, force bool) error {
	signal := syscall.SIGINT
	if force {
		signal = syscall.SIGKILL
	}
	err := syscall.Kill(-command.Process.Pid, signal)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}
