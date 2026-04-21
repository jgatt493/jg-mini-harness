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
