package config

import (
	"time"

	"github.com/ship-digital/pull-watch/internal/logger"
)

type Config struct {
	PollInterval time.Duration
	Command      []string
	GitDir       string
	Verbose      bool
	GracefulStop bool
	StopTimeout  time.Duration
	Logger       *logger.Logger
}

func GetUsageString() string {
	return `Usage: pull-watch [options] -- <command>

Options:
  -interval duration     Poll interval (e.g. 15s, 1m) (default 15s)
  -git-dir string       Git repository directory (default ".")
  -verbose              Enable verbose logging
  -graceful             Try graceful stop before force kill
  -stop-timeout duration Timeout for graceful stop before force kill (default 5s)`
}
