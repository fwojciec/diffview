# Git-Only Diffstory Design

## Problem

When a diff is piped to diffstory, we lose git context (branch, commits). The original task (diffview-1ib) proposed detecting git context at runtime and comparing piped diffs against expected diffs. This adds complexity for marginal benefit.

## Decision

Remove pipe/file modes from diffstory entirely. The tool only works in git repos with a configured remote.

**Rationale:**
- diffstory is for reviewing PR-worthy work, which implies a remote exists
- Simpler implementation with clear failure modes
- Git context is always available, improving classification quality

## Design

### Behavior

1. `diffstory` with no args → detect base branch, show diff from base...HEAD
2. No pipe mode, no file mode
3. Clear errors for: no remote, not a git repo, already on base branch

### Base Branch Detection

Use `git symbolic-ref refs/remotes/origin/HEAD` to get the remote's default branch.

**Why this approach:**
- Reflects what GitHub/GitLab actually configured
- Requires remote (appropriate for PR workflow)
- Single source of truth, no guessing

### Interface Changes

Add to `GitRunner`:

```go
// DefaultBranch returns the default branch name from origin/HEAD.
// Returns an error if no remote is configured.
DefaultBranch(ctx context.Context, repoPath string) (string, error)
```

### Files Changed

- `diffview.go` - add DefaultBranch to interface
- `git/runner.go` - implement via git symbolic-ref
- `mock/git.go` - add mock function field
- `cmd/diffstory/main.go` - remove pipe/file modes, use DefaultBranch

### Removed

- `DIFFSTORY_BASE_BRANCH` environment variable
- Pipe mode (`echo diff | diffstory`)
- File mode (`diffstory file.diff`)

## Future Work

Issue diffview-75e: Support arbitrary commit ranges (`diffstory main...feature`)
