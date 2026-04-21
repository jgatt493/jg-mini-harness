package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jeremygatt/jg-mini-harness/internal/runner"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "run" {
		fmt.Fprintf(os.Stderr, "Usage: harness run [test-dir] [flags] (default: ./TDD)\n")
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
