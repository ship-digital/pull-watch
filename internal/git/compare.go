package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
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
func (r *GitRepository) CompareCommits(ctx context.Context, commitA, commitB string) (CommitComparisonResult, error) {
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
