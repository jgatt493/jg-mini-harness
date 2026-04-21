# Mini TDD Harness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go CLI binary that scans a directory of test specs, spawns Claude Code sessions to implement each one, verifies via test commands, and reports results.

**Architecture:** Single Go binary with three internal packages — runner (discovery + loop), executor (subprocess management), reporter (error.md + terminal output). CLI entry point parses flags and delegates to runner.

**Tech Stack:** Go stdlib only (os/exec, filepath, flag, time, fmt, os, strings)

**Spec:** `docs/superpowers/specs/2026-04-21-mini-tdd-harness-design.md`

---

## File Map

| File | Responsibility |
|------|---------------|
| `go.mod` | Module definition |
| `cmd/harness/main.go` | CLI entry point, flag parsing, calls runner |
| `cmd/harness/main_test.go` | Integration tests for CLI |
| `internal/runner/runner.go` | Test discovery, status management, core retry loop |
| `internal/runner/runner_test.go` | Unit tests for runner |
| `internal/executor/executor.go` | Shell out to claude CLI and test commands, capture output |
| `internal/executor/executor_test.go` | Unit tests for executor |
| `internal/reporter/reporter.go` | Write error.md, format terminal output, print summary |
| `internal/reporter/reporter_test.go` | Unit tests for reporter |

---

### Task 1: Project Scaffolding

**Files:**
- Create: `go.mod`
- Create: `cmd/harness/main.go`

- [ ] **Step 1: Initialize Go module**

```bash
cd /Users/jeremygatt/Projects/jg-mini-harness && go mod init github.com/jeremygatt/jg-mini-harness
```

- [ ] **Step 2: Create minimal main.go that prints usage**

Create `cmd/harness/main.go`:

```go
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "run" {
		fmt.Fprintf(os.Stderr, "Usage: harness run <test-dir> [flags]\n")
		os.Exit(1)
	}
	fmt.Println("harness: not yet implemented")
}
```

- [ ] **Step 3: Verify it compiles and runs**

```bash
go build -o harness ./cmd/harness && ./harness
```

Expected: prints usage to stderr, exits 1.

```bash
./harness run ./tests
```

Expected: prints "harness: not yet implemented", exits 0.

- [ ] **Step 4: Commit**

```bash
git add go.mod cmd/harness/main.go
git commit -m "feat: scaffold Go project with minimal main"
```

---

### Task 2: Test Discovery (Runner)

**Files:**
- Create: `internal/runner/runner.go`
- Create: `internal/runner/runner_test.go`

- [ ] **Step 1: Write failing test for test discovery**

Create `internal/runner/runner_test.go`:

```go
package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create two valid test dirs
	for _, name := range []string{"alpha", "beta"} {
		testDir := filepath.Join(dir, name)
		os.MkdirAll(testDir, 0o755)
		os.WriteFile(filepath.Join(testDir, "spec.md"), []byte("build something"), 0o644)
		os.WriteFile(filepath.Join(testDir, "test_cmd"), []byte("echo ok"), 0o644)
	}

	// Create an invalid dir (missing spec.md)
	invalidDir := filepath.Join(dir, "gamma")
	os.MkdirAll(invalidDir, 0o755)
	os.WriteFile(filepath.Join(invalidDir, "test_cmd"), []byte("echo ok"), 0o644)

	return dir
}

func TestDiscoverTests(t *testing.T) {
	dir := setupTestDir(t)

	tests, err := DiscoverTests(dir)
	if err != nil {
		t.Fatalf("DiscoverTests failed: %v", err)
	}

	if len(tests) != 2 {
		t.Fatalf("expected 2 tests, got %d", len(tests))
	}

	if tests[0].Name != "alpha" || tests[1].Name != "beta" {
		t.Errorf("expected alpha, beta; got %s, %s", tests[0].Name, tests[1].Name)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/runner/ -v -run TestDiscoverTests
```

Expected: FAIL — `DiscoverTests` not defined.

- [ ] **Step 3: Implement DiscoverTests**

Create `internal/runner/runner.go`:

```go
package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/runner/ -v -run TestDiscoverTests
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/runner/runner.go internal/runner/runner_test.go
git commit -m "feat: add test discovery with directory scanning"
```

---

### Task 3: Status Management (Runner)

**Files:**
- Modify: `internal/runner/runner.go`
- Modify: `internal/runner/runner_test.go`

- [ ] **Step 1: Write failing tests for status read/write**

Append to `internal/runner/runner_test.go`:

```go
func TestReadStatus(t *testing.T) {
	dir := t.TempDir()

	// No .status file → pending
	status, err := ReadStatus(dir)
	if err != nil {
		t.Fatal(err)
	}
	if status != StatusPending {
		t.Errorf("expected pending, got %s", status)
	}

	// Write pass status
	if err := WriteStatus(dir, StatusPass); err != nil {
		t.Fatal(err)
	}
	status, err = ReadStatus(dir)
	if err != nil {
		t.Fatal(err)
	}
	if status != StatusPass {
		t.Errorf("expected pass, got %s", status)
	}
}

func TestShouldRun(t *testing.T) {
	tests := []struct {
		status     Status
		retryFail  bool
		shouldRun  bool
	}{
		{StatusPending, false, true},
		{StatusPass, false, false},
		{StatusFail, false, false},
		{StatusFail, true, true},
		{StatusInProgress, false, false},
	}
	for _, tt := range tests {
		got := ShouldRun(tt.status, tt.retryFail)
		if got != tt.shouldRun {
			t.Errorf("ShouldRun(%s, retryFailed=%v) = %v, want %v",
				tt.status, tt.retryFail, got, tt.shouldRun)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/runner/ -v -run "TestReadStatus|TestShouldRun"
```

Expected: FAIL — types and functions not defined.

- [ ] **Step 3: Implement status management**

Add to `internal/runner/runner.go`:

```go
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
```

Add `"strings"` to the import block.

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/runner/ -v -run "TestReadStatus|TestShouldRun"
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/runner/runner.go internal/runner/runner_test.go
git commit -m "feat: add status file read/write and ShouldRun logic"
```

---

### Task 4: Executor — Claude Invocation

**Files:**
- Create: `internal/executor/executor.go`
- Create: `internal/executor/executor_test.go`

- [ ] **Step 1: Write failing test for prompt building**

Create `internal/executor/executor_test.go`:

```go
package executor

import (
	"strings"
	"testing"
)

func TestBuildPrompt_FirstAttempt(t *testing.T) {
	prompt := BuildPrompt("Build a login system", "npm test", "")

	if !strings.Contains(prompt, "Build a login system") {
		t.Error("prompt missing spec")
	}
	if !strings.Contains(prompt, "npm test") {
		t.Error("prompt missing test command")
	}
	if strings.Contains(prompt, "previous attempt") {
		t.Error("first attempt should not mention previous attempt")
	}
}

func TestBuildPrompt_Retry(t *testing.T) {
	prompt := BuildPrompt("Build a login system", "npm test", "FAIL: expected 200 got 404")

	if !strings.Contains(prompt, "FAIL: expected 200 got 404") {
		t.Error("retry prompt missing failure output")
	}
	if !strings.Contains(prompt, "files from the previous attempt") {
		t.Error("retry prompt missing file context hint")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/executor/ -v -run TestBuildPrompt
```

Expected: FAIL — `BuildPrompt` not defined.

- [ ] **Step 3: Implement prompt builder and executor**

Create `internal/executor/executor.go`:

```go
package executor

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// BuildPrompt constructs the prompt for Claude.
// prevOutput is empty on the first attempt.
func BuildPrompt(spec, testCmd, prevOutput string) string {
	var b strings.Builder

	b.WriteString("Here is the spec:\n\n")
	b.WriteString(spec)
	b.WriteString("\n\nThe test command to verify your implementation is:\n")
	b.WriteString(testCmd)

	if prevOutput != "" {
		b.WriteString("\n\nThe previous attempt failed with this output:\n")
		b.WriteString(prevOutput)
		b.WriteString("\n\nNote: files from the previous attempt are still on disk. Read them to understand what was already tried before making changes.")
	}

	b.WriteString("\n\nWrite the implementation code to make the test pass. The test command will be run from the project root directory.")

	return b.String()
}

// ExecResult holds the output of a command execution.
type ExecResult struct {
	Output   string // combined stdout + stderr
	ExitCode int
}

// RunClaude invokes the claude CLI with the given prompt.
func RunClaude(ctx context.Context, claudeCmd, prompt, workDir string) ExecResult {
	cmd := exec.CommandContext(ctx, claudeCmd, "-p", "--dangerously-skip-permissions", prompt)
	cmd.Dir = workDir

	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return ExecResult{Output: string(out), ExitCode: exitCode}
}

// RunTestCmd executes a test command via sh -c.
func RunTestCmd(ctx context.Context, testCmd, workDir string) ExecResult {
	cmd := exec.CommandContext(ctx, "sh", "-c", testCmd)
	cmd.Dir = workDir

	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return ExecResult{Output: string(out), ExitCode: exitCode}
}

// FormatExitError returns a human-readable error string.
func FormatExitError(result ExecResult) string {
	return fmt.Sprintf("exit code %d:\n%s", result.ExitCode, result.Output)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/executor/ -v -run TestBuildPrompt
```

Expected: PASS.

- [ ] **Step 5: Write failing test for RunTestCmd**

Append to `internal/executor/executor_test.go`:

```go
func TestRunTestCmd_Pass(t *testing.T) {
	ctx := context.Background()
	result := RunTestCmd(ctx, "echo hello", t.TempDir())

	if result.ExitCode != 0 {
		t.Errorf("expected exit 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "hello") {
		t.Errorf("expected output to contain 'hello', got %q", result.Output)
	}
}

func TestRunTestCmd_Fail(t *testing.T) {
	ctx := context.Background()
	result := RunTestCmd(ctx, "exit 1", t.TempDir())

	if result.ExitCode != 1 {
		t.Errorf("expected exit 1, got %d", result.ExitCode)
	}
}
```

Add `"context"` to the import block at the top of the test file.

- [ ] **Step 6: Run tests to verify they pass**

```bash
go test ./internal/executor/ -v
```

Expected: PASS (these use the real implementation, no mocking needed).

- [ ] **Step 7: Commit**

```bash
git add internal/executor/executor.go internal/executor/executor_test.go
git commit -m "feat: add executor with prompt builder and command runner"
```

---

### Task 5: Reporter — Error.md and Terminal Output

**Files:**
- Create: `internal/reporter/reporter.go`
- Create: `internal/reporter/reporter_test.go`

- [ ] **Step 1: Write failing test for error.md generation**

Create `internal/reporter/reporter_test.go`:

```go
package reporter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriteErrorReport(t *testing.T) {
	dir := t.TempDir()

	report := ErrorReport{
		TestName:     "auth-login",
		Attempts:     5,
		Duration:     5 * time.Minute,
		LastExitCode: 1,
		Spec:         "Build a login system",
		FinalOutput:  "FAIL: expected 200 got 404",
		ClaudeOutput: "I created auth.go with...",
		History: []AttemptRecord{
			{Number: 1, TestOutput: "FAIL: file not found"},
			{Number: 2, TestOutput: "FAIL: expected 200 got 404"},
		},
	}

	err := WriteErrorReport(dir, report)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "error.md"))
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)

	checks := []string{
		"# Failed: auth-login",
		"**Attempts:** 5",
		"**Time spent:** 5m0s",
		"**Last exit code:** 1",
		"Build a login system",
		"FAIL: expected 200 got 404",
		"I created auth.go with...",
		"### Attempt 1",
		"### Attempt 2",
	}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("error.md missing: %q", check)
		}
	}
}

func TestDeleteErrorReport(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "error.md")
	os.WriteFile(path, []byte("old"), 0o644)

	DeleteErrorReport(dir)

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("error.md should have been deleted")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/reporter/ -v
```

Expected: FAIL — types not defined.

- [ ] **Step 3: Implement reporter**

Create `internal/reporter/reporter.go`:

```go
package reporter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type AttemptRecord struct {
	Number     int
	TestOutput string
}

type ErrorReport struct {
	TestName     string
	Attempts     int
	Duration     time.Duration
	LastExitCode int
	Spec         string
	FinalOutput  string
	ClaudeOutput string
	History      []AttemptRecord
}

func WriteErrorReport(testDir string, r ErrorReport) error {
	var b strings.Builder

	fmt.Fprintf(&b, "# Failed: %s\n\n", r.TestName)
	fmt.Fprintf(&b, "**Attempts:** %d\n", r.Attempts)
	fmt.Fprintf(&b, "**Time spent:** %s\n", r.Duration)
	fmt.Fprintf(&b, "**Last exit code:** %d\n", r.LastExitCode)

	b.WriteString("\n## Spec\n")
	b.WriteString(r.Spec)
	b.WriteString("\n")

	b.WriteString("\n## Final Test Output\n")
	b.WriteString(r.FinalOutput)
	b.WriteString("\n")

	b.WriteString("\n## Claude Output (Last Attempt)\n")
	b.WriteString(r.ClaudeOutput)
	b.WriteString("\n")

	b.WriteString("\n## Attempt History\n")
	for _, a := range r.History {
		fmt.Fprintf(&b, "### Attempt %d\n", a.Number)
		b.WriteString(a.TestOutput)
		b.WriteString("\n")
	}

	return os.WriteFile(filepath.Join(testDir, "error.md"), []byte(b.String()), 0o644)
}

func DeleteErrorReport(testDir string) {
	os.Remove(filepath.Join(testDir, "error.md"))
}

// FormatProgress returns a single-line progress string.
func FormatProgress(index, total int, name, status string, attempts int, duration time.Duration) string {
	attemptWord := "attempts"
	if attempts == 1 {
		attemptWord = "attempt"
	}
	line := fmt.Sprintf("[%d/%d] %s... %s (%d %s, %s)", index, total, name, status, attempts, attemptWord, duration.Round(time.Second))
	if status == "FAIL" {
		line += " → error.md written"
	}
	return line
}

// FormatSummary returns the final summary line.
func FormatSummary(passed, failed int) string {
	total := passed + failed
	return fmt.Sprintf("\nSummary: %d/%d passed, %d failed", passed, total, failed)
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/reporter/ -v
```

Expected: PASS.

- [ ] **Step 5: Write test for FormatProgress and FormatSummary**

Append to `internal/reporter/reporter_test.go`:

```go
func TestFormatProgress(t *testing.T) {
	line := FormatProgress(1, 4, "auth-login", "PASS", 2, 45*time.Second)
	if line != "[1/4] auth-login... PASS (2 attempts, 45s)" {
		t.Errorf("unexpected: %q", line)
	}

	line = FormatProgress(2, 4, "data-parser", "FAIL", 5, 5*time.Minute)
	if !strings.Contains(line, "error.md written") {
		t.Errorf("FAIL line missing error.md note: %q", line)
	}
}

func TestFormatSummary(t *testing.T) {
	s := FormatSummary(3, 1)
	if s != "\nSummary: 3/4 passed, 1 failed" {
		t.Errorf("unexpected: %q", s)
	}
}
```

- [ ] **Step 6: Run all reporter tests**

```bash
go test ./internal/reporter/ -v
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/reporter/reporter.go internal/reporter/reporter_test.go
git commit -m "feat: add reporter with error.md generation and progress formatting"
```

---

### Task 6: Core Loop — Wire It All Together

**Files:**
- Modify: `internal/runner/runner.go`
- Modify: `internal/runner/runner_test.go`

- [ ] **Step 1: Write failing test for the run loop**

Append to `internal/runner/runner_test.go`:

```go
func TestRun_AllPass(t *testing.T) {
	dir := t.TempDir()
	projectDir := t.TempDir()

	// Create a test that will pass — test_cmd just checks a file exists
	testDir := filepath.Join(dir, "create-file")
	os.MkdirAll(testDir, 0o755)
	os.WriteFile(filepath.Join(testDir, "spec.md"), []byte("create a file called hello.txt"), 0o644)
	os.WriteFile(filepath.Join(testDir, "test_cmd"), []byte("test -f hello.txt"), 0o644)

	// Pre-create the file so the test passes without Claude
	os.WriteFile(filepath.Join(projectDir, "hello.txt"), []byte("hi"), 0o644)

	cfg := RunConfig{
		TestDir:     dir,
		ProjectDir:  projectDir,
		ClaudeCmd:   "echo", // no-op claude — just echo the prompt
		MaxAttempts: 3,
		Timeout:     30 * time.Second,
		RetryFailed: false,
	}

	result := Run(cfg)
	if result.Failed != 0 {
		t.Errorf("expected 0 failures, got %d", result.Failed)
	}
	if result.Passed != 1 {
		t.Errorf("expected 1 pass, got %d", result.Passed)
	}

	status, _ := ReadStatus(testDir)
	if status != StatusPass {
		t.Errorf("expected pass status, got %s", status)
	}
}

func TestRun_FailExhaustsRetries(t *testing.T) {
	dir := t.TempDir()
	projectDir := t.TempDir()

	testDir := filepath.Join(dir, "will-fail")
	os.MkdirAll(testDir, 0o755)
	os.WriteFile(filepath.Join(testDir, "spec.md"), []byte("do something impossible"), 0o644)
	os.WriteFile(filepath.Join(testDir, "test_cmd"), []byte("false"), 0o644) // always fails

	cfg := RunConfig{
		TestDir:     dir,
		ProjectDir:  projectDir,
		ClaudeCmd:   "echo",
		MaxAttempts: 2,
		Timeout:     30 * time.Second,
		RetryFailed: false,
	}

	result := Run(cfg)
	if result.Failed != 1 {
		t.Errorf("expected 1 failure, got %d", result.Failed)
	}

	status, _ := ReadStatus(testDir)
	if status != StatusFail {
		t.Errorf("expected fail status, got %s", status)
	}

	// error.md should exist
	if _, err := os.Stat(filepath.Join(testDir, "error.md")); os.IsNotExist(err) {
		t.Error("error.md should have been written")
	}
}

func TestRun_SkipsPassedTests(t *testing.T) {
	dir := t.TempDir()
	projectDir := t.TempDir()

	testDir := filepath.Join(dir, "already-done")
	os.MkdirAll(testDir, 0o755)
	os.WriteFile(filepath.Join(testDir, "spec.md"), []byte("anything"), 0o644)
	os.WriteFile(filepath.Join(testDir, "test_cmd"), []byte("true"), 0o644)
	os.WriteFile(filepath.Join(testDir, ".status"), []byte("pass\n"), 0o644)

	cfg := RunConfig{
		TestDir:     dir,
		ProjectDir:  projectDir,
		ClaudeCmd:   "echo",
		MaxAttempts: 3,
		Timeout:     30 * time.Second,
		RetryFailed: false,
	}

	result := Run(cfg)
	// Should report 0 passed, 0 failed — it was skipped entirely
	if result.Passed != 0 || result.Failed != 0 {
		t.Errorf("expected 0/0, got passed=%d failed=%d", result.Passed, result.Failed)
	}
}
```

```go
func TestRun_RetryFailed(t *testing.T) {
	dir := t.TempDir()
	projectDir := t.TempDir()

	testDir := filepath.Join(dir, "retry-me")
	os.MkdirAll(testDir, 0o755)
	os.WriteFile(filepath.Join(testDir, "spec.md"), []byte("create hello.txt"), 0o644)
	os.WriteFile(filepath.Join(testDir, "test_cmd"), []byte("test -f hello.txt"), 0o644)
	os.WriteFile(filepath.Join(testDir, ".status"), []byte("fail\n"), 0o644)
	os.WriteFile(filepath.Join(testDir, "error.md"), []byte("old error"), 0o644)

	// Pre-create the file so test passes this time
	os.WriteFile(filepath.Join(projectDir, "hello.txt"), []byte("hi"), 0o644)

	cfg := RunConfig{
		TestDir:     dir,
		ProjectDir:  projectDir,
		ClaudeCmd:   "echo",
		MaxAttempts: 1,
		Timeout:     30 * time.Second,
		RetryFailed: true,
	}

	result := Run(cfg)
	if result.Passed != 1 {
		t.Errorf("expected 1 pass, got %d", result.Passed)
	}

	status, _ := ReadStatus(testDir)
	if status != StatusPass {
		t.Errorf("expected pass status, got %s", status)
	}

	// error.md should be deleted on success
	if _, err := os.Stat(filepath.Join(testDir, "error.md")); !os.IsNotExist(err) {
		t.Error("error.md should have been deleted after passing")
	}
}
```

Add `"time"` to the test file imports.

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/runner/ -v -run "TestRun_"
```

Expected: FAIL — `RunConfig` and `Run` not defined.

- [ ] **Step 3: Implement the Run function**

Add to `internal/runner/runner.go`:

Replace the entire import block in `runner.go` with:

```go
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
	tests, err := DiscoverTests(cfg.TestDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error discovering tests: %v\n", err)
		return RunResult{}
	}

	var result RunResult
	total := len(tests)
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
		WriteStatus(test.Dir, StatusInProgress)

		passed, attempts, duration := runSingleTest(cfg, test)

		if passed {
			WriteStatus(test.Dir, StatusPass)
			reporter.DeleteErrorReport(test.Dir)
			fmt.Println(reporter.FormatProgress(index, total, test.Name, "PASS", attempts, duration))
			result.Passed++
		} else {
			WriteStatus(test.Dir, StatusFail)
			fmt.Println(reporter.FormatProgress(index, total, test.Name, "FAIL", attempts, duration))
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
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/runner/ -v -run "TestRun_"
```

Expected: PASS.

- [ ] **Step 5: Run all tests**

```bash
go test ./...
```

Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/runner/runner.go internal/runner/runner_test.go
git commit -m "feat: implement core run loop with retry, status, and error reporting"
```

---

### Task 7: CLI Entry Point — Wire Flags to Runner

**Files:**
- Modify: `cmd/harness/main.go`
- Create: `cmd/harness/main_test.go`

- [ ] **Step 1: Write failing integration test**

Create `cmd/harness/main_test.go`:

```go
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLI_MissingArgs(t *testing.T) {
	cmd := exec.Command("go", "run", ".", )
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected non-zero exit")
	}
	if !strings.Contains(string(out), "Usage:") {
		t.Errorf("expected usage message, got: %s", out)
	}
}

func TestCLI_RunPassingTest(t *testing.T) {
	testDir := t.TempDir()
	projectDir := t.TempDir()

	// Create a test that passes
	td := filepath.Join(testDir, "simple")
	os.MkdirAll(td, 0o755)
	os.WriteFile(filepath.Join(td, "spec.md"), []byte("create hello.txt"), 0o644)
	os.WriteFile(filepath.Join(td, "test_cmd"), []byte("test -f hello.txt"), 0o644)

	// Pre-create the file
	os.WriteFile(filepath.Join(projectDir, "hello.txt"), []byte("hi"), 0o644)

	cmd := exec.Command("go", "run", ".",
		"run", testDir,
		"--project-dir", projectDir,
		"--claude-cmd", "echo",
		"--max-attempts", "1",
		"--timeout", "10s",
	)
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected success, got error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(string(out), "PASS") {
		t.Errorf("expected PASS in output, got: %s", out)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./cmd/harness/ -v -run TestCLI_RunPassingTest
```

Expected: FAIL — main.go doesn't parse flags yet.

- [ ] **Step 3: Implement full main.go with flag parsing**

Rewrite `cmd/harness/main.go`:

```go
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jeremygatt/jg-mini-harness/internal/runner"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "run" {
		fmt.Fprintf(os.Stderr, "Usage: harness run <test-dir> [flags]\n")
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

	// Extract positional arg (test dir) before flags
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: harness run <test-dir> [flags]\n")
		os.Exit(1)
	}

	testDir := args[0]
	runCmd.Parse(args[1:])

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
```

- [ ] **Step 4: Run CLI tests**

```bash
go test ./cmd/harness/ -v
```

Expected: PASS.

- [ ] **Step 5: Build the binary and smoke test**

```bash
go build -o harness ./cmd/harness && ./harness
```

Expected: prints usage, exits 1.

- [ ] **Step 6: Run all tests**

```bash
go test ./...
```

Expected: all PASS.

- [ ] **Step 7: Commit**

```bash
git add cmd/harness/main.go cmd/harness/main_test.go
git commit -m "feat: wire CLI flags to runner, complete harness binary"
```

---

### Task 8: End-to-End Smoke Test

**Files:**
- Create: `tests/example-echo/spec.md`
- Create: `tests/example-echo/test_cmd`

- [ ] **Step 1: Create an example test**

Create `tests/example-echo/spec.md`:

```markdown
Create a file called `output.txt` in the project root containing the text "hello world".
```

Create `tests/example-echo/test_cmd`:

```
grep -q "hello world" output.txt
```

- [ ] **Step 2: Build and run the harness against the example**

```bash
go build -o harness ./cmd/harness
./harness run ./tests --claude-cmd echo --max-attempts 1 --timeout 10s
```

This will fail (echo doesn't create files) but it validates the full pipeline: discovery → execution → test → error.md.

- [ ] **Step 3: Verify error.md was written**

```bash
cat tests/example-echo/error.md
```

Expected: contains "Failed: example-echo", attempt history, spec contents.

- [ ] **Step 4: Verify .status file**

```bash
cat tests/example-echo/.status
```

Expected: `fail`

- [ ] **Step 5: Clean up and commit**

```bash
rm -f tests/example-echo/.status tests/example-echo/error.md
git add tests/example-echo/spec.md tests/example-echo/test_cmd
git commit -m "feat: add example test for smoke testing"
```

- [ ] **Step 6: Add .gitignore**

Create `.gitignore`:

```
harness
tests/**/.status
tests/**/error.md
output.txt
```

```bash
git add .gitignore
git commit -m "chore: add .gitignore for build artifacts and test state"
```

---
