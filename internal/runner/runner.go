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
