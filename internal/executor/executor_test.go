package executor

import (
	"context"
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
