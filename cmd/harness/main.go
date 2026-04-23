package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jeremygatt/jg-mini-harness/internal/runner"
)

const Version = "0.3.0"

const repo = "https://github.com/jgatt493/jg-mini-harness.git"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: harness <command>\n\nCommands:\n  run       Run TDD test specs\n  update    Update harness to latest version\n  version   Print version\n")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version", "--version", "-v":
		fmt.Printf("jg-mini-harness v%s\n", Version)
		return
	case "update":
		runUpdate()
		return
	case "run":
		// existing run logic
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\nUsage: harness <command>\n\nCommands:\n  run       Run TDD test specs\n  update    Update harness to latest version\n  version   Print version\n", os.Args[1])
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

func runUpdate() {
	self, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding harness binary: %v\n", err)
		os.Exit(1)
	}
	self, _ = filepath.EvalSymlinks(self)

	tmpDir, err := os.MkdirTemp("", "harness-update-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tmpDir)

	fmt.Printf("Current version: v%s\n", Version)
	fmt.Println("Cloning latest...")

	clone := exec.Command("git", "clone", "--depth=1", repo, tmpDir)
	clone.Stdout = os.Stdout
	clone.Stderr = os.Stderr
	if err := clone.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error cloning repo: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Building...")
	buildPath := filepath.Join(tmpDir, "harness-new")
	build := exec.Command("go", "build", "-o", buildPath, "./cmd/harness")
	build.Dir = tmpDir
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error building: %v\n", err)
		os.Exit(1)
	}

	// Get new version before replacing
	out, _ := exec.Command(buildPath, "version").Output()
	newVersion := strings.TrimSpace(string(out))

	// Replace the current binary
	if err := os.Rename(buildPath, self); err != nil {
		// Rename fails across filesystems, fall back to copy
		data, err := os.ReadFile(buildPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading new binary: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(self, data, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing binary to %s: %v\n", self, err)
			os.Exit(1)
		}
	}

	fmt.Printf("Updated: %s\n", newVersion)
}
