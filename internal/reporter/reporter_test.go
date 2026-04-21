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
