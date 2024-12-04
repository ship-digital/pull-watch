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

	"github.com/ship-digital/pull-watch/internal/config"
	"github.com/ship-digital/pull-watch/internal/git"
)

type ProcessManager struct {
	mu       sync.Mutex
	cmd      *exec.Cmd
	doneChan chan struct{}
}

func (pm *ProcessManager) Start(cfg *config.Config) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.cmd != nil {
		return fmt.Errorf("process already running")
	}

	pm.doneChan = make(chan struct{})
	pm.cmd = exec.Command(cfg.Command[0], cfg.Command[1:]...)
	pm.cmd.Stdout = os.Stdout
	pm.cmd.Stderr = os.Stderr
	pm.cmd.Stdin = os.Stdin

	if err := pm.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	go func() {
		pm.cmd.Wait()
		close(pm.doneChan)
	}()

	return nil
}

func (pm *ProcessManager) Stop() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.cmd == nil || pm.cmd.Process == nil {
		return nil
	}

	// Try SIGTERM first
	if err := pm.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return pm.cmd.Process.Kill()
	}

	// Wait for graceful shutdown with timeout
	select {
	case <-pm.doneChan:
	case <-time.After(5 * time.Second):
		return pm.cmd.Process.Kill()
	}

	return nil
}

func Watch(cfg *config.Config) error {
	repo := git.New(cfg.GitDir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Get initial commit
	lastCommit, err := repo.GetLatestCommit(ctx)
	if err != nil {
		return fmt.Errorf("failed to get initial commit: %w", err)
	}

	if cfg.Verbose {
		log.Printf("Starting watch with %v interval\n", cfg.PollInterval)
		log.Printf("Initial commit: %s\n", lastCommit)
		log.Printf("Command: %s\n", strings.Join(cfg.Command, " "))
	}

	pm := &ProcessManager{}
	if err := pm.Start(cfg); err != nil {
		return err
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := checkAndUpdate(ctx, cfg, repo, &lastCommit, pm); err != nil && cfg.Verbose {
				log.Printf("Error during update check: %v\n", err)
			}

		case <-pm.doneChan:
			if err := pm.Start(cfg); err != nil && cfg.Verbose {
				log.Printf("Error restarting command: %v\n", err)
			}

		case sig := <-sigChan:
			if cfg.Verbose {
				log.Printf("Received signal %v, shutting down...\n", sig)
			}
			return pm.Stop()
		}
	}
}

func checkAndUpdate(ctx context.Context, cfg *config.Config, repo *git.Repository, lastCommit *string, pm *ProcessManager) error {
	pullOutput, err := repo.Pull(ctx)
	if err != nil {
		return fmt.Errorf("failed to pull changes: %w", err)
	}

	currentCommit, err := repo.GetLatestCommit(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current commit: %w", err)
	}

	if currentCommit != *lastCommit {
		if cfg.Verbose {
			log.Printf("\nChanges detected!\n")
			log.Printf("Previous commit: %s\n", *lastCommit)
			log.Printf("New commit: %s\n", currentCommit)
			log.Printf("Pull output: %s\n", pullOutput)
		}

		*lastCommit = currentCommit

		if err := pm.Stop(); err != nil && cfg.Verbose {
			log.Printf("Error stopping process: %v\n", err)
		}

		if err := pm.Start(cfg); err != nil {
			return fmt.Errorf("failed to restart command: %w", err)
		}
	}

	return nil
}
