package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type Repository struct {
	dir string
}

func New(dir string) *Repository {
	return &Repository{dir: dir}
}

func (r *Repository) execGitCmd(ctx context.Context, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = r.dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git command failed: %v\nstderr: %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

func (r *Repository) GetLatestCommit(ctx context.Context) (string, error) {
	return r.execGitCmd(ctx, "rev-parse", "HEAD")
}

func (r *Repository) Pull(ctx context.Context) (string, error) {
	return r.execGitCmd(ctx, "pull")
}
