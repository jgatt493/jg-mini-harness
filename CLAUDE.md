# Mini TDD Harness

A Go CLI that automates test-driven development by iterating through test specs,
spawning Claude Code sessions to implement them, and verifying via test commands.

## Install

```bash
# One-liner (requires Go)
curl -fsSL https://raw.githubusercontent.com/jgatt493/jg-mini-harness/master/install.sh | bash

# Or manually
git clone https://github.com/jgatt493/jg-mini-harness.git
cd jg-mini-harness
go build -o harness ./cmd/harness
cp harness ~/.local/bin/harness
```

## Quick Start

```bash
harness run              # scans ./TDD by default
harness run --retry-failed
harness version          # prints version
```

## Test Convention

Each test is a directory under `TDD/` with:
- `spec.md` — requirement for Claude to implement
- `test_cmd` — shell command to verify (exit 0 = pass)

## Version

Check with `harness version` — outputs `jg-mini-harness v<semver>`.

## Skills

- skills/write-tdd.md — Write TDD test specs for the harness
