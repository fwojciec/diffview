---
description: Validate, close beads issue, create PR, and clean up worktree
allowed-tools: Bash(bd:*), Bash(git:*), Bash(gh:*), Bash(make:*)
---

## Current State

Working directory: !`pwd`
Branch: !`git branch --show-current`
Main repo: !`git worktree list | head -1 | awk '{print $1}'`
Git status: !`git status --porcelain`

## Your Workflow

### 1. Final Validation

Run `make validate` (the full validation suite).

If any issues arise:
- Fix them systematically
- Re-run validation
- Do not proceed until validation passes cleanly

### 2. Commit Outstanding Work

Ensure all implementation work is committed:
- [ ] No uncommitted code changes
- [ ] No temporary files or debug artifacts
- [ ] All commits have meaningful messages

### 3. Close Beads Issue

Extract the task ID from the current branch name (format: `diffview-XXX`).

```bash
# Close the issue (shared DB, works from any worktree)
bd close <task-id>
```

Note: Beads state is in the shared database. The JSONL export will be synced in step 6.

### 4. Create Pull Request

Push branch and create PR:

```bash
git push -u origin <branch-name>
gh pr create --title "<title>" --body "$(cat <<'EOF'
## Summary
<2-3 bullets of what changed>

## Test Plan
- [ ] <verification steps>

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

Report the PR URL to the user.

### 5. Sync Beads State

Sync beads from the main repo to commit the closed issue state:

```bash
MAIN_REPO=$(git worktree list | head -1 | awk '{print $1}')

# Sync beads (exports JSONL and commits in main repo)
cd "$MAIN_REPO" && bd sync && cd -
```

### 6. Worktree Cleanup Instructions

**Tell the user:**

> PR created! The worktree can be removed after the PR is merged.
>
> After merge, run from main repo:
> ```bash
> cd <main-repo-path>
> git worktree remove .git/beads-worktrees/<task-id>
> git branch -d <task-id>  # Delete local branch
> git fetch --prune        # Clean up remote tracking
> ```
>
> Or to continue working on another task, just run `/start-task` from the main repo.

### 7. Final Verification

- [ ] PR is created and URL shared with user
- [ ] Beads issue shows as `closed` in `bd show <task-id>`
- [ ] User knows how to clean up worktree after merge
