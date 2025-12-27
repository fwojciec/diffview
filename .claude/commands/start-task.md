---
description: Pick a ready beads task, create worktree, and implement with behavioral TDD
allowed-tools: Bash(bd:*), Bash(git:*), Bash(make:*)
---

## Current State

Working directory: !`pwd`
Is worktree: !`git rev-parse --is-inside-work-tree 2>/dev/null && git worktree list | grep -q "$(pwd)" && echo "yes (worktree)" || echo "no (main)"`
Main repo: !`git worktree list | head -1 | awk '{print $1}'`
Existing worktrees: !`git worktree list`

## In-Progress Work

!`bd list --status in_progress 2>/dev/null || echo "None"`

## Task Argument

Provided task ID: $1

## Your Workflow

### 1. Pre-flight Validation

Before proceeding, verify:
- [ ] No uncommitted changes in current directory
- [ ] Daemon mode is disabled: `export BEADS_NO_DAEMON=1`

If there are uncommitted changes, ask user how to proceed.

### 2. Check for Abandoned Work

If there are issues with status `in_progress`:
- Show them to the user
- Ask: "Continue with existing in-progress work, or start fresh task?"
- If continuing: navigate to existing worktree
- If starting fresh: ask if abandoned work should be reset to `open`

### 3. Task Selection

**If a task ID was provided via argument ($1)**:
- Verify the task exists: run `bd show <task-id>`
- Skip to step 4 (Worktree Setup)

**If no task ID was provided**:
- Run `bd ready` to show available tasks
- Present the ready tasks to the user with a brief recommendation based on:
  - Task complexity and dependencies
  - Logical ordering (foundational work before dependent work)
- Use the AskUserQuestion tool to let the user choose which task to work on

### 4. Worktree Setup

Once you have a task ID:

```bash
# Get main repo path
MAIN_REPO=$(git worktree list | head -1 | awk '{print $1}')

# Create worktree in hidden location
git worktree add "$MAIN_REPO/.git/beads-worktrees/<task-id>" -b <task-id>

# Mark task as in-progress (shared DB, works from anywhere)
bd update <task-id> -s in_progress
```

**Tell the user:**
> Worktree created at: `$MAIN_REPO/.git/beads-worktrees/<task-id>`
>
> To work on this task, open a new Claude Code session in that directory:
> ```bash
> cd $MAIN_REPO/.git/beads-worktrees/<task-id>
> claude
> ```
>
> Or continue in this session if you prefer (I'll work in the worktree path).

Show full task details: `bd show <task-id>`

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

Note: Beads state is in the shared database. It will be synced when finishing the task.

### 7. Validation

After implementation is complete:
1. Run `make validate`
2. Address any issues that arise (linting, test failures, etc.)
3. Iterate until validation passes

Only proceed to step 8 when `make validate` passes cleanly.

### 8. Self-Review

Before finishing, get independent perspectives on the implementation. Launch two review subagents in parallel—each provides a "second opinion" from a different angle, and running them concurrently saves time.

**Launch both reviews in parallel using a single message with multiple Task calls:**
1. `Task(subagent_type="superpowers:code-reviewer")` - correctness, style, bugs (has conversation context for second opinion)
2. `Task(subagent_type="beads-review")` - forward compatibility with downstream work

Wait for both to complete. The value is in getting two independent assessments—issues flagged by both reviewers deserve extra attention.

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
