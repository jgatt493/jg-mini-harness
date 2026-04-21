# Mini TDD Harness

A Go CLI that automates test-driven development by iterating through test specs, spawning Claude Code sessions to implement them, and verifying via test commands.

## Quick Start

```bash
go build -o harness ./cmd/harness
./harness run              # scans ./TDD by default
./harness run --retry-failed
```

## Test Convention

Each test is a directory under `TDD/` with:
- `spec.md` — requirement for Claude to implement
- `test_cmd` — shell command to verify (exit 0 = pass)

## Skills

- skills/write-tdd.md — Write TDD test specs for the harness (auto-triggered when writing tests)
