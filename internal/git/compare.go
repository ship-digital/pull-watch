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

// IsAncestor checks if commitA is an ancestor of commitB
func (r *GitRepository) IsAncestor(ctx context.Context, commitA, commitB string) (bool, error) {
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
	if repo.cfg.Verbose {
		repo.cfg.Logger.MultiColor(
			logger.InfoSegment("Local commit: "),
			logger.HighlightSegment(localCommit),
		)
		repo.cfg.Logger.MultiColor(
			logger.InfoSegment("Remote commit: "),
			logger.HighlightSegment(remoteCommit),
		)
	}

	// Compare commits
	comparison, err := repo.compareCommits(ctx, localCommit, remoteCommit)
	if err != nil {
		return UnknownCommitComparisonResult, fmt.Errorf("failed to compare commits: %w", err)
	}

	// Handle different comparison results
	switch comparison {
	case AIsAncestorOfB:
		if repo.cfg.Verbose {
			repo.cfg.Logger.MultiColor(
				logger.InfoSegment("Local commit is "),
				logger.HighlightSegment("behind"),
				logger.InfoSegment(" remote commit, "),
				logger.HighlightSegment("pulling changes..."),
			)
		}
		if _, err := repo.Pull(ctx); err != nil {
			return UnknownCommitComparisonResult, fmt.Errorf("failed to pull changes: %w", err)
		}
		return AIsAncestorOfB, nil

	case BIsAncestorOfA:
		if repo.cfg.Verbose {
			repo.cfg.Logger.MultiColor(
				logger.InfoSegment("Local commit is "),
				logger.HighlightSegment("ahead"),
				logger.InfoSegment(" of remote commit, "),
				logger.HighlightSegment("not pulling."),
			)
		}
		return BIsAncestorOfA, nil

	case CommitsDiverged:
		if repo.cfg.Verbose {
			repo.cfg.Logger.MultiColor(
				logger.InfoSegment("Local commit and remote commit "),
				logger.HighlightSegment("have diverged"),
				logger.InfoSegment(": "),
				logger.HighlightSegment("not pulling."),
			)
		}
		return CommitsDiverged, nil

	case CommitsEqual:
		if repo.cfg.Verbose {
			repo.cfg.Logger.MultiColor(
				logger.InfoSegment("Local commit and remote commit "),
				logger.HighlightSegment("are the same"),
				logger.InfoSegment(": "),
				logger.HighlightSegment("not pulling."),
			)
		}
		return CommitsEqual, nil

	default:
		return UnknownCommitComparisonResult, fmt.Errorf("unknown commit comparison result: %v", comparison)
	}
}
