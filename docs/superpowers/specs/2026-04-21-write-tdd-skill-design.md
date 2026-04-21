# Write TDD Skill — Design Spec

## Overview

A Claude Code skill (`skills/write-tdd.md`) that helps users author TDD test specs for the mini harness. When triggered, it reads the project, helps the user define what needs testing, and generates `TDD/<name>/spec.md` + `TDD/<name>/test_cmd` pairs ready for the harness to consume.

## Trigger

The skill should be auto-invoked when the user expresses intent to write TDD tests. Examples:
- "let's write TDD tests for this app"
- "I need test specs for the auth system"
- "write tests for this feature"
- "add TDD specs"

The skill description in CLAUDE.md and frontmatter should be written to match these patterns.

## Skill File

Single file: `skills/write-tdd.md`

Registered in `CLAUDE.md` so Claude Code can discover and auto-invoke it.

## Workflow

The skill operates in three modes depending on user input:

### Mode 1: Exploratory
User says something like "let's write TDD tests for this app."

1. Read the project structure, tech stack, existing source code
2. Read existing `TDD/` directory to understand what's already covered
3. Propose areas that need test coverage
4. Present a numbered list of test names + one-line descriptions
5. User approves, edits, removes, or adds to the list
6. Generate all approved tests

### Mode 2: Directed
User says something like "I need tests for the auth system."

1. Read the relevant code/area
2. Read existing `TDD/` to avoid duplicates
3. Propose specific tests for that area
4. Same approval → generation flow

### Mode 3: Bulk Import
User dumps a list of test ideas (numbered list, bullet points, freeform text).

1. Parse the list into individual tests
2. Propose test names, suggest splitting any that are too broad
3. Same approval → generation flow

### Generation Phase (all modes)

After the user approves the test list:

For each test:
1. Create `TDD/<test-name>/spec.md` — detailed, self-contained requirement
2. Create `TDD/<test-name>/test_cmd` — appropriate verification command

Test names should be kebab-case directory names (e.g. `auth-login`, `parse-csv-headers`).

## Spec.md Quality Requirements

Each `spec.md` is consumed by `claude -p --dangerously-skip-permissions` in headless mode. It must be self-contained — no follow-up questions possible. Each spec must include:

- **What to build** — the requirement in plain language
- **Where to put it** — file paths or enough context for Claude to decide
- **Constraints** — what NOT to do (don't break existing functionality, don't modify unrelated code)
- **Acceptance criteria** — what "done" looks like, aligned with the `test_cmd`

## Test Command Generation

The skill reads the project to determine the appropriate test runner and generates `test_cmd` accordingly:

- Go project → `go test ./...` or specific package test
- Node/TS project → `npm test`, `npx vitest run`, etc.
- Python project → `pytest tests/...`
- Generic → `sh -c "..."` with inline verification

The test command must:
- Exit 0 on success, non-zero on failure
- Be a single line (first line of `test_cmd` is what the harness reads)
- Run from the project root directory

## Duplicate Awareness

Before generating, the skill checks existing `TDD/` subdirectories and their `spec.md` files. If a proposed test overlaps with an existing one, it flags the overlap and asks the user whether to skip, replace, or keep both.

## Iteration Support

After generating tests, the skill remains available for:
- "Let's add more tests"
- "Break test X into smaller pieces"
- "This test is too vague, make it more specific"
- "Remove test Y"

The skill can modify existing `spec.md` and `test_cmd` files, not just create new ones.

## File Structure

```
jg-mini-harness/
  skills/
    write-tdd.md          # The skill file
  CLAUDE.md               # References the skill
  TDD/                    # Generated test specs go here
    feature-name/
      spec.md
      test_cmd
```

## Scope Exclusions

- Does not run the harness (that's a separate process)
- Does not implement the code to pass the tests
- Does not manage `.status` files
- No Go code — the skill is pure prompt markdown
