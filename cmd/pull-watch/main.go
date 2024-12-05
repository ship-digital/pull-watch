package main

import (
	"fmt"
	"os"

	"github.com/hashicorp/cli"
	"github.com/ship-digital/pull-watch/internal/config"
	"github.com/ship-digital/pull-watch/internal/runner"
)

var version = "dev"

type MainCommand struct {
	ui cli.Ui
}

func (c *MainCommand) Run(args []string) int {
	cfg, err := config.Parse()
	if err != nil {
		c.ui.Error(fmt.Sprintf("Error: %v", err))
		config.PrintUsage()
		return 1
	}

	if err := runner.Watch(cfg); err != nil {
		c.ui.Error(fmt.Sprintf("Error: %v", err))
		return 1
	}

	return 0
}

func (c *MainCommand) Help() string {
	return config.GetUsageString()
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

	c := cli.NewCLI("pull-watch", version)
	c.Args = os.Args[1:]

	c.Commands = map[string]cli.CommandFactory{
		"": func() (cli.Command, error) {
			return &MainCommand{ui: ui}, nil
		},
		"version": func() (cli.Command, error) {
			return &VersionCommand{
				Version: version,
				ui:      ui,
			}, nil
		},
	}

	exitStatus, err := c.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	os.Exit(exitStatus)
}
