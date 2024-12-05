package git

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ship-digital/pull-watch/internal/config"
	"github.com/ship-digital/pull-watch/internal/errz"
	"github.com/ship-digital/pull-watch/internal/executor"
)

// Repository defines the interface for git operations
type Repository interface {
	GetLatestCommit(ctx context.Context) (string, error)
	GetRemoteCommit(ctx context.Context) (string, error)
	Fetch(ctx context.Context) error
	Pull(ctx context.Context) (string, error)
	GetCurrentBranch(ctx context.Context) (string, error)
	IsClean(ctx context.Context) (bool, error)
	CompareCommits(ctx context.Context, local, remote string) (CommitComparisonResult, error)
}

// GitRepository implements the Repository interface
type GitRepository struct {
	cfg      *config.Config
	executor executor.CommandExecutor
}

// Option configures a GitRepository
type Option func(*GitRepository)

// WithExecutor sets a custom executor
func WithExecutor(exec executor.CommandExecutor) Option {
	return func(r *GitRepository) {
		r.executor = exec
	}
}

func New(cfg *config.Config, opts ...Option) *GitRepository {
	r := &GitRepository{
		cfg:      cfg,
		executor: executor.New(cfg),
	}

	for _, opt := range opts {
		opt(r)
	}

	return r
}

func (r *GitRepository) execGitCmd(ctx context.Context, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	return r.executor.ExecuteCommand(ctx, "git", args...)
}

func (r *GitRepository) GetLatestCommit(ctx context.Context) (string, error) {
	output, err := r.execGitCmd(ctx, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

func (r *GitRepository) Fetch(ctx context.Context) error {
	_, err := r.executor.ExecuteCommand(ctx, "git", "-C", r.cfg.GitDir, "fetch")
	return err
}

func (r *GitRepository) Pull(ctx context.Context) (string, error) {
	return r.execGitCmd(ctx, "pull")
}

func (r *GitRepository) GetRemoteCommit(ctx context.Context) (string, error) {
	// Get upstream branch (e.g. "origin/main")
	remoteBranch, err := r.execGitCmd(ctx, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	if err != nil {
		if strings.Contains(err.Error(), "no upstream") {
			return "", errz.ErrNoUpstreamBranch
		}
		return "", fmt.Errorf("failed to get tracking branch: %w", err)
	}

	// Split into remote and branch (e.g. ["origin", "main"])
	parts := strings.SplitN(strings.TrimSpace(remoteBranch), "/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid tracking branch format: %s", remoteBranch)
	}
	remote := parts[0]
	branch := parts[1]

	// Try specific branch first
	output, err := r.execGitCmd(ctx, "ls-remote", remote, fmt.Sprintf("refs/heads/%s", branch))
	if err != nil {
		return "", err
	}

	// If no output, try HEAD as fallback (for default branches)
	if strings.TrimSpace(output) == "" {
		output, err = r.execGitCmd(ctx, "ls-remote", remote, "HEAD")
		if err != nil {
			return "", err
		}
	}

	parts = strings.Split(strings.TrimSpace(output), "\t")
	if len(parts) == 0 {
		return "", fmt.Errorf("unexpected ls-remote output format")
	}
	return parts[0], nil
}

// GetCurrentBranch returns the name of the current branch
func (r *GitRepository) GetCurrentBranch(ctx context.Context) (string, error) {
	output, err := r.executor.ExecuteCommand(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// IsClean returns true if the working directory is clean (no uncommitted changes)
func (r *GitRepository) IsClean(ctx context.Context) (bool, error) {
	output, err := r.executor.ExecuteCommand(ctx, "git", "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(output) == "", nil
}
