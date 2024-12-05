package main

import (
	"fmt"
	"os"

	"github.com/ship-digital/pull-watch/internal/config"
	"github.com/ship-digital/pull-watch/internal/runner"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("pull-watch version %s\n", version)
		return
	}

	cfg, err := config.Parse()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		config.PrintUsage()
		os.Exit(1)
	}

	if err := runner.Watch(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
