# CLAUDE.md

Strategic guidance for LLMs working with this codebase.

## Why This Codebase Exists

**Core Problem**: Reviewing code written by AI agents is difficult. Standard diff viewers show line-by-line changes but lack context for understanding larger refactors, multi-file changes, or the intent behind modifications.

**Solution**: A diff viewer tool designed specifically for reviewing agent-generated code, with features that help humans quickly understand and validate AI-authored changes.

## Design Philosophy

- **Ben Johnson Standard Package Layout** - domain types in root, dependencies in subdirectories
- **Local-first** - all processing happens locally
- **CLI-native** - designed for terminal workflows
- **Process over polish** - systematic validation results in quality rather than fixing issues post-hoc

## Workflows

Use slash commands for standard development workflows:

| Command | Purpose |
|---------|---------|
| `/start-task` | Pick a ready task, create branch, implement with TDD |
| `/finish-task` | Validate, close beads issue, create PR |
| `/address-pr-comments` | Fetch, evaluate, and respond to PR feedback |

**Quick reference**:
```bash
make validate     # Quality gate - run before completing any task
```

## Architecture Patterns

**Ben Johnson Pattern**:
- Root package: domain types and interfaces only (no external dependencies)
- Subdirectories: one per external dependency
- `mock/`: manual mocks with function fields for testing
- `cmd/diffview/`: wires everything together

**File Naming Convention**:
- `foo/foo.go`: shared utilities for the package
- `foo/foo_test.go`: shared test utilities (in `foo_test` package)
- Entity files: named after domain entity (`user.go`, `viewer.go`)

When uncertain about where code belongs, use the `go-standard-package-layout` skill.

## Skills

### Architecture

**`go-standard-package-layout`** - Use when:
- Creating new packages or files
- Deciding where code belongs
- Naming packages or files
- Writing mocks in `mock/`

### Development (invoked automatically by `/start-task`)

- **`superpowers:test-driven-development`** - Write test first, watch it fail, implement
- **`superpowers:systematic-debugging`** - Understand root cause before fixing
- **`superpowers:verification-before-completion`** - Evidence before assertions

## Writing Issues

Issues should be easy to complete. Always include a description when creating:

```bash
bd create "Title" -p P2 -t task --description "## Problem
[What needs to be fixed/added]

## Entrypoints
- [File or function where work starts]

## Validation
- [ ] Specific testable outcome
- [ ] make validate passes"
```

**Principles**:
- Write **what** needs doing, not **how**
- One issue = one PR
- Reference specific files to reduce discovery time

## Test Philosophy

**TDD is mandatory** - write failing tests first, then implement.

**Package Convention**:
- All tests MUST use external test packages: `package foo_test` (not `package foo`)
- This enforces testing through the public API only
- Linter (`testpackage`) will fail on tests in the same package

**Parallel Tests**:
- All tests MUST call `t.Parallel()` at the start of:
  - Every top-level test function
  - Every subtest (`t.Run` callback)
- Linter (`paralleltest`) will fail on missing parallel calls

**Example Pattern**:
```go
package sqlite_test  // External test package

func TestFoo(t *testing.T) {
    t.Parallel()  // Required

    t.Run("subtest", func(t *testing.T) {
        t.Parallel()  // Also required
        // test code...
    })
}
```

**Assertions**:
- Use `require` for setup (fails fast)
- Use `assert` for test assertions (continues on failure)
- Use `assert.Empty(t, slice)` not `assert.Len(t, slice, 0)`

**Interface Compliance Checks**:
Go's `var _ Interface = (*Type)(nil)` pattern verifies interface implementation at compile time. These checks MUST be in production code, NOT in tests:

```go
// CORRECT: In parser.go (production code)
var _ diffview.Parser = (*Parser)(nil)

// WRONG: In parser_test.go (test file)
var _ diffview.Parser = (*Parser)(nil)  // Don't do this
```

Why: Tests provide runtime verification only. Production code provides compile-time guarantees—the compiler catches interface mismatches immediately, before any code runs.

## Linting

golangci-lint enforces:
- No global state (`gochecknoglobals`) - per Ben Johnson pattern
- Separate test packages (`testpackage`)
- Error checking (`errcheck`) - all errors must be handled

## Reference Documentation

- `.claude/commands/` - Workflow commands
