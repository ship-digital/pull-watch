package config

import (
	"time"

	"github.com/ship-digital/pull-watch/internal/logger"
)

type Config struct {
	PollInterval  time.Duration
	Command       []string
	GitDir        string
	Verbose       bool
	GracefulStop  bool
	StopTimeout   time.Duration
	Logger        *logger.Logger
	RunOnStart    bool
	ShowTimestamp bool
}
