// Orbit — main entry point.
// Keeps this file deliberately thin: parse build-time vars, wire up CLI, execute.
package main

import (
	"github.com/f9-o/orbit/internal/cli"
	"github.com/f9-o/orbit/internal/cli/commands"
)

// Build-time variables injected via:
//
//	go build -ldflags "-X main.version=v1.0.0 -X main.commit=abc1234 -X main.buildDate=2025-01-01"
var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

func main() {
	// Propagate build metadata to the version command
	commands.Version = version
	commands.Commit = commit
	commands.BuildDate = buildDate

	cli.Execute()
}
