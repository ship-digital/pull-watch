package git

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// MockExecutor implements CommandExecutor for testing
type MockExecutor struct {
	Responses map[string]struct {
		Output string
		Error  error
	}
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
			repo := NewWithExecutor("/fake/dir", mockExecutor)
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
	}{
		{
			name: "successful remote commit lookup",
			mockResp: map[string]struct {
				Output string
				Error  error
			}{
				"git rev-parse --abbrev-ref --symbolic-full-name @{u}": {
					Output: "origin/main\n",
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
				"git ls-remote origin HEAD": {
					Output: "",
					Error:  fmt.Errorf("fatal: unable to access 'https://github.com/user/repo.git/': Failed to connect"),
				},
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := &MockExecutor{Responses: tt.mockResp}
			repo := NewWithExecutor("/fake/dir", mockExecutor)
			got, err := repo.GetRemoteCommit(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("GetRemoteCommit() error = %v, wantErr %v", err, tt.wantErr)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := &MockExecutor{Responses: tt.mockResp}
			repo := NewWithExecutor("/fake/dir", mockExecutor)
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
