//go:build windows

package runner

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

// prepareCommand wraps script files with appropriate interpreters on Windows
func prepareCommand(command []string) (string, []string) {
	if len(command) == 0 {
		return "", nil
	}

	executable := command[0]
	args := command[1:]

	// Get file extension
	ext := strings.ToLower(filepath.Ext(executable))

	switch ext {
	case ".ps1":
		// PowerShell script
		newArgs := []string{"-File", executable}
		newArgs = append(newArgs, args...)
		return "powershell.exe", newArgs
	case ".bat", ".cmd":
		// Batch script
		newArgs := []string{"/c", executable}
		newArgs = append(newArgs, args...)
		return "cmd.exe", newArgs
	default:
		// Regular executable
		return executable, args
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
