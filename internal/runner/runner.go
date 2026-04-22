package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jeremygatt/jg-mini-harness/internal/executor"
	"github.com/jeremygatt/jg-mini-harness/internal/reporter"
)

// TestSpec represents a single test directory with its spec and command.
type TestSpec struct {
	Name    string // directory name
	Dir     string // full path to test directory
	Spec    string // contents of spec.md
	TestCmd string // first line of test_cmd
}

// DiscoverTests scans testDir for valid test subdirectories.
// A valid test dir contains both spec.md and test_cmd.
// Returns tests sorted alphabetically by name.
func DiscoverTests(testDir string) ([]TestSpec, error) {
	entries, err := os.ReadDir(testDir)
	if err != nil {
		return nil, fmt.Errorf("reading test directory: %w", err)
	}

	var tests []TestSpec
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dir := filepath.Join(testDir, entry.Name())
		specPath := filepath.Join(dir, "spec.md")
		cmdPath := filepath.Join(dir, "test_cmd")

		specData, err := os.ReadFile(specPath)
		if err != nil {
			continue // skip dirs without spec.md
		}

		cmdData, err := os.ReadFile(cmdPath)
		if err != nil {
			continue // skip dirs without test_cmd
		}

		// Read first line of test_cmd
		cmd := string(cmdData)
		if idx := strings.IndexByte(cmd, '\n'); idx != -1 {
			cmd = cmd[:idx]
		}
		cmd = strings.TrimSpace(cmd)

		tests = append(tests, TestSpec{
			Name:    entry.Name(),
			Dir:     dir,
			Spec:    string(specData),
			TestCmd: cmd,
		})
	}

	sort.Slice(tests, func(i, j int) bool {
		return tests[i].Name < tests[j].Name
	})

	return tests, nil
}

type Status string

const (
	StatusPending    Status = ""
	StatusPass       Status = "pass"
	StatusFail       Status = "fail"
	StatusInProgress Status = "in_progress"
)

func ReadStatus(testDir string) (Status, error) {
	data, err := os.ReadFile(filepath.Join(testDir, ".status"))
	if os.IsNotExist(err) {
		return StatusPending, nil
	}
	if err != nil {
		return "", fmt.Errorf("reading status: %w", err)
	}
	return Status(strings.TrimSpace(string(data))), nil
}

func WriteStatus(testDir string, status Status) error {
	return os.WriteFile(filepath.Join(testDir, ".status"), []byte(string(status)+"\n"), 0o644)
}

func ShouldRun(status Status, retryFailed bool) bool {
	switch status {
	case StatusPending:
		return true
	case StatusFail:
		return retryFailed
	default:
		return false
	}
}

type RunConfig struct {
	TestDir     string
	ProjectDir  string
	ClaudeCmd   string
	MaxAttempts int
	Timeout     time.Duration
	RetryFailed bool
}

type RunResult struct {
	Passed int
	Failed int
}

func Run(cfg RunConfig) RunResult {
	fmt.Printf("Scanning %s for tests...\n", cfg.TestDir)

	tests, err := DiscoverTests(cfg.TestDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error discovering tests: %v\n", err)
		return RunResult{}
	}

	// Tally statuses
	var pending, passed, failed, inProgress int
	for _, test := range tests {
		status, _ := ReadStatus(test.Dir)
		switch status {
		case StatusPending:
			pending++
		case StatusPass:
			passed++
		case StatusFail:
			if cfg.RetryFailed {
				pending++
			} else {
				failed++
			}
		case StatusInProgress:
			inProgress++
		}
	}

	fmt.Printf("Found %d tests: %d pending, %d passed, %d failed", len(tests), pending, passed, failed)
	if inProgress > 0 {
		fmt.Printf(", %d in_progress (stale)", inProgress)
	}
	fmt.Println()

	if pending == 0 {
		fmt.Println("Nothing to run.")
		return RunResult{}
	}

	fmt.Printf("\nStarting test run (%d to process)...\n\n", pending)

	var result RunResult
	index := 0

	for _, test := range tests {
		status, err := ReadStatus(test.Dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading status for %s: %v\n", test.Name, err)
			continue
		}

		if !ShouldRun(status, cfg.RetryFailed) {
			continue
		}

		index++
		fmt.Printf("[%d/%d] %s... ", index, pending, test.Name)
		WriteStatus(test.Dir, StatusInProgress)

		passed, attempts, duration := runSingleTest(cfg, test)

		if passed {
			WriteStatus(test.Dir, StatusPass)
			reporter.DeleteErrorReport(test.Dir)
			fmt.Printf("PASS (%d attempts, %s)\n", attempts, duration.Round(time.Second))
			result.Passed++
		} else {
			WriteStatus(test.Dir, StatusFail)
			fmt.Printf("FAIL (%d attempts, %s) → error.md written\n", attempts, duration.Round(time.Second))
			result.Failed++
		}
	}

	fmt.Println(reporter.FormatSummary(result.Passed, result.Failed))
	return result
}

func runSingleTest(cfg RunConfig, test TestSpec) (passed bool, attempts int, duration time.Duration) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	var history []reporter.AttemptRecord
	var lastTestResult executor.ExecResult
	var lastClaudeResult executor.ExecResult
	prevOutput := ""

	for attempts = 1; attempts <= cfg.MaxAttempts; attempts++ {
		elapsed := time.Since(start)
		if elapsed >= cfg.Timeout {
			break
		}

		prompt := executor.BuildPrompt(test.Spec, test.TestCmd, prevOutput)
		lastClaudeResult = executor.RunClaude(ctx, cfg.ClaudeCmd, prompt, cfg.ProjectDir)

		lastTestResult = executor.RunTestCmd(ctx, test.TestCmd, cfg.ProjectDir)

		if lastTestResult.ExitCode == 0 {
			return true, attempts, time.Since(start)
		}

		history = append(history, reporter.AttemptRecord{
			Number:     attempts,
			TestOutput: lastTestResult.Output,
		})

		prevOutput = lastTestResult.Output
	}

	// Exhausted retries — write error report
	report := reporter.ErrorReport{
		TestName:     test.Name,
		Attempts:     attempts - 1,
		Duration:     time.Since(start),
		LastExitCode: lastTestResult.ExitCode,
		Spec:         test.Spec,
		FinalOutput:  lastTestResult.Output,
		ClaudeOutput: lastClaudeResult.Output,
		History:      history,
	}
	reporter.WriteErrorReport(test.Dir, report)

	return false, attempts - 1, time.Since(start)
}
