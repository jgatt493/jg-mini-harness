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
		line += " \u2192 error.md written"
	}
	return line
}

// FormatSummary returns the final summary line.
func FormatSummary(passed, failed int) string {
	total := passed + failed
	return fmt.Sprintf("\nSummary: %d/%d passed, %d failed", passed, total, failed)
}
