package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ship-digital/pull-watch/internal/config"
	"github.com/ship-digital/pull-watch/internal/git"
	"github.com/ship-digital/pull-watch/internal/logger"
)

var (
	initialBackoff = 5 * time.Second
	maxBackoff     = 5 * time.Minute
)

type ProcessManager struct {
	mu          sync.Mutex
	cmd         *exec.Cmd
	doneChan    chan struct{}
	stopped     bool
	logger      *logger.Logger
	lastLogTime time.Time
	backoff     time.Duration
}

func (pm *ProcessManager) Start(cfg *config.Config) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Reset backoff when starting new process
	pm.backoff = 0
	pm.lastLogTime = time.Time{}

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
	log := logger.New()
	repo := git.New(cfg.GitDir, cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lastLocalCommit, err := repo.GetLatestCommit(ctx)
	if err != nil {
		return fmt.Errorf("failed to get initial commit: %w", err)
	}

	lastRemoteCommit, err := repo.GetRemoteCommit(ctx)
	if err != nil {
		return fmt.Errorf("failed to get initial remote commit: %w", err)
	}

	if cfg.Verbose {
		log.Info("Starting watch with %v interval", cfg.PollInterval)
		log.Info("Local commit: %s", lastLocalCommit)
		log.Info("Remote commit: %s", lastRemoteCommit)
		log.Info("Command: %s", strings.Join(cfg.Command, " "))
	}

	if lastLocalCommit != lastRemoteCommit {
		log.Info("Pulling changes...")
		if _, err := repo.Pull(ctx); err != nil {
			return fmt.Errorf("failed to pull initial changes: %w", err)
		}
	}

	pm := &ProcessManager{logger: log}
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
			if err := checkAndUpdate(ctx, cfg, repo, &lastLocalCommit, pm, processExited); err != nil && cfg.Verbose {
				log.Error("Error during update check: %v", err)
			}
			processExited = false

		case <-pm.doneChan:
			if !processExited {
				processExited = true
				now := time.Now()

				// Initialize backoff on first exit
				if pm.backoff == 0 {
					pm.backoff = initialBackoff
				}

				// Log only if enough time has passed
				if now.Sub(pm.lastLogTime) >= pm.backoff {
					if cfg.Verbose {
						log.Info("Process exited, waiting for changes before restart")
					}
					pm.lastLogTime = now
					// Increase backoff for next time (cap at maxBackoff)
					pm.backoff = time.Duration(float64(pm.backoff) * 1.5)
					if pm.backoff > maxBackoff {
						pm.backoff = maxBackoff
					}
				}
			}

		case sig := <-sigChan:
			if cfg.Verbose {
				log.Info("Received signal %v, shutting down...", sig)
			}
			// Stop the process and wait for it to finish before exiting
			if err := pm.Stop(cfg); err != nil {
				log.Error("Error stopping process: %v", err)
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
	// Get local and remote hashes
	localHash, err := repo.GetLatestCommit(ctx)
	if err != nil {
		return fmt.Errorf("failed to get local commit: %w", err)
	}

	remoteHash, err := repo.GetRemoteCommit(ctx)
	if err != nil {
		return fmt.Errorf("failed to get remote commit: %w", err)
	}

	// Only pull and restart if remote hash differs
	if remoteHash != localHash {
		if cfg.Verbose {
			pm.logger.Info("\nChanges detected!")
			pm.logger.Info("Local commit: %s", localHash)
			pm.logger.Info("Remote commit: %s", remoteHash)
		}

		pullOutput, err := repo.Pull(ctx)
		if err != nil {
			return fmt.Errorf("failed to pull changes: %w", err)
		}

		if cfg.Verbose {
			pm.logger.Info("Pull output: %s", pullOutput)
		}

		*lastCommit = remoteHash

		if err := pm.Stop(cfg); err != nil && cfg.Verbose {
			pm.logger.Error("Error stopping process: %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		if err := pm.Start(cfg); err != nil {
			return fmt.Errorf("failed to restart command: %w", err)
		}
	}

	return nil
}
