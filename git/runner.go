// Package git provides access to git operations via shell commands.
package git

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/fwojciec/diffview"
)

// Compile-time interface verification.
var _ diffview.GitRunner = (*Runner)(nil)

// Runner executes git commands via shell.
type Runner struct{}

// NewRunner creates a new git runner.
func NewRunner() *Runner {
	return &Runner{}
}

// Log returns commit hashes from the repository at repoPath, limited to n commits.
func (r *Runner) Log(ctx context.Context, repoPath string, limit int) ([]string, error) {
	args := []string{"-C", repoPath, "log", "--format=%H", fmt.Sprintf("-n%d", limit)}
	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("git log failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	// Filter empty lines
	var hashes []string
	for _, line := range lines {
		if line != "" {
			hashes = append(hashes, line)
		}
	}
	return hashes, nil
}

// Show returns the diff for a specific commit hash.
func (r *Runner) Show(ctx context.Context, repoPath string, hash string) (string, error) {
	args := []string{"-C", repoPath, "show", "--format=", hash}
	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git show failed: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("git show failed: %w", err)
	}
	return string(output), nil
}

// Message returns the commit message for a specific commit hash.
func (r *Runner) Message(ctx context.Context, repoPath string, hash string) (string, error) {
	args := []string{"-C", repoPath, "show", "--format=%B", "-s", hash}
	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git show failed: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("git show failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// MergeCommits returns merge commit hashes from the repository, limited to n commits.
func (r *Runner) MergeCommits(ctx context.Context, repoPath string, limit int) ([]string, error) {
	args := []string{"-C", repoPath, "log", "--merges", "--format=%H", fmt.Sprintf("-n%d", limit)}
	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("git log --merges failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("git log --merges failed: %w", err)
	}

	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return nil, nil
	}
	lines := strings.Split(trimmed, "\n")
	hashes := make([]string, 0, len(lines))
	for _, line := range lines {
		if line != "" {
			hashes = append(hashes, line)
		}
	}
	return hashes, nil
}

// CommitsInRange returns commits between base and head (base exclusive, head inclusive).
func (r *Runner) CommitsInRange(ctx context.Context, repoPath, base, head string) ([]diffview.CommitBrief, error) {
	// Use null byte as separator between hash and subject for safe parsing
	// Format: hash<NUL>subject
	rangeArg := fmt.Sprintf("%s..%s", base, head)
	args := []string{"-C", repoPath, "log", "--format=%H%x00%s", rangeArg}
	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("git log failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("git log failed: %w", err)
	}

	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return nil, nil
	}
	lines := strings.Split(trimmed, "\n")
	commits := make([]diffview.CommitBrief, 0, len(lines))
	for _, line := range lines {
		parts := strings.SplitN(line, "\x00", 2)
		if len(parts) != 2 {
			continue
		}
		commits = append(commits, diffview.CommitBrief{
			Hash:    parts[0],
			Message: parts[1],
		})
	}
	return commits, nil
}

// DiffRange returns the combined diff between base and head.
// Uses three-dot notation (base...head) to show changes introduced by head since merge-base.
func (r *Runner) DiffRange(ctx context.Context, repoPath, base, head string) (string, error) {
	// Three-dot diff: shows changes in head relative to the merge-base with base
	rangeArg := fmt.Sprintf("%s...%s", base, head)
	args := []string{"-C", repoPath, "diff", rangeArg}
	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git diff failed: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("git diff failed: %w", err)
	}
	return string(output), nil
}

// CurrentBranch returns the name of the currently checked out branch.
func (r *Runner) CurrentBranch(ctx context.Context, repoPath string) (string, error) {
	args := []string{"-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD"}
	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git rev-parse failed: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("git rev-parse failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// MergeBase returns the best common ancestor commit between two refs.
func (r *Runner) MergeBase(ctx context.Context, repoPath, ref1, ref2 string) (string, error) {
	args := []string{"-C", repoPath, "merge-base", ref1, ref2}
	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git merge-base failed: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("git merge-base failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
