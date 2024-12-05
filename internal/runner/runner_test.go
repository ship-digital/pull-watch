package runner

import (
	"context"
	"fmt"
	"syscall"
	"testing"
	"time"

	"github.com/ship-digital/pull-watch/internal/config"
	"github.com/ship-digital/pull-watch/internal/git"
	"github.com/ship-digital/pull-watch/internal/logger"
)

// MockRepo implements a mock git repository for testing
type MockRepo struct {
	localCommits  []string
	remoteCommits []string
	pullError     error
	compareResult git.CommitComparisonResult
	compareError  error
	currentIndex  int
}

func (m *MockRepo) GetLatestCommit(ctx context.Context) (string, error) {
	if m.currentIndex >= len(m.localCommits) {
		return "", fmt.Errorf("no more local commits")
	}
	commit := m.localCommits[m.currentIndex]
	return commit, nil
}

func (m *MockRepo) GetRemoteCommit(ctx context.Context) (string, error) {
	if m.currentIndex >= len(m.remoteCommits) {
		return "", fmt.Errorf("no more remote commits")
	}
	commit := m.remoteCommits[m.currentIndex]
	return commit, nil
}

func (m *MockRepo) Pull(ctx context.Context) (string, error) {
	if m.pullError != nil {
		return "", m.pullError
	}
	m.currentIndex++
	return "Changes pulled successfully", nil
}

func (m *MockRepo) CompareCommits(ctx context.Context, local, remote string) (git.CommitComparisonResult, error) {
	if m.compareError != nil {
		return git.UnknownCommitComparisonResult, m.compareError
	}
	return m.compareResult, nil
}

func (m *MockRepo) Fetch(ctx context.Context) error {
	return nil // For testing we can just return nil
}

func (m *MockRepo) GetCurrentBranch(ctx context.Context) (string, error) {
	return "main", nil // For testing we can return a fixed branch
}

func (m *MockRepo) IsClean(ctx context.Context) (bool, error) {
	return true, nil // For testing we can assume the repo is clean
}

func TestProcessManager_Start(t *testing.T) {
	tests := []struct {
		name    string
		command []string
		wantErr bool
	}{
		{
			name:    "valid command",
			command: []string{"sleep", "0.1"},
			wantErr: false,
		},
		{
			name:    "invalid command",
			command: []string{"nonexistentcommand"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Command: tt.command,
				Logger:  logger.New(),
			}
			pm := NewProcessManager(cfg)

			err := pm.Start(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil {
				// Ensure process is cleaned up
				_ = pm.Stop(cfg)
			}
		})
	}
}

func TestProcessManager_Stop(t *testing.T) {
	tests := []struct {
		name         string
		command      []string
		gracefulStop bool
		stopTimeout  time.Duration
		waitForExit  bool
		wantErr      bool
	}{
		{
			name: "graceful stop - quick process",
			command: []string{"bash", "-c", `
				cleanup() {
					exit 0
				}
				trap cleanup SIGTERM
				sleep 5
			`},
			gracefulStop: true,
			stopTimeout:  time.Second,
			wantErr:      false,
		},
		{
			name:         "force stop - long running process",
			command:      []string{"sleep", "10"},
			gracefulStop: false,
			stopTimeout:  time.Second,
			wantErr:      false,
		},
		{
			name:         "stop already exited process",
			command:      []string{"sleep", "0.1"},
			gracefulStop: true,
			stopTimeout:  time.Second,
			waitForExit:  true,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Command:      tt.command,
				Logger:       logger.New(),
				GracefulStop: tt.gracefulStop,
				StopTimeout:  tt.stopTimeout,
			}
			pm := NewProcessManager(cfg)

			// Start the process
			if err := pm.Start(cfg); err != nil {
				t.Fatalf("Failed to start process: %v", err)
			}

			// Give it a moment to start and verify it's running
			time.Sleep(100 * time.Millisecond)

			if pm.cmd == nil || pm.cmd.Process == nil {
				t.Fatal("Process did not start properly")
			}

			// For the "already exited" test case, wait for process to exit
			if tt.waitForExit {
				select {
				case <-pm.doneChan:
					// Process exited naturally
				case <-time.After(2 * time.Second):
					t.Fatal("Process did not exit within timeout")
				}
			} else {
				// Verify process is running for non-waitForExit cases
				if err := pm.cmd.Process.Signal(syscall.Signal(0)); err != nil {
					t.Fatalf("Process is not running before stop: %v", err)
				}
			}

			// Stop the process
			err := pm.Stop(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Stop() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify process is actually stopped
			select {
			case <-pm.doneChan:
				// Process stopped successfully
			case <-time.After(2 * time.Second):
				t.Error("Process did not stop within timeout")
			}
		})
	}
}

func TestWatch_BasicScenarios(t *testing.T) {
	tests := []struct {
		name          string
		localCommits  []string
		remoteCommits []string
		compareResult git.CommitComparisonResult
		pullError     error
		compareError  error
		runOnStart    bool
		expectRestart bool
		wantErr       bool
	}{
		{
			name:          "no changes detected",
			localCommits:  []string{"abc123"},
			remoteCommits: []string{"abc123"},
			compareResult: git.CommitsEqual,
			runOnStart:    true,
			expectRestart: false,
			wantErr:       false,
		},
		{
			name:          "changes detected",
			localCommits:  []string{"abc123"},
			remoteCommits: []string{"def456"},
			compareResult: git.AIsAncestorOfB,
			runOnStart:    true,
			expectRestart: true,
			wantErr:       false,
		},
		{
			name:          "pull error",
			localCommits:  []string{"abc123"},
			remoteCommits: []string{"def456"},
			compareResult: git.AIsAncestorOfB,
			pullError:     fmt.Errorf("network error"),
			runOnStart:    true,
			expectRestart: false,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockRepo{
				localCommits:  tt.localCommits,
				remoteCommits: tt.remoteCommits,
				compareResult: tt.compareResult,
				pullError:     tt.pullError,
				compareError:  tt.compareError,
			}

			cfg := &config.Config{
				Command:      []string{"sleep", "0.1"},
				Logger:       logger.New(),
				RunOnStart:   tt.runOnStart,
				PollInterval: 100 * time.Millisecond,
				Verbose:      true,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()

			errChan := make(chan error)

			go func() {
				errChan <- Watch(cfg, WithRepository(mockRepo))
			}()

			select {
			case err := <-errChan:
				if (err != nil) != tt.wantErr {
					t.Errorf("Watch() error = %v, wantErr %v", err, tt.wantErr)
				}
			case <-ctx.Done():
				// Test timeout reached, this is expected
			}
		})
	}
}

func TestWatch_EdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		localCommits  []string
		remoteCommits []string
		compareResult git.CommitComparisonResult
		setupFunc     func(*config.Config)
		wantErr       bool
	}{
		{
			name:          "diverged branches",
			localCommits:  []string{"abc123"},
			remoteCommits: []string{"def456"},
			compareResult: git.CommitsDiverged,
			wantErr:       false,
		},
		{
			name:          "local ahead",
			localCommits:  []string{"def456"},
			remoteCommits: []string{"abc123"},
			compareResult: git.BIsAncestorOfA,
			wantErr:       false,
		},
		{
			name:          "invalid command",
			localCommits:  []string{"abc123"},
			remoteCommits: []string{"def456"},
			compareResult: git.AIsAncestorOfB,
			setupFunc: func(cfg *config.Config) {
				cfg.Command = []string{"nonexistentcommand"}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockRepo{
				localCommits:  tt.localCommits,
				remoteCommits: tt.remoteCommits,
				compareResult: tt.compareResult,
			}

			cfg := &config.Config{
				Command:      []string{"sleep", "0.1"},
				Logger:       logger.New(),
				RunOnStart:   true,
				PollInterval: 100 * time.Millisecond,
				Verbose:      true,
			}

			if tt.setupFunc != nil {
				tt.setupFunc(cfg)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()

			errChan := make(chan error)
			go func() {
				errChan <- Watch(cfg, WithRepository(mockRepo))
			}()

			select {
			case err := <-errChan:
				if (err != nil) != tt.wantErr {
					t.Errorf("Watch() error = %v, wantErr %v", err, tt.wantErr)
				}
			case <-ctx.Done():
				// Expected timeout
			}
		})
	}
}
