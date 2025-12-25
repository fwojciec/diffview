---
description: Pick a ready beads task, create branch, and implement with behavioral TDD
allowed-tools: Bash(bd:*), Bash(git:*), Bash(make:*)
---

## Current State

Branch: !`git branch --show-current`
Uncommitted changes: !`git status --porcelain`
Beads uncommitted: !`git status --porcelain .beads/`

## In-Progress Work

!`bd list --status in_progress 2>/dev/null || echo "None"`

## Task Argument

Provided task ID: $1

## Your Workflow

### 1. Pre-flight Validation

Before proceeding, verify:
- [ ] Currently on `main` branch (if not, ask user before proceeding)
- [ ] No uncommitted changes in `.beads/` directory (if there are, commit and push them first)
- [ ] Working tree is clean (if not, ask user how to proceed)

If any checks fail, stop and resolve with the user before continuing.

### 2. Check for Abandoned Work

If there are issues with status `in_progress`:
- Show them to the user
- Ask: "Continue with existing in-progress work, or start fresh task?"
- If continuing: skip to step 4 with existing branch
- If starting fresh: ask if abandoned work should be reset to `open`

### 3. Task Selection

**If a task ID was provided via argument ($1)**:
- Verify the task exists: run `bd show <task-id>`
- Skip to step 4 (Branch Setup)

**If no task ID was provided**:
- Run `bd ready` to show available tasks
- Present the ready tasks to the user with a brief recommendation based on:
  - Task complexity and dependencies
  - Logical ordering (foundational work before dependent work)
- Use the AskUserQuestion tool to let the user choose which task to work on

### 4. Branch Setup

Once you have a task ID (either from argument or user selection):
1. Create branch first: `git checkout -b <task-id>` (e.g., `git checkout -b diffview-abc`)
2. Mark the task as in-progress: `bd update <task-id> -s in_progress`
3. Commit the beads change: `git add .beads/ && git commit -m "Start work on <task-id>"`
4. Show full task details: `bd show <task-id>`

**Note**: All commits happen on the feature branch, keeping main clean.

### 5. Implementation

#### When to Use TDD

Use the `superpowers:test-driven-development` skill when implementing **behavioral requirements**—code that has observable effects, makes decisions, or transforms data.

**TDD applies when:**
- Implementing a use case or requirement
- Adding business logic or decision-making code
- Creating public API contracts
- Building adapters that integrate with external systems

**TDD does NOT apply when:**
- Creating pure data types (structs with no methods, or only trivial accessors)
- Defining interfaces or type aliases
- Writing code during the REFACTOR phase (new internal classes, helpers extracted from working code)
- Adding configuration or constants

The key insight from Kent Beck: *"Adding a new class is not the trigger for writing tests. The trigger is implementing a requirement."*

#### Decision Heuristic

Before writing a test, ask: "Does this code have behavior that could be wrong?"

- **Yes** → Write a failing test first (RED-GREEN-REFACTOR)
- **No** → Implement directly; behavior tests elsewhere will catch integration issues

Example: Creating a `DiffStats` struct with fields is not testable behavior. Computing statistics from a diff IS testable behavior.

#### The RED-GREEN-REFACTOR Cycle

When TDD applies:
1. Write a failing test that expresses the requirement
2. Implement minimal code to pass (design quality takes a backseat)
3. Refactor to clean up (do NOT write new tests here—they couple to implementation)
4. Repeat for next requirement

Tests written during refactoring tend to couple to implementation details and break during future refactors. Let behavioral tests cover refactored internals.

#### Architectural Decisions

If the task involves any of these:
- Creating new packages or files
- Deciding where code belongs
- Adding new mocks or mock methods
- Package naming decisions

Then **ALSO** use the `go-standard-package-layout` skill for guidance.

### 6. Progress Checkpointing

At major milestones during implementation, update beads notes:
```bash
bd update <task-id> --notes "COMPLETED: [what's done]
IN_PROGRESS: [current work]
NEXT: [immediate next step]
KEY_DECISIONS: [any important choices made]"
```

Commit beads changes with your code commits to keep them in sync.

### 7. Validation

After implementation is complete:
1. Run `make validate`
2. Address any issues that arise (linting, test failures, etc.)
3. Iterate until validation passes

Only proceed to step 8 when `make validate` passes cleanly.

### 8. Self-Review

Before finishing, get a fresh perspective on the implementation. This is an internal feedback loop to catch issues before committing—not a formal PR review (that happens after `/finish-task`).

**Launch both reviews in parallel:**
1. `code-review:code-review` - correctness, style, bugs (review uncommitted changes, not a PR)
2. `beads-review` - forward compatibility with downstream work

Wait for both to complete.

**Evaluate feedback with YAGNI awareness:**

When processing review suggestions, distinguish between:
- **Feature YAGNI**: Reject suggestions to add features/capabilities you don't need yet
- **Structural YAGNI**: Accept suggestions that add seams, boundaries, or dependency injection - even if "you don't need it yet"

The question: "Does this preserve our capacity for change?" - not "Do we need this feature?"

Use `superpowers:receiving-code-review` to evaluate each suggestion on merit. Accept what improves correctness or structural discipline. Push back on stylistic preferences.

**If beads-review recommends issue updates:**
- Update current issue notes with review findings
- Update downstream issues if insights affect their specs

**If changes are needed:**
- Implement fixes (return to step 5 if substantial)
- Run `make validate` again
- Repeat self-review if changes were significant

Only proceed to `/finish-task` when both reviews are addressed and validation passes.
