package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/cli"
	"github.com/ship-digital/pull-watch/internal/config"
	"github.com/ship-digital/pull-watch/internal/runner"
)

var version = "dev"

type MainCommand struct {
	ui cli.Ui

	// Flag values
	pollInterval time.Duration
	gitDir       string
	verbose      bool
	graceful     bool
	stopTimeout  time.Duration
}

func (c *MainCommand) Run(args []string) int {
	if len(args) == 0 {
		c.ui.Error("Error: no arguments provided")
		c.ui.Output(c.Help())
		return 1
	}

	// Find the index of "--" separator
	cmdIndex := -1
	for i, arg := range args {
		if arg == "--" {
			cmdIndex = i
			break
		}
	}

	if cmdIndex == -1 {
		c.ui.Error("Error: command separator '--' not found")
		c.ui.Output(c.Help())
		return 1
	}

	// Parse flags first
	flags := flag.NewFlagSet("pull-watch", flag.ContinueOnError)
	flags.SetOutput(io.Discard) // Suppress flag errors
	flags.DurationVar(&c.pollInterval, "interval", 15*time.Second, "Poll interval (e.g. 15s, 1m)")
	flags.StringVar(&c.gitDir, "git-dir", ".", "Git repository directory")
	flags.BoolVar(&c.verbose, "verbose", false, "Enable verbose logging")
	flags.BoolVar(&c.graceful, "graceful", false, "Try graceful stop before force kill")
	flags.DurationVar(&c.stopTimeout, "stop-timeout", 5*time.Second, "Timeout for graceful stop before force kill")

	// Parse only the flags before "--"
	if err := flags.Parse(args[:cmdIndex]); err != nil {
		return 1
	}

	// Get command and its args after "--"
	cmdArgs := args[cmdIndex+1:]
	if len(cmdArgs) == 0 {
		c.ui.Error("Error: no command provided")
		c.ui.Output(c.Help())
		return 1
	}

	cfg := &config.Config{
		PollInterval: c.pollInterval,
		Command:      cmdArgs,
		GitDir:       c.gitDir,
		Verbose:      c.verbose,
		GracefulStop: c.graceful,
		StopTimeout:  c.stopTimeout,
	}

	if err := runner.Watch(cfg); err != nil {
		c.ui.Error(fmt.Sprintf("Error: %v", err))
		return 1
	}

	return 0
}

func (c *MainCommand) Help() string {
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	flags.DurationVar(&c.pollInterval, "interval", 15*time.Second, "Poll interval (e.g. 15s, 1m)")
	flags.StringVar(&c.gitDir, "git-dir", ".", "Git repository directory")
	flags.BoolVar(&c.verbose, "verbose", false, "Enable verbose logging")
	flags.BoolVar(&c.graceful, "graceful", false, "Try graceful stop before force kill")
	flags.DurationVar(&c.stopTimeout, "stop-timeout", 5*time.Second, "Timeout for graceful stop before force kill")

	var buf strings.Builder
	flags.SetOutput(&buf)
	flags.PrintDefaults()

	return fmt.Sprintf(`
Usage: pull-watch [options] -- <command>

  Watch git repository for changes and run commands.

Options:
%s`, buf.String())
}

func (c *MainCommand) Synopsis() string {
	return "Watch git repository for changes and run commands"
}

type VersionCommand struct {
	Version string
	ui      cli.Ui
}

func (c *VersionCommand) Run(_ []string) int {
	c.ui.Output(fmt.Sprintf("pull-watch version %s", c.Version))
	return 0
}

func (c *VersionCommand) Help() string {
	return "Prints the pull-watch version"
}

func (c *VersionCommand) Synopsis() string {
	return "Prints the pull-watch version"
}

func main() {
	ui := &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      os.Stdout,
		ErrorWriter: os.Stderr,
	}

	// Create the command directly
	cmd := &MainCommand{ui: ui}

	// If first argument is "version", handle it specially
	if len(os.Args) > 1 && os.Args[1] == "version" {
		versionCmd := &VersionCommand{
			Version: version,
			ui:      ui,
		}
		os.Exit(versionCmd.Run(nil))
	}

	// Pass all arguments to the main command
	exitStatus := cmd.Run(os.Args[1:])
	os.Exit(exitStatus)
}
