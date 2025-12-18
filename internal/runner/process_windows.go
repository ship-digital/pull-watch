//go:build windows

package runner

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

var (
	cachedPowerShell      string
	cachedExecutionPolicy string
	cacheMutex            sync.Mutex
	cacheInitialized      bool
)

func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

// findPowerShell finds the best available PowerShell executable
// Returns pwsh.exe if available (PowerShell 7+), otherwise powershell.exe (5.1)
func findPowerShell() string {
	// Try pwsh.exe first (PowerShell 7+) - modern, faster, better
	if path, err := exec.LookPath("pwsh"); err == nil {
		return path
	}

	// Fall back to powershell.exe (Windows PowerShell 5.1) - always available on Windows
	if path, err := exec.LookPath("powershell"); err == nil {
		return path
	}

	// Ultimate fallback - use the name and let the OS search PATH
	return "powershell.exe"
}

// getExecutionPolicy queries the current effective PowerShell execution policy
// Returns the policy name (e.g., "RemoteSigned", "Bypass", etc.)
func getExecutionPolicy(psExe string) string {
	// Query the effective execution policy
	cmd := exec.Command(psExe, "-NoProfile", "-Command", "Get-ExecutionPolicy")
	output, err := cmd.Output()
	if err != nil {
		// If we can't determine the policy, use Bypass as a safe fallback
		// This ensures scripts can run even if detection fails
		return "Bypass"
	}

	policy := strings.TrimSpace(string(output))
	if policy == "" {
		return "Bypass"
	}

	return policy
}

// initializePowerShellCache initializes the cached PowerShell path and execution policy
// This is called once to avoid repeated lookups
func initializePowerShellCache() {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	if cacheInitialized {
		return
	}

	cachedPowerShell = findPowerShell()
	cachedExecutionPolicy = getExecutionPolicy(cachedPowerShell)
	cacheInitialized = true
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
		// Initialize cache on first use
		initializePowerShellCache()

		// Use -ExecutionPolicy with the same policy as the current system
		// This respects the user's configured execution policy
		// -NoProfile skips loading profile scripts for cleaner execution
		// -File specifies the script file to execute
		newArgs := []string{
			"-ExecutionPolicy", cachedExecutionPolicy,
			"-NoProfile",
			"-File", executable,
		}
		newArgs = append(newArgs, args...)
		return cachedPowerShell, newArgs
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
