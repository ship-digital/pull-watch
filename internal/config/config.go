package config

import (
	"flag"
	"fmt"
	"os"
	"time"
)

type Config struct {
	PollInterval time.Duration
	Command      []string
	GitDir       string
	Verbose      bool
	GracefulStop bool
	StopTimeout  time.Duration
}

func Parse() (*Config, error) {
	cfg := &Config{}

	flag.DurationVar(&cfg.PollInterval, "interval", 15*time.Second, "Poll interval (e.g. 15s, 1m)")
	flag.StringVar(&cfg.GitDir, "git-dir", ".", "Git repository directory")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "Enable verbose logging")
	flag.BoolVar(&cfg.GracefulStop, "graceful", false, "Try graceful stop before force kill")
	flag.DurationVar(&cfg.StopTimeout, "stop-timeout", 5*time.Second, "Timeout for graceful stop before force kill")

	// Custom usage message
	flag.Usage = PrintUsage
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		return nil, fmt.Errorf("no command provided")
	}

	cfg.Command = args
	return cfg, nil
}

func PrintUsage() {
	fmt.Fprintf(os.Stderr, `Usage: pull-watch [options] -- <command>

Options:
`)
	flag.PrintDefaults()
}
