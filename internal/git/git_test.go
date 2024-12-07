package git

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ship-digital/pull-watch/internal/config"
	"github.com/ship-digital/pull-watch/internal/errz"
)

// MockExecutor implements CommandExecutor for testing
type MockExecutor struct {
	Responses map[string]struct {
		Output string
		Error  error
	}
	Config *config.Config
}

func (m *MockExecutor) GetConfig() *config.Config {
	return m.Config
}

func (m *MockExecutor) ExecuteCommand(ctx context.Context, name string, args ...string) (string, error) {
	key := fmt.Sprintf("%s %s", name, strings.Join(args, " "))
	if response, ok := m.Responses[key]; ok {
		return response.Output, response.Error
	}
	return "", fmt.Errorf("unexpected command: %s", key)
}

func TestGetLatestCommit(t *testing.T) {
	tests := []struct {
		name     string
		mockResp map[string]struct {
			Output string
			Error  error
		}
		want    string
		wantErr bool
	}{
		{
			name: "successful commit lookup",
			mockResp: map[string]struct {
				Output string
				Error  error
			}{
				"git rev-parse HEAD": {
					Output: "abcdef0123456789\n",
					Error:  nil,
				},
			},
			want:    "abcdef0123456789",
			wantErr: false,
		},
		{
			name: "git command error",
			mockResp: map[string]struct {
				Output string
				Error  error
			}{
				"git rev-parse HEAD": {
					Output: "",
					Error:  fmt.Errorf("fatal: not a git repository"),
				},
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := &MockExecutor{Responses: tt.mockResp}
			repo := New(&config.Config{GitDir: "/fake/dir"}, WithExecutor(mockExecutor))
			got, err := repo.GetLatestCommit(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("GetLatestCommit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetLatestCommit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetRemoteCommit(t *testing.T) {
	tests := []struct {
		name     string
		mockResp map[string]struct {
			Output string
			Error  error
		}
		want    string
		wantErr bool
		errType error
	}{
		{
			name: "successful remote commit lookup - specific branch",
			mockResp: map[string]struct {
				Output string
				Error  error
			}{
				"git rev-parse --abbrev-ref --symbolic-full-name @{u}": {
					Output: "origin/feature\n",
					Error:  nil,
				},
				"git ls-remote origin refs/heads/feature": {
					Output: "abcdef0123456789\trefs/heads/feature\n",
					Error:  nil,
				},
			},
			want:    "abcdef0123456789",
			wantErr: false,
		},
		{
			name: "successful remote commit lookup - default branch",
			mockResp: map[string]struct {
				Output string
				Error  error
			}{
				"git rev-parse --abbrev-ref --symbolic-full-name @{u}": {
					Output: "origin/main\n",
					Error:  nil,
				},
				"git ls-remote origin refs/heads/main": {
					Output: "", // No output for specific branch
					Error:  nil,
				},
				"git ls-remote origin HEAD": {
					Output: "abcdef0123456789\tHEAD\n",
					Error:  nil,
				},
			},
			want:    "abcdef0123456789",
			wantErr: false,
		},
		{
			name: "no upstream branch",
			mockResp: map[string]struct {
				Output string
				Error  error
			}{
				"git rev-parse --abbrev-ref --symbolic-full-name @{u}": {
					Output: "",
					Error:  fmt.Errorf("fatal: no upstream branch"),
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "remote unreachable",
			mockResp: map[string]struct {
				Output string
				Error  error
			}{
				"git rev-parse --abbrev-ref --symbolic-full-name @{u}": {
					Output: "origin/main\n",
					Error:  nil,
				},
				"git ls-remote origin refs/heads/main": {
					Output: "",
					Error:  fmt.Errorf("fatal: unable to access 'https://github.com/user/repo.git/': Failed to connect"),
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "origin remote doesn't exist",
			mockResp: map[string]struct {
				Output string
				Error  error
			}{
				"git remote get-url origin": {
					Output: "",
					Error:  fmt.Errorf("fatal: 'origin' does not appear to be a git repository"),
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "no remotes configured",
			mockResp: map[string]struct {
				Output string
				Error  error
			}{
				"git remote -v": {
					Output: "",
					Error:  nil,
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "no upstream branch configured",
			mockResp: map[string]struct {
				Output string
				Error  error
			}{
				"git rev-parse --abbrev-ref --symbolic-full-name @{u}": {
					Output: "",
					Error:  fmt.Errorf("fatal: no upstream configured for branch 'ls-remote'\nexit status 128"),
				},
			},
			want:    "",
			wantErr: true,
			errType: errz.ErrNoUpstreamBranch,
		},
		{
			name: "remote with special characters",
			mockResp: map[string]struct {
				Output string
				Error  error
			}{
				"git rev-parse --abbrev-ref --symbolic-full-name @{u}": {
					Output: "upstream.gitlab/feature\n",
					Error:  nil,
				},
				"git ls-remote upstream.gitlab refs/heads/feature": {
					Output: "abcdef0123456789\trefs/heads/feature\n",
					Error:  nil,
				},
			},
			want:    "abcdef0123456789",
			wantErr: false,
		},
		{
			name: "branch with special characters",
			mockResp: map[string]struct {
				Output string
				Error  error
			}{
				"git rev-parse --abbrev-ref --symbolic-full-name @{u}": {
					Output: "origin/feature/with/slashes-and.dots\n",
					Error:  nil,
				},
				"git ls-remote origin refs/heads/feature/with/slashes-and.dots": {
					Output: "abcdef0123456789\trefs/heads/feature/with/slashes-and.dots\n",
					Error:  nil,
				},
			},
			want:    "abcdef0123456789",
			wantErr: false,
		},
		{
			name: "unicode branch name",
			mockResp: map[string]struct {
				Output string
				Error  error
			}{
				"git rev-parse --abbrev-ref --symbolic-full-name @{u}": {
					Output: "origin/feature/🚀-emoji\n",
					Error:  nil,
				},
				"git ls-remote origin refs/heads/feature/🚀-emoji": {
					Output: "abcdef0123456789\trefs/heads/feature/🚀-emoji\n",
					Error:  nil,
				},
			},
			want:    "abcdef0123456789",
			wantErr: false,
		},
		{
			name: "multiple remotes with same branch",
			mockResp: map[string]struct {
				Output string
				Error  error
			}{
				"git rev-parse --abbrev-ref --symbolic-full-name @{u}": {
					Output: "upstream/main\n",
					Error:  nil,
				},
				"git ls-remote upstream refs/heads/main": {
					Output: "", // No output for specific branch
					Error:  nil,
				},
				"git ls-remote upstream HEAD": {
					Output: "abcdef0123456789\tHEAD\n",
					Error:  nil,
				},
			},
			want:    "abcdef0123456789",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := &MockExecutor{Responses: tt.mockResp}
			repo := New(&config.Config{GitDir: "/fake/dir"}, WithExecutor(mockExecutor))
			got, err := repo.GetRemoteCommit(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("GetRemoteCommit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.errType != nil && err != tt.errType {
				t.Errorf("GetRemoteCommit() error = %v, want error type %v", err, tt.errType)
				return
			}

			if got != tt.want {
				t.Errorf("GetRemoteCommit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPull(t *testing.T) {
	tests := []struct {
		name     string
		mockResp map[string]struct {
			Output string
			Error  error
		}
		want    string
		wantErr bool
	}{
		{
			name: "successful pull - fast-forward",
			mockResp: map[string]struct {
				Output string
				Error  error
			}{
				"git pull": {
					Output: "Updating abcdef0..123456\nFast-forward\n main.go | 2 +-\n 1 file changed",
					Error:  nil,
				},
			},
			want:    "Updating abcdef0..123456\nFast-forward\n main.go | 2 +-\n 1 file changed",
			wantErr: false,
		},
		{
			name: "pull with merge conflicts",
			mockResp: map[string]struct {
				Output string
				Error  error
			}{
				"git pull": {
					Output: "",
					Error:  fmt.Errorf("error: Your local changes would be overwritten by merge"),
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "pull with no changes",
			mockResp: map[string]struct {
				Output string
				Error  error
			}{
				"git pull": {
					Output: "Already up to date.",
					Error:  nil,
				},
			},
			want:    "Already up to date.",
			wantErr: false,
		},
		{
			name: "pull with network error",
			mockResp: map[string]struct {
				Output string
				Error  error
			}{
				"git pull": {
					Output: "",
					Error:  fmt.Errorf("fatal: unable to access: Could not resolve host"),
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "origin remote doesn't exist during pull",
			mockResp: map[string]struct {
				Output string
				Error  error
			}{
				"git pull": {
					Output: "",
					Error:  fmt.Errorf("fatal: 'origin' does not appear to be a git repository"),
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "no remotes configured during pull",
			mockResp: map[string]struct {
				Output string
				Error  error
			}{
				"git pull": {
					Output: "",
					Error:  fmt.Errorf("fatal: No remote repository specified."),
				},
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := &MockExecutor{Responses: tt.mockResp}
			repo := New(&config.Config{GitDir: "/fake/dir"}, WithExecutor(mockExecutor))
			got, err := repo.Pull(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("Pull() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Pull() = %v, want %v", got, tt.want)
			}
		})
	}
}
