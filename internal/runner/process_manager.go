package runner

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/ship-digital/pull-watch/internal/config"
	"github.com/ship-digital/pull-watch/internal/logger"
)

// Processor defines the interface for process management
type Processor interface {
	Start() error
	Stop() error
	GetDoneChan() <-chan struct{}
	GetBackoff() time.Duration
	SetBackoff(d time.Duration)
	GetLastLogTime() time.Time
	SetLastLogTime(t time.Time)
	GetLogger() *logger.Logger
}

var (
	// initialBackoff is the initial backoff time
	initialBackoff = 5 * time.Second
	// maxBackoff is the maximum backoff time
	maxBackoff = 5 * time.Minute
)

var _ Processor = &ProcessManager{}

type ProcessManager struct {
	mu          sync.Mutex
	cfg         *config.Config
	cmd         *exec.Cmd
	doneChan    chan struct{}
	stopped     bool
	logger      *logger.Logger
	lastLogTime time.Time
	backoff     time.Duration
}

func New(cfg *config.Config) *ProcessManager {
	return &ProcessManager{
		cfg:    cfg,
		logger: cfg.Logger,
	}
}

func (pm *ProcessManager) Start() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Reset backoff when starting new process
	pm.backoff = 0
	pm.lastLogTime = time.Time{}

	// Make sure any previous process is fully cleaned up
	if pm.cmd != nil {
		if err := pm.forceStop(); err != nil {
			pm.logger.MultiColor(logger.QuietLevel,
				logger.ErrorSegment("Failed to clean up previous process: "),
				logger.HighlightSegment(fmt.Sprintf("%v", err)),
			)
		}
		pm.cmd = nil
	}

	pm.stopped = false
	pm.doneChan = make(chan struct{})
	pm.cmd = exec.Command(pm.cfg.Command[0], pm.cfg.Command[1:]...)
	pm.cmd.Stdout = os.Stdout
	pm.cmd.Stderr = os.Stderr
	pm.cmd.Stdin = os.Stdin

	setProcessGroup(pm.cmd)

	if err := pm.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	go func() {
		pm.cmd.Wait()
		pm.mu.Lock()
		close(pm.doneChan)
		pm.cmd = nil
		pm.mu.Unlock()
	}()

	return nil
}

func (pm *ProcessManager) Stop() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.cmd == nil || pm.cmd.Process == nil {
		return nil
	}

	pm.stopped = true

	if pm.cfg.GracefulStop {
		return pm.gracefulStop(pm.cfg.StopTimeout)
	}
	return pm.forceStop()
}

func (pm *ProcessManager) GetDoneChan() <-chan struct{} {
	return pm.doneChan
}

func (pm *ProcessManager) GetBackoff() time.Duration {
	return pm.backoff
}

func (pm *ProcessManager) SetBackoff(d time.Duration) {
	pm.backoff = d
}

func (pm *ProcessManager) GetLastLogTime() time.Time {
	return pm.lastLogTime
}

func (pm *ProcessManager) SetLastLogTime(t time.Time) {
	pm.lastLogTime = t
}

func (pm *ProcessManager) GetLogger() *logger.Logger {
	return pm.logger
}

func (pm *ProcessManager) gracefulStop(timeout time.Duration) error {
	if err := terminateProcess(pm.cmd); err != nil {
		return pm.forceStop()
	}

	select {
	case <-pm.doneChan:
		return nil
	case <-time.After(timeout):
		return pm.forceStop()
	}
}

func (pm *ProcessManager) forceStop() error {
	return killProcess(pm.cmd)
}
