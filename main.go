package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/hashicorp/cli"
	"github.com/ship-digital/pull-watch/internal/config"
	"github.com/ship-digital/pull-watch/internal/logger"
	"github.com/ship-digital/pull-watch/internal/runner"
)

var version = "dev"

type MainCommand struct {
	ui  cli.Ui
	log *logger.Logger

	// Flag values
	pollInterval  time.Duration
	gitDir        string
	quiet         bool
	verbose       bool
	graceful      bool
	stopTimeout   time.Duration
	runOnStart    bool
	showTimestamp bool
	showVersion   bool
	noRestart     bool
}

func (c *MainCommand) setupFlags(flags *flag.FlagSet) {
	flags.DurationVar(&c.pollInterval, "interval", 15*time.Second, "Poll interval (e.g. 15s, 1m)")
	flags.StringVar(&c.gitDir, "git-dir", ".", "Git repository directory")
	flags.BoolVar(&c.verbose, "verbose", false, "Enable verbose logging")
	flags.BoolVar(&c.quiet, "quiet", false, "Show only errors and warnings")
	flags.BoolVar(&c.graceful, "graceful", false, "Try graceful stop before force kill")
	flags.DurationVar(&c.stopTimeout, "stop-timeout", 5*time.Second, "Timeout for graceful stop before force kill")
	flags.BoolVar(&c.runOnStart, "run-on-start", false, "Run command on startup regardless of git state")
	flags.BoolVar(&c.showTimestamp, "timestamp", false, "Show timestamps in logs")
	flags.BoolVar(&c.showVersion, "version", false, "Show version information")
	flags.BoolVar(&c.noRestart, "no-restart", false, "Pull changes without restarting the command, useful if the command has a built-in auto-reload feature")
}

func (c *MainCommand) Run(args []string) int {
	// Show help if no args or help flags
	if len(args) == 0 || (len(args) > 0 && (args[0] == "-h" || args[0] == "--help")) {
		c.ui.Output(c.Help())
		return 0
	}

	// Find the index of "--" separator
	cmdIndex := -1
	for i, arg := range args {
		if arg == "--" {
			cmdIndex = i
			break
		}
	}

	// Parse flags first
	flags := flag.NewFlagSet("pull-watch", flag.ContinueOnError)
	flags.SetOutput(io.Discard) // Suppress flag errors
	c.setupFlags(flags)
	// Parse flags before "--" or all flags if no "--" found
	flagArgs := args
	if cmdIndex != -1 {
		flagArgs = args[:cmdIndex]
	}
	if err := flags.Parse(flagArgs); err != nil {
		return 1
	}

	// Handle version flag
	if c.showVersion {
		versionCmd := &VersionCommand{
			Version: version,
			ui:      c.ui,
		}
		return versionCmd.Run(nil)
	}

	if cmdIndex == -1 {
		c.ui.Error("Error: command separator '--' not found")
		c.ui.Output(c.Help())
		return 1
	}

	// Get command and its args after "--"
	cmdArgs := args[cmdIndex+1:]
	if len(cmdArgs) == 0 {
		c.ui.Error("Error: no command provided")
		c.ui.Output(c.Help())
		return 1
	}

	// quietVerbose indicates that the user passed in both flags
	quietVerbose := false

	// Determine log level
	var logLevel logger.LogLevel
	switch {
	case c.verbose && c.quiet:
		quietVerbose = true
		logLevel = logger.VerboseLevel
	case c.verbose:
		logLevel = logger.VerboseLevel
	case c.quiet:
		logLevel = logger.QuietLevel
	default:
		logLevel = logger.DefaultLevel
	}

	// Initialize logger with options
	var opts []logger.Option
	if c.showTimestamp {
		opts = append(opts, logger.WithTimestamp())
	}
	opts = append(opts, logger.WithLogLevel(logLevel))

	c.log = logger.New(opts...)

	cfg := &config.Config{
		PollInterval:  c.pollInterval,
		Command:       cmdArgs,
		GitDir:        c.gitDir,
		LogLevel:      logLevel,
		GracefulStop:  c.graceful,
		StopTimeout:   c.stopTimeout,
		Logger:        c.log,
		RunOnStart:    c.runOnStart,
		ShowTimestamp: c.showTimestamp,
		NoRestart:     c.noRestart,
	}

	if quietVerbose {
		c.log.MultiColor(logger.QuietLevel,
			logger.ErrorSegment("Warning: "),
			logger.InfoSegment("both "),
			logger.HighlightSegment("-verbose"),
			logger.InfoSegment(" and "),
			logger.HighlightSegment("-quiet"),
			logger.InfoSegment(" flags set. Only "),
			logger.HighlightSegment("-verbose"),
			logger.InfoSegment(" considered!"),
		)
	}

	if err := runner.Run(cfg); err != nil {
		c.ui.Error(fmt.Sprintf("Error: %v", err))
		return 1
	}

	return 0
}

func (c *MainCommand) Help() string {
	flags := flag.NewFlagSet("", flag.ContinueOnError)
	c.setupFlags(flags)

	var buf strings.Builder
	flags.SetOutput(&buf)
	flags.PrintDefaults()

	return fmt.Sprintf(`
Usage: pull-watch [options] -- <command>

 Watch git repository for remote changes and run commands.

 It's like: 'git pull && <command>' but with polling and automatic process management.

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
	if info, ok := debug.ReadBuildInfo(); ok {
		version = info.Main.Version
	}

	c.ui.Output(fmt.Sprintf("pull-watch version (%s) %s", c.Version, version))
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

	// Handle version subcommand
	if len(os.Args) > 1 && os.Args[1] == "version" {
		versionCmd := &VersionCommand{
			Version: version,
			ui:      ui,
		}
		os.Exit(versionCmd.Run(nil))
	}

	// Create the command directly
	cmd := &MainCommand{ui: ui}
	exitStatus := cmd.Run(os.Args[1:])
	os.Exit(exitStatus)
}
