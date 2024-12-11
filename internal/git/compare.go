package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/ship-digital/pull-watch/internal/logger"
)

type CommitComparisonResult int

const (
	UnknownCommitComparisonResult CommitComparisonResult = -2
	AIsAncestorOfB                CommitComparisonResult = -1
	CommitsEqual                  CommitComparisonResult = 0
	BIsAncestorOfA                CommitComparisonResult = 1
	CommitsDiverged               CommitComparisonResult = 2
)

// commitExistsLocally checks if a commit exists in the local repository
func (r *GitRepository) commitExistsLocally(ctx context.Context, commit string) bool {
	_, err := r.executor.ExecuteCommand(ctx, "git", "cat-file", "-e", commit)
	return err == nil
}

// IsAncestor checks if commitA is an ancestor of commitB
func (r *GitRepository) IsAncestor(ctx context.Context, commitA, commitB string) (bool, error) {
	// Check if commits exist locally
	for _, commit := range []string{commitA, commitB} {
		if !r.commitExistsLocally(ctx, commit) {
			// Commit not found locally, try fetching
			if err := r.Fetch(ctx); err != nil {
				return false, fmt.Errorf("failed to fetch after missing commit %s: %w", commit, err)
			}
			// Verify commit exists after fetch
			if !r.commitExistsLocally(ctx, commit) {
				return false, fmt.Errorf("commit %s not found even after fetch", commit)
			}
			break // Only need to fetch once
		}
	}

	_, err := r.executor.ExecuteCommand(ctx, "git", "-C", r.cfg.GitDir, "merge-base", "--is-ancestor", commitA, commitB)
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			// Exit code 1 means commitA is not an ancestor of commitB
			return false, nil
		}
		// Return any other error
		return false, fmt.Errorf("failed to check ancestry: %w", err)
	}
	// If no error, commitA is an ancestor of commitB
	return true, nil
}

// CompareCommits compares two commits and returns the result
func (r *GitRepository) compareCommits(ctx context.Context, commitA, commitB string) (CommitComparisonResult, error) {
	if commitA == commitB {
		return CommitsEqual, nil
	}

	isAncestorAB, err := r.IsAncestor(ctx, commitA, commitB)
	if err != nil {
		return UnknownCommitComparisonResult, err
	}

	isAncestorBA, err := r.IsAncestor(ctx, commitB, commitA)
	if err != nil {
		return UnknownCommitComparisonResult, err
	}

	if isAncestorAB {
		return AIsAncestorOfB, nil
	}
	if isAncestorBA {
		return BIsAncestorOfA, nil
	}
	return CommitsDiverged, nil
}

// HandleCommitComparison handles the commit comparison and decides whether to pull changes
func (repo *GitRepository) HandleCommitComparison(ctx context.Context, localCommit, remoteCommit string) (CommitComparisonResult, error) {
	// Log commits if verbose
	repo.cfg.Logger.MultiColor(logger.VerboseLevel,
		logger.InfoSegment("Local commit: "),
		logger.HighlightSegment(localCommit),
	)
	repo.cfg.Logger.MultiColor(logger.VerboseLevel,
		logger.InfoSegment("Remote commit: "),
		logger.HighlightSegment(remoteCommit),
	)

	// Compare commits
	comparison, err := repo.compareCommits(ctx, localCommit, remoteCommit)
	if err != nil {
		return UnknownCommitComparisonResult, fmt.Errorf("failed to compare commits: %w", err)
	}

	// Handle different comparison results
	switch comparison {
	case AIsAncestorOfB:
		repo.cfg.Logger.MultiColor(logger.DefaultLevel,
			logger.InfoSegment("Local commit is "),
			logger.HighlightSegment("behind"),
			logger.InfoSegment(" remote commit, "),
			logger.HighlightSegment("pulling changes..."),
		)

		if _, err := repo.Pull(ctx); err != nil {
			return UnknownCommitComparisonResult, fmt.Errorf("failed to pull changes: %w", err)
		}
		return AIsAncestorOfB, nil

	case BIsAncestorOfA:
		repo.cfg.Logger.MultiColor(logger.VerboseLevel,
			logger.InfoSegment("Local commit is "),
			logger.HighlightSegment("ahead"),
			logger.InfoSegment(" of remote commit, "),
			logger.HighlightSegment("not pulling."),
		)
		return BIsAncestorOfA, nil

	case CommitsDiverged:
		repo.cfg.Logger.MultiColor(logger.VerboseLevel,
			logger.InfoSegment("Local commit and remote commit "),
			logger.HighlightSegment("have diverged"),
			logger.InfoSegment(": "),
			logger.HighlightSegment("not pulling."),
		)
		return CommitsDiverged, nil

	case CommitsEqual:
		repo.cfg.Logger.MultiColor(logger.VerboseLevel,
			logger.InfoSegment("Local commit and remote commit "),
			logger.HighlightSegment("are the same"),
			logger.InfoSegment(": "),
			logger.HighlightSegment("not pulling."),
		)

		return CommitsEqual, nil

	default:
		return UnknownCommitComparisonResult, fmt.Errorf("unknown commit comparison result: %v", comparison)
	}
}
