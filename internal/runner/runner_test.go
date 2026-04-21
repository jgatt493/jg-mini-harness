package runner

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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
	if result.Passed != 0 || result.Failed != 0 {
		t.Errorf("expected 0/0, got passed=%d failed=%d", result.Passed, result.Failed)
	}
}

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
