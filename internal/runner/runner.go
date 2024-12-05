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
			pm.logger.MultiColor(
				logger.ErrorSegment("Failed to clean up previous process: "),
				logger.HighlightSegment(fmt.Sprintf("%v", err)),
			)
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

// handleCommitComparison handles the commit comparison and decides whether to pull changes
func (pm *ProcessManager) handleCommitComparison(ctx context.Context, cfg *config.Config, repo git.Repository, localCommit, remoteCommit string) (git.CommitComparisonResult, error) {
	// Log commits if verbose
	if cfg.Verbose {
		pm.logger.MultiColor(
			logger.InfoSegment("Local commit: "),
			logger.HighlightSegment(localCommit),
		)
		pm.logger.MultiColor(
			logger.InfoSegment("Remote commit: "),
			logger.HighlightSegment(remoteCommit),
		)
	}

	// Compare commits
	comparison, err := repo.CompareCommits(ctx, localCommit, remoteCommit)
	if err != nil {
		return git.UnknownCommitComparisonResult, fmt.Errorf("failed to compare commits: %w", err)
	}

	// Handle different comparison results
	switch comparison {
	case git.AIsAncestorOfB:
		if cfg.Verbose {
			pm.logger.MultiColor(
				logger.InfoSegment("Local commit is "),
				logger.HighlightSegment("behind"),
				logger.InfoSegment(" remote commit, "),
				logger.HighlightSegment("pulling changes..."),
			)
		}
		if _, err := repo.Pull(ctx); err != nil {
			return git.UnknownCommitComparisonResult, fmt.Errorf("failed to pull changes: %w", err)
		}
		return git.AIsAncestorOfB, nil

	case git.BIsAncestorOfA:
		if cfg.Verbose {
			pm.logger.MultiColor(
				logger.InfoSegment("Local commit is "),
				logger.HighlightSegment("ahead"),
				logger.InfoSegment(" of remote commit, "),
				logger.HighlightSegment("not pulling."),
			)
		}
		return git.BIsAncestorOfA, nil

	case git.CommitsDiverged:
		if cfg.Verbose {
			pm.logger.MultiColor(
				logger.InfoSegment("Local commit and remote commit "),
				logger.HighlightSegment("have diverged"),
				logger.InfoSegment(": "),
				logger.HighlightSegment("not pulling."),
			)
		}
		return git.CommitsDiverged, nil

	case git.CommitsEqual:
		if cfg.Verbose {
			pm.logger.MultiColor(
				logger.InfoSegment("Local commit and remote commit "),
				logger.HighlightSegment("are the same"),
				logger.InfoSegment(": "),
				logger.HighlightSegment("not pulling."),
			)
		}
		return git.CommitsEqual, nil

	default:
		return git.UnknownCommitComparisonResult, fmt.Errorf("unknown commit comparison result: %v", comparison)
	}
}

// WatchOption configures the Watch function
type WatchOption func(*watchOptions)

type watchOptions struct {
	repository git.Repository
}

// WithRepository sets a custom repository implementation
func WithRepository(repo git.Repository) WatchOption {
	return func(opts *watchOptions) {
		opts.repository = repo
	}
}

func Watch(cfg *config.Config, opts ...WatchOption) error {
	options := &watchOptions{}
	for _, opt := range opts {
		opt(options)
	}

	repo := options.repository
	if repo == nil {
		repo = git.New(cfg)
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

	pm := NewProcessManager(cfg)

	comparison, err := pm.handleCommitComparison(ctx, cfg, repo, lastLocalCommit, lastRemoteCommit)
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
		if err := pm.Start(cfg); err != nil {
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
						cfg.Logger.Info("Process exited, waiting for changes before restart")
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
				cfg.Logger.MultiColor(
					logger.InfoSegment("Received signal "),
					logger.HighlightSegment(fmt.Sprintf("%v", sig)),
					logger.InfoSegment(", shutting down..."),
				)
			}
			// Stop the process and wait for it to finish before exiting
			if err := pm.Stop(cfg); err != nil {
				cfg.Logger.MultiColor(
					logger.ErrorSegment("Error stopping process: "),
					logger.HighlightSegment(fmt.Sprintf("%v", err)),
				)
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

func checkAndUpdate(ctx context.Context, cfg *config.Config, repo git.Repository, lastCommit *string, pm *ProcessManager, shouldStart bool) error {
	localHash, err := repo.GetLatestCommit(ctx)
	if err != nil {
		return fmt.Errorf("failed to get local commit: %w", err)
	}

	remoteHash, err := repo.GetRemoteCommit(ctx)
	if err != nil {
		return fmt.Errorf("failed to get remote commit: %w", err)
	}

	comparison, err := pm.handleCommitComparison(ctx, cfg, repo, localHash, remoteHash)
	if err != nil {
		return err
	}

	if comparison == git.AIsAncestorOfB {
		if cfg.Verbose {
			pm.logger.Info("\nChanges detected!")
		}

		if _, err := pm.handleCommitComparison(ctx, cfg, repo, localHash, remoteHash); err != nil {
			return err
		}

		*lastCommit = remoteHash

		if err := pm.Stop(cfg); err != nil && cfg.Verbose {
			pm.logger.MultiColor(
				logger.ErrorSegment("Error stopping process: "),
				logger.HighlightSegment(fmt.Sprintf("%v", err)),
			)
		}

		time.Sleep(100 * time.Millisecond)

		if err := pm.Start(cfg); err != nil {
			return fmt.Errorf("failed to restart command: %w", err)
		}
	}

	return nil
}

func NewProcessManager(cfg *config.Config) *ProcessManager {
	return &ProcessManager{
		logger: cfg.Logger,
	}
}
