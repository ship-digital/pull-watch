package git

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ship-digital/pull-watch/internal/executor"
)

type Repository struct {
	dir      string
	executor executor.CommandExecutor
}

func New(dir string) *Repository {
	return &Repository{
		dir:      dir,
		executor: &executor.DefaultExecutor{Dir: dir},
	}
}

// NewWithExecutor creates a new Repository with a custom executor (useful for testing)
func NewWithExecutor(dir string, exec executor.CommandExecutor) *Repository {
	return &Repository{
		dir:      dir,
		executor: exec,
	}
}

func (r *Repository) execGitCmd(ctx context.Context, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	return r.executor.ExecuteCommand(ctx, "git", args...)
}

func (r *Repository) GetLatestCommit(ctx context.Context) (string, error) {
	output, err := r.execGitCmd(ctx, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

func (r *Repository) Fetch(ctx context.Context) error {
	_, err := r.executor.ExecuteCommand(ctx, "git", "-C", r.dir, "fetch")
	return err
}

func (r *Repository) Pull(ctx context.Context) (string, error) {
	return r.execGitCmd(ctx, "pull")
}

func (r *Repository) GetRemoteCommit(ctx context.Context) (string, error) {
	remoteBranch, err := r.execGitCmd(ctx, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	if err != nil {
		return "", fmt.Errorf("failed to get tracking branch: %w", err)
	}

	parts := strings.SplitN(remoteBranch, "/", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid tracking branch format: %s", remoteBranch)
	}
	remote := parts[0]

	output, err := r.execGitCmd(ctx, "ls-remote", remote, "HEAD")
	if err != nil {
		return "", err
	}
	hash := strings.Split(output, "\t")[0]
	return hash, nil
}

// GetCurrentBranch returns the name of the current branch
func (r *Repository) GetCurrentBranch(ctx context.Context) (string, error) {
	output, err := r.executor.ExecuteCommand(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// IsClean returns true if the working directory is clean (no uncommitted changes)
func (r *Repository) IsClean(ctx context.Context) (bool, error) {
	output, err := r.executor.ExecuteCommand(ctx, "git", "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(output) == "", nil
}
