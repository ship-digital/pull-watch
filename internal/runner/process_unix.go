//go:build !windows

package runner

import (
	"os/exec"
	"syscall"
)

func setProcessGroup(cmd *exec.Cmd) {
	// Set process group so child processes get signals
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}

// prepareCommand returns the command as-is on Unix systems
func prepareCommand(command []string) (string, []string) {
	if len(command) == 0 {
		return "", nil
	}
	return command[0], command[1:]
}

func terminateProcess(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	// Send SIGTERM to process group instead of just the process
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
}

func killProcess(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	// Send SIGKILL to process group
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
