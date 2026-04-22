package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jeremygatt/jg-mini-harness/internal/runner"
)

const Version = "0.2.0"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: harness <command>\n\nCommands:\n  run       Run TDD test specs\n  version   Print version\n")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version", "--version", "-v":
		fmt.Printf("jg-mini-harness v%s\n", Version)
		return
	case "run":
		// existing run logic
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\nUsage: harness <command>\n\nCommands:\n  run       Run TDD test specs\n  version   Print version\n", os.Args[1])
		os.Exit(1)
	}

	runCmd := flag.NewFlagSet("run", flag.ExitOnError)

	maxAttempts := runCmd.Int("max-attempts", 5, "Max retries per test")
	timeout := runCmd.Duration("timeout", 5*time.Minute, "Max time per test")
	projectDir := runCmd.String("project-dir", ".", "Working directory for claude and test commands")
	claudeCmd := runCmd.String("claude-cmd", "claude", "Path to claude CLI")
	retryFailed := runCmd.Bool("retry-failed", false, "Re-run tests with .status = fail")

	// Parse everything after "run"
	args := os.Args[2:]

	// Extract optional positional arg (test dir), default to ./TDD
	testDir := "./TDD"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		testDir = args[0]
		args = args[1:]
	}
	runCmd.Parse(args)

	cfg := runner.RunConfig{
		TestDir:     testDir,
		ProjectDir:  *projectDir,
		ClaudeCmd:   *claudeCmd,
		MaxAttempts: *maxAttempts,
		Timeout:     *timeout,
		RetryFailed: *retryFailed,
	}

	result := runner.Run(cfg)
	if result.Failed > 0 {
		os.Exit(1)
	}
}
