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
	localCommits   []string
	remoteCommits  []string
	pullError      error
	compareResult  git.CommitComparisonResult
	compareError   error
	currentIndex   int
	compareHandler func(local, remote string) git.CommitComparisonResult
}

func (m *MockRepo) GetLatestCommit(ctx context.Context) (string, error) {
	return m.localCommits[0], nil
}

func (m *MockRepo) GetRemoteCommit(ctx context.Context) (string, error) {
	return m.remoteCommits[m.currentIndex], nil
}

func (m *MockRepo) Pull(ctx context.Context) (string, error) {
	if m.pullError != nil {
		return "", m.pullError
	}
	m.localCommits[0] = m.remoteCommits[m.currentIndex]
	return "Changes pulled successfully", nil
}

func (m *MockRepo) HandleCommitComparison(ctx context.Context, local, remote string) (git.CommitComparisonResult, error) {
	if m.compareError != nil {
		return git.UnknownCommitComparisonResult, m.compareError
	}
	if m.compareHandler != nil {
		return m.compareHandler(local, remote), nil
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

// TestProcessManager wraps ProcessManager for testing
type TestProcessManager struct {
	pm         *ProcessManager
	executions chan struct{}
}

func NewTestProcessManager(cfg *config.Config, executions chan struct{}) *TestProcessManager {
	return &TestProcessManager{
		pm:         New(cfg),
		executions: executions,
	}
}

// Start implements the same interface as ProcessManager
func (pm *TestProcessManager) Start() error {
	err := pm.pm.Start()
	if err == nil {
		// Signal execution
		select {
		case pm.executions <- struct{}{}:
		default:
		}
	}
	return err
}

// Stop delegates to the underlying ProcessManager
func (pm *TestProcessManager) Stop() error {
	return pm.pm.Stop()
}

// GetDoneChan implements Processor interface
func (pm *TestProcessManager) GetDoneChan() <-chan struct{} {
	return pm.pm.GetDoneChan()
}

// GetBackoff implements Processor interface
func (pm *TestProcessManager) GetBackoff() time.Duration {
	return pm.pm.GetBackoff()
}

// SetBackoff implements Processor interface
func (pm *TestProcessManager) SetBackoff(d time.Duration) {
	pm.pm.SetBackoff(d)
}

// GetLastLogTime implements Processor interface
func (pm *TestProcessManager) GetLastLogTime() time.Time {
	return pm.pm.GetLastLogTime()
}

// SetLastLogTime implements Processor interface
func (pm *TestProcessManager) SetLastLogTime(t time.Time) {
	pm.pm.SetLastLogTime(t)
}

// GetLogger implements Processor interface
func (pm *TestProcessManager) GetLogger() *logger.Logger {
	return pm.pm.GetLogger()
}

func (pm *TestProcessManager) handleCommitComparison(ctx context.Context, cfg *config.Config, repo git.Repository, local, remote string) (git.CommitComparisonResult, error) {
	result, err := repo.HandleCommitComparison(ctx, local, remote)
	if err != nil {
		return git.UnknownCommitComparisonResult, err
	}

	if cfg.Verbose {
		pm.pm.logger.MultiColor(
			logger.InfoSegment("Local commit: "),
			logger.HighlightSegment(local),
		)
		pm.pm.logger.MultiColor(
			logger.InfoSegment("Remote commit: "),
			logger.HighlightSegment(remote),
		)
	}

	switch result {
	case git.CommitsEqual:
		if cfg.Verbose {
			pm.pm.logger.Info("Local commit and remote commit are the same: not pulling.")
		}
	case git.AIsAncestorOfB:
		if cfg.Verbose {
			pm.pm.logger.Info("Local commit is behind remote commit, pulling changes...")
		}
		if _, err := repo.Pull(ctx); err != nil {
			return result, fmt.Errorf("failed to pull changes: %w", err)
		}
	case git.BIsAncestorOfA:
		if cfg.Verbose {
			pm.pm.logger.Info("Local commit is ahead of remote commit, not pulling.")
		}
	case git.CommitsDiverged:
		if cfg.Verbose {
			pm.pm.logger.Info("Local and remote commits have diverged, not pulling.")
		}
	}

	return result, nil
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
			pm := New(cfg)

			err := pm.Start()
			if (err != nil) != tt.wantErr {
				t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil {
				// Ensure process is cleaned up
				_ = pm.Stop()
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
			pm := New(cfg)

			// Start the process
			if err := pm.Start(); err != nil {
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
			err := pm.Stop()
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
				currentIndex:  0,
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
				errChan <- Run(cfg, WithRepository(mockRepo))
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
				errChan <- Run(cfg, WithRepository(mockRepo))
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

func TestWatch_LocalAheadThenBehind(t *testing.T) {
	mockRepo := &MockRepo{
		localCommits:  []string{"def456"},                     // Local starts at def456
		remoteCommits: []string{"abc123", "def456", "ghi789"}, // Remote moves from abc123 -> def456 -> ghi789
		currentIndex:  0,                                      // Start at abc123
		compareHandler: func(local, remote string) git.CommitComparisonResult {
			switch {
			case local == remote:
				return git.CommitsEqual
			case local == "def456" && remote == "ghi789":
				return git.AIsAncestorOfB
			default:
				return git.BIsAncestorOfA
			}
		},
	}

	executions := make(chan struct{}, 10)
	cfg := &config.Config{
		Command:      []string{"sleep", "0.1"},
		Logger:       logger.New(),
		PollInterval: 50 * time.Millisecond,
		Verbose:      true,
		RunOnStart:   false,
	}

	testPM := NewTestProcessManager(cfg, executions)

	// Run Watch in a goroutine
	errChan := make(chan error, 1)
	watchDone := make(chan struct{})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	go func() {
		err := Run(cfg, WithRepository(mockRepo), WithProcessManager(testPM))
		errChan <- err
		close(watchDone)
	}()

	// Wait for initial check
	time.Sleep(200 * time.Millisecond)

	// Drain any startup executions
	drainExecutions(executions)

	// Simulate remote moving ahead
	mockRepo.currentIndex = 2 // Move to ghi789

	// Wait for execution
	select {
	case <-executions:
		// Success - command executed after remote moved ahead
		cancel() // Signal Watch to stop
		return
	case <-time.After(500 * time.Millisecond):
		t.Error("Command was not executed after remote moved ahead")
	case err := <-errChan:
		t.Errorf("Watch returned unexpectedly with error: %v", err)
	case <-ctx.Done():
		t.Error("Test timed out")
	}
}

// Helper function to drain the executions channel
func drainExecutions(ch chan struct{}) {
	for {
		select {
		case <-ch:
			continue
		default:
			return
		}
	}
}
