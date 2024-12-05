package executor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ship-digital/pull-watch/internal/config"
	"github.com/ship-digital/pull-watch/internal/logger"
)

// CommandExecutor defines an interface for executing commands
type CommandExecutor interface {
	ExecuteCommand(ctx context.Context, name string, args ...string) (string, error)
	GetConfig() *config.Config
}

// DefaultExecutor is the real command executor
type DefaultExecutor struct {
	cfg *config.Config
}

func New(cfg *config.Config) *DefaultExecutor {
	return &DefaultExecutor{cfg: cfg}
}

func (e *DefaultExecutor) ExecuteCommand(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = e.cfg.GitDir

	if e.cfg.Verbose {
		e.cfg.Logger.MultiColor(
			logger.InfoSegment("Executing command: "),
			logger.HighlightSegment(fmt.Sprintf("%s %s", name, strings.Join(args, " "))),
		)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("command failed: %w\nstderr: %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

func (e *DefaultExecutor) GetConfig() *config.Config {
	return e.cfg
}
