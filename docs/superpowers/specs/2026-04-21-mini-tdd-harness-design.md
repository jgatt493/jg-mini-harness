# Mini TDD Harness — Design Spec

## Overview

A standalone Go CLI binary that automates test-driven development by iterating through a directory of test specs, spawning Claude Code sessions to write implementations, and verifying them against user-defined test commands. Designed to run as a background process while the user works in a separate Claude Code session writing new specs.

## Problem

Writing a large number of test specs up front and then manually running Claude Code against each one is tedious. The harness automates the loop: pick up a test, have Claude implement it, verify it passes, move on.

## Architecture

### Directory Structure

```
jg-mini-harness/
  cmd/
    harness/
      main.go              # Entry point, CLI flags
  internal/
    runner/
      runner.go            # Core loop: discover, execute, retry
    executor/
      executor.go          # Shells out to claude CLI and test commands
    reporter/
      reporter.go          # Writes error.md files, terminal output
  go.mod
```

### Test Directory Convention

Tests live in a user-specified directory (default: `./tests`). Each subdirectory is one test:

```
tests/
  auth-login/
    spec.md          # Natural language requirement for Claude
    test_cmd         # Shell command to verify (exit 0 = pass)
    .status          # Written by harness: pass | fail | in_progress
    error.md         # Written by harness on failure
  data-parser/
    spec.md
    test_cmd
```

**Required files per test:**
- `spec.md` — the requirement/prompt. Freeform markdown describing what to build.
- `test_cmd` — a file whose first line is read and executed via `sh -c "<contents>"`. Exit code 0 means pass, non-zero means fail.

**Harness-managed files per test:**
- `.status` — tracks test state. Values: `pass`, `fail`, `in_progress`. Absence means pending.
- `error.md` — written on failure with full attempt history. Deleted if test passes on re-run.

## Core Loop

```
scan test directory for subdirectories
sort alphabetically
for each test dir:
  if .status exists and is "pass" → skip
  if .status exists and is "fail" → skip (unless --retry-failed)
  if .status exists and is "in_progress" → skip (stale from crashed run)

  write .status = "in_progress"
  attempts = 0
  start_time = now()
  attempt_history = []

  loop:
    attempts++

    build prompt:
      - include spec.md contents
      - include test_cmd contents
      - if previous attempt failed, include failure output
      - instruct Claude to make the test pass

    shell out to: claude -p "{prompt}" (working dir = project-dir)

    run test_cmd (working dir = project-dir)
    capture stdout + stderr

    if exit code 0:
      write .status = "pass"
      delete error.md if it exists
      log: [n/total] test-name... PASS (attempts, duration)
      break

    if attempts >= max-attempts OR elapsed >= timeout:
      write .status = "fail"
      write error.md with full attempt history
      log: [n/total] test-name... FAIL (attempts, duration) → error.md written
      break

    record attempt output in attempt_history
    continue loop

print summary: X/Y passed, Z failed
```

## Claude Invocation

The harness shells out to the `claude` CLI using `-p` (prompt mode, non-interactive) with `--dangerously-skip-permissions` for fully autonomous execution:

```bash
claude -p --dangerously-skip-permissions "Here is the spec:

{contents of spec.md}

The test command to verify your implementation is:
{contents of test_cmd}

{if retry: The previous attempt failed with this output:
{previous test output}

Note: files from the previous attempt are still on disk. Read them to understand what was already tried before making changes.
}

Write the implementation code to make the test pass. The test command will be run from the project root directory."
```

The working directory for both `claude` and `test_cmd` is `--project-dir` (defaults to the current directory). This ensures Claude writes files in the actual project, not inside the test directory.

**Retry context:** On retries, Claude receives the spec, the test command, and the previous failure output. Claude does NOT receive its own previous response, but the files it wrote in previous attempts are still on disk. The prompt instructs Claude to read existing files to understand prior work. This avoids bloating the prompt while giving Claude full context via the filesystem.

**Claude output capture:** The harness captures Claude's stdout/stderr from each `claude -p` invocation. On failure, Claude's output from the last attempt is included in `error.md` for debugging.

## CLI Interface

```bash
# Basic usage
harness run ./tests

# With options
harness run ./tests \
  --max-attempts 5 \
  --timeout 5m \
  --project-dir . \
  --claude-cmd claude
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--max-attempts` | `5` | Max retries per test before marking as failed |
| `--timeout` | `5m` | Max wall-clock time per test (Go duration format) |
| `--project-dir` | `.` | Working directory for claude and test commands |
| `--claude-cmd` | `claude` | Path to the claude CLI binary |
| `--retry-failed` | `false` | Re-run tests with `.status = fail` |

## Terminal Output

Progress log during execution:

```
[1/4] auth-login... PASS (2 attempts, 45s)
[2/4] data-parser... FAIL (5 attempts, 5m00s) → error.md written
[3/4] api-routes... PASS (1 attempt, 22s)
[4/4] db-schema... PASS (3 attempts, 1m30s)

Summary: 3/4 passed, 1 failed
```

## Error Reporting

On failure, `error.md` is written inside the test directory:

```markdown
# Failed: auth-login

**Attempts:** 5
**Time spent:** 5m00s
**Last exit code:** 1

## Spec
<contents of spec.md>

## Final Test Output
<stdout + stderr from the last test_cmd run>

## Claude Output (Last Attempt)
<stdout + stderr from the last claude -p invocation>

## Attempt History
### Attempt 1
<test output>
### Attempt 2
<test output>
...
```

Full attempt history is included so the user can see whether Claude was converging or going in circles.

Behavior:
- `error.md` is overwritten if the test is re-run and fails again.
- `error.md` is deleted if the test passes on re-run.

## Status Management

The `.status` file is a plain text file containing a single word:

- **No file** — test is pending, will be picked up on next run
- `in_progress` — currently being worked on
- `pass` — completed successfully, skipped on future runs
- `fail` — exhausted retries, `error.md` has details

**Re-queuing tests:**
- Delete a single test's `.status` file to re-queue it
- `find tests -name .status -delete` to re-queue everything
- Use `--retry-failed` flag to re-run all failed tests without manual file deletion
- Extensible: future statuses can be added without changing the file format

**Stale `in_progress`:** If the harness crashes mid-run, tests marked `in_progress` will be skipped on restart. The user must manually delete the `.status` file to re-queue. This is intentional — avoids re-running partially completed work that may have side effects.

## Language & Dependencies

- **Go** (latest stable)
- **Zero external dependencies** — stdlib only (`os/exec`, `filepath`, `flag`, `time`, `fmt`, `os`)
- Single binary output, no runtime requirements beyond having `claude` CLI on PATH

## Scope Exclusions

The following are explicitly out of scope for the initial build:

- Parallelization (one test at a time)
- Watch mode / file system polling
- Per-test config overrides (custom timeouts per test)
- Web UI or dashboard
- Persistent database or structured logging
- Skill/plugin integration with Claude Code
