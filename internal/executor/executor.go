package executor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CommandExecutor defines an interface for executing commands
type CommandExecutor interface {
	ExecuteCommand(ctx context.Context, name string, args ...string) (string, error)
}

// DefaultExecutor is the real command executor
type DefaultExecutor struct {
	Dir string
}

func (e *DefaultExecutor) ExecuteCommand(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = e.Dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("command failed: %v\nstderr: %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}
