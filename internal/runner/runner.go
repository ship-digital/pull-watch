package runner

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ship-digital/pull-watch/internal/config"
	"github.com/ship-digital/pull-watch/internal/git"
	"github.com/ship-digital/pull-watch/internal/logger"
)

// WatchOption configures the Watch function
type WatchOption func(*watchOptions)

type watchOptions struct {
	repository     git.Repository
	processManager Processor
}

// WithRepository sets a custom repository implementation
func WithRepository(repo git.Repository) WatchOption {
	return func(opts *watchOptions) {
		opts.repository = repo
	}
}

// WithProcessManager sets a custom process manager for testing
func WithProcessManager(pm Processor) WatchOption {
	return func(opts *watchOptions) {
		opts.processManager = pm
	}
}

func Run(cfg *config.Config, opts ...WatchOption) error {
	options := &watchOptions{}
	for _, opt := range opts {
		opt(options)
	}

	repo := options.repository
	if repo == nil {
		repo = git.New(cfg)
	}

	pm := options.processManager
	if pm == nil {
		pm = New(cfg)
	}

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
		cfg.Logger.MultiColor(
			logger.InfoSegment("Starting watch with "),
			logger.HighlightSegment(cfg.PollInterval.String()),
		)
		cfg.Logger.MultiColor(
			logger.InfoSegment("Local commit: "),
			logger.HighlightSegment(lastLocalCommit),
		)
		cfg.Logger.MultiColor(
			logger.InfoSegment("Remote commit: "),
			logger.HighlightSegment(lastRemoteCommit),
		)
		cfg.Logger.MultiColor(
			logger.InfoSegment("Command: "),
			logger.HighlightSegment(strings.Join(cfg.Command, " ")),
		)
	}

	comparison, err := repo.HandleCommitComparison(ctx, lastLocalCommit, lastRemoteCommit)
	if err != nil {
		return err
	}

	shouldStart := cfg.RunOnStart || comparison == git.AIsAncestorOfB

	if cfg.Verbose {
		if shouldStart && cfg.RunOnStart {
			cfg.Logger.MultiColor(
				logger.HighlightSegment("Starting"),
				logger.InfoSegment(" command on startup"),
			)
		} else if !shouldStart {
			cfg.Logger.MultiColor(
				logger.HighlightSegment("Not starting"),
				logger.InfoSegment(" command on startup (use -run-on-start to override)"),
			)
		}
	}

	if shouldStart {
		if err := pm.Start(); err != nil {
			return err
		}
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
				cfg.Logger.MultiColor(
					logger.ErrorSegment("Error during update check: "),
					logger.HighlightSegment(fmt.Sprintf("%v", err)),
				)
			}
			processExited = false

		case <-pm.GetDoneChan():
			if !processExited {
				processExited = true
				now := time.Now()

				// Initialize backoff on first exit
				if pm.GetBackoff() == 0 {
					pm.SetBackoff(initialBackoff)
				}

				// Log only if enough time has passed
				if now.Sub(pm.GetLastLogTime()) >= pm.GetBackoff() {
					if cfg.Verbose {
						cfg.Logger.Info("Process exited, waiting for changes before restart")
					}
					pm.SetLastLogTime(now)
					// Increase backoff for next time (cap at maxBackoff)
					newBackoff := time.Duration(float64(pm.GetBackoff()) * 1.5)
					if newBackoff > maxBackoff {
						newBackoff = maxBackoff
					}
					pm.SetBackoff(newBackoff)
				}
			}

		case sig := <-sigChan:
			if cfg.Verbose {
				cfg.Logger.MultiColor(
					logger.InfoSegment("Received signal "),
					logger.HighlightSegment(fmt.Sprintf("%v", sig)),
					logger.InfoSegment(", shutting down..."),
				)
			}
			// Stop the process and wait for it to finish before exiting
			if err := pm.Stop(); err != nil {
				cfg.Logger.MultiColor(
					logger.ErrorSegment("Error stopping process: "),
					logger.HighlightSegment(fmt.Sprintf("%v", err)),
				)
				return err
			}
			// Wait for process to fully terminate
			select {
			case <-pm.GetDoneChan():
				return nil
			case <-time.After(5 * time.Second):
				// Force exit if process doesn't terminate in time
				return fmt.Errorf("process failed to terminate gracefully")
			}
		}
	}
}

func checkAndUpdate(ctx context.Context, cfg *config.Config, repo git.Repository, lastCommit *string, pm Processor, shouldStart bool) error {
	localHash, err := repo.GetLatestCommit(ctx)
	if err != nil {
		return fmt.Errorf("failed to get local commit: %w", err)
	}

	remoteHash, err := repo.GetRemoteCommit(ctx)
	if err != nil {
		return fmt.Errorf("failed to get remote commit: %w", err)
	}

	comparison, err := repo.HandleCommitComparison(ctx, localHash, remoteHash)
	if err != nil {
		return err
	}

	if comparison == git.AIsAncestorOfB {
		if cfg.Verbose {
			pm.GetLogger().Info("\nChanges detected!")
		}

		*lastCommit = remoteHash

		if err := pm.Stop(); err != nil && cfg.Verbose {
			pm.GetLogger().MultiColor(
				logger.ErrorSegment("Error stopping process: "),
				logger.HighlightSegment(fmt.Sprintf("%v", err)),
			)
		}

		time.Sleep(100 * time.Millisecond)

		if err := pm.Start(); err != nil {
			return fmt.Errorf("failed to restart command: %w", err)
		}
	}

	return nil
}
