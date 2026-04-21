package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLI_MissingArgs(t *testing.T) {
	cmd := exec.Command("go", "run", ".")
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
