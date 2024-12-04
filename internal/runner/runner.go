package runner

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/ship-digital/pull-watch/internal/config"
	"github.com/ship-digital/pull-watch/internal/git"
)

var (
	prefix     = color.New(color.FgCyan).Sprint("[pull-watch] ")
	errorColor = color.New(color.FgRed).SprintFunc()
	infoColor  = color.New(color.FgGreen).SprintFunc()
)

// Custom logger that adds our prefix
type prefixLogger struct {
	*log.Logger
}

func newLogger() *prefixLogger {
	return &prefixLogger{
		Logger: log.New(os.Stderr, prefix, log.LstdFlags),
	}
}

func (l *prefixLogger) Error(format string, v ...interface{}) {
	l.Printf(errorColor("ERROR: "+format), v...)
}

func (l *prefixLogger) Info(format string, v ...interface{}) {
	l.Printf(infoColor(format), v...)
}

type ProcessManager struct {
	mu       sync.Mutex
	cmd      *exec.Cmd
	doneChan chan struct{}
	stopped  bool
	logger   *prefixLogger
}

func (pm *ProcessManager) Start(cfg *config.Config) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Make sure any previous process is fully cleaned up
	if pm.cmd != nil {
		if err := pm.forceStop(); err != nil && cfg.Verbose {
			pm.logger.Error("Failed to clean up previous process: %v", err)
		}
		pm.cmd = nil
	}

	pm.stopped = false
	pm.doneChan = make(chan struct{})
	pm.cmd = exec.Command(cfg.Command[0], cfg.Command[1:]...)
	pm.cmd.Stdout = os.Stdout
	pm.cmd.Stderr = os.Stderr
	pm.cmd.Stdin = os.Stdin

	setProcessGroup(pm.cmd)

	if err := pm.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	go func() {
		pm.cmd.Wait()
		pm.mu.Lock()
		close(pm.doneChan)
		pm.cmd = nil
		pm.mu.Unlock()
	}()

	return nil
}

func (pm *ProcessManager) Stop(cfg *config.Config) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.cmd == nil || pm.cmd.Process == nil {
		return nil
	}

	pm.stopped = true

	if cfg.GracefulStop {
		return pm.gracefulStop(cfg.StopTimeout)
	}
	return pm.forceStop()
}

func (pm *ProcessManager) gracefulStop(timeout time.Duration) error {
	if err := terminateProcess(pm.cmd); err != nil {
		return pm.forceStop()
	}

	select {
	case <-pm.doneChan:
		return nil
	case <-time.After(timeout):
		return pm.forceStop()
	}
}

func (pm *ProcessManager) forceStop() error {
	return killProcess(pm.cmd)
}

func Watch(cfg *config.Config) error {
	logger := newLogger()
	repo := git.New(cfg.GitDir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lastCommit, err := repo.GetLatestCommit(ctx)
	if err != nil {
		return fmt.Errorf("failed to get initial commit: %w", err)
	}

	if cfg.Verbose {
		logger.Info("Starting watch with %v interval", cfg.PollInterval)
		logger.Info("Initial commit: %s", lastCommit)
		logger.Info("Command: %s", strings.Join(cfg.Command, " "))
	}

	pm := &ProcessManager{logger: logger}
	if err := pm.Start(cfg); err != nil {
		return err
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	var processExited bool
	for {
		select {
		case <-ticker.C:
			if err := checkAndUpdate(ctx, cfg, repo, &lastCommit, pm, processExited); err != nil && cfg.Verbose {
				logger.Error("Error during update check: %v", err)
			}
			processExited = false

		case <-pm.doneChan:
			if !processExited {
				processExited = true
				if cfg.Verbose {
					logger.Info("Process exited, waiting for changes before restart")
				}
			}

		case sig := <-sigChan:
			if cfg.Verbose {
				logger.Info("Received signal %v, shutting down...", sig)
			}
			// Stop the process and wait for it to finish before exiting
			if err := pm.Stop(cfg); err != nil {
				logger.Error("Error stopping process: %v", err)
				return err
			}
			// Wait for process to fully terminate
			select {
			case <-pm.doneChan:
				return nil
			case <-time.After(5 * time.Second):
				// Force exit if process doesn't terminate in time
				return fmt.Errorf("process failed to terminate gracefully")
			}
		}
	}
}

func checkAndUpdate(ctx context.Context, cfg *config.Config, repo *git.Repository, lastCommit *string, pm *ProcessManager, shouldStart bool) error {
	pullOutput, err := repo.Pull(ctx)
	if err != nil {
		return fmt.Errorf("failed to pull changes: %w", err)
	}

	currentCommit, err := repo.GetLatestCommit(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current commit: %w", err)
	}

	hasChanges := currentCommit != *lastCommit

	// Only restart if there are actual changes
	if hasChanges {
		if cfg.Verbose {
			pm.logger.Info("\nChanges detected!")
			pm.logger.Info("Previous commit: %s", *lastCommit)
			pm.logger.Info("New commit: %s", currentCommit)
			pm.logger.Info("Pull output: %s", pullOutput)
		}

		*lastCommit = currentCommit

		if err := pm.Stop(cfg); err != nil && cfg.Verbose {
			pm.logger.Error("Error stopping process: %v", err)
		}

		// Add a small delay to ensure process cleanup
		time.Sleep(100 * time.Millisecond)

		if err := pm.Start(cfg); err != nil {
			return fmt.Errorf("failed to restart command: %w", err)
		}
	}

	return nil
}
