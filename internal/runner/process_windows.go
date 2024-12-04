//go:build windows

package runner

import (
	"fmt"
	"os/exec"
	"syscall"
)

func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

// Windows specific process termination
func terminateProcess(cmd *exec.Cmd) error {
	// On Windows, we'll use taskkill /T to terminate the process tree
	kill := exec.Command("taskkill", "/T", "/PID", fmt.Sprint(cmd.Process.Pid))
	return kill.Run()
}

func killProcess(cmd *exec.Cmd) error {
	// Force kill with /F flag
	kill := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprint(cmd.Process.Pid))
	return kill.Run()
}
