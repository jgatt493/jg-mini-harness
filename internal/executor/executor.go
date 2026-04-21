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
