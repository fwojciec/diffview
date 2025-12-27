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
