package git_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fwojciec/diffview/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRepo creates a temporary git repository with a known history for testing.
// Returns the repo path and a cleanup function.
func setupTestRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	// Initialize repo with "main" as default branch
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")

	// Create initial commit on main
	writeFile(t, dir, "README.md", "# Test Repo\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "Initial commit")

	return dir
}

// runGit executes a git command in the given directory.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "command git %v failed: %s", args, string(output))
	return string(output)
}

// writeFile creates a file with the given content.
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
}

func TestRunner_MergeCommits(t *testing.T) {
	t.Parallel()

	t.Run("returns merge commits from history", func(t *testing.T) {
		t.Parallel()
		dir := setupTestRepo(t)

		// Create a feature branch with commits
		runGit(t, dir, "checkout", "-b", "feature-1")
		writeFile(t, dir, "feature.txt", "feature content\n")
		runGit(t, dir, "add", ".")
		runGit(t, dir, "commit", "-m", "Add feature")

		// Merge back to main
		runGit(t, dir, "checkout", "main")
		runGit(t, dir, "merge", "--no-ff", "-m", "Merge feature-1", "feature-1")

		// Create and merge another branch
		runGit(t, dir, "checkout", "-b", "feature-2")
		writeFile(t, dir, "feature2.txt", "feature 2 content\n")
		runGit(t, dir, "add", ".")
		runGit(t, dir, "commit", "-m", "Add feature 2")
		runGit(t, dir, "checkout", "main")
		runGit(t, dir, "merge", "--no-ff", "-m", "Merge feature-2", "feature-2")

		runner := git.NewRunner()
		ctx := context.Background()

		hashes, err := runner.MergeCommits(ctx, dir, 10)

		require.NoError(t, err)
		assert.Len(t, hashes, 2)
		// Most recent merge first
		assert.NotEmpty(t, hashes[0])
		assert.NotEmpty(t, hashes[1])
	})

	t.Run("respects limit", func(t *testing.T) {
		t.Parallel()
		dir := setupTestRepo(t)

		// Create three merges
		for i := 1; i <= 3; i++ {
			branchName := fmt.Sprintf("feature-%d", i)
			runGit(t, dir, "checkout", "-b", branchName)
			writeFile(t, dir, fmt.Sprintf("file%d.txt", i), "content\n")
			runGit(t, dir, "add", ".")
			runGit(t, dir, "commit", "-m", "Commit on "+branchName)
			runGit(t, dir, "checkout", "main")
			runGit(t, dir, "merge", "--no-ff", "-m", "Merge "+branchName, branchName)
		}

		runner := git.NewRunner()
		ctx := context.Background()

		hashes, err := runner.MergeCommits(ctx, dir, 2)

		require.NoError(t, err)
		assert.Len(t, hashes, 2)
	})

	t.Run("returns empty slice when no merge commits", func(t *testing.T) {
		t.Parallel()
		dir := setupTestRepo(t)

		runner := git.NewRunner()
		ctx := context.Background()

		hashes, err := runner.MergeCommits(ctx, dir, 10)

		require.NoError(t, err)
		assert.Empty(t, hashes)
	})
}

func TestRunner_CommitsInRange(t *testing.T) {
	t.Parallel()

	t.Run("returns commits between base and head", func(t *testing.T) {
		t.Parallel()
		dir := setupTestRepo(t)

		// Create a feature branch with multiple commits
		runGit(t, dir, "checkout", "-b", "feature")
		writeFile(t, dir, "file1.txt", "content 1\n")
		runGit(t, dir, "add", ".")
		runGit(t, dir, "commit", "-m", "First feature commit")

		writeFile(t, dir, "file2.txt", "content 2\n")
		runGit(t, dir, "add", ".")
		runGit(t, dir, "commit", "-m", "Second feature commit")

		// Get the hashes we need
		head := strings.TrimSpace(runGit(t, dir, "rev-parse", "feature"))
		base := strings.TrimSpace(runGit(t, dir, "rev-parse", "main"))

		runner := git.NewRunner()
		ctx := context.Background()

		commits, err := runner.CommitsInRange(ctx, dir, base, head)

		require.NoError(t, err)
		assert.Len(t, commits, 2)
		// Commits are returned in reverse chronological order (newest first)
		assert.Equal(t, "Second feature commit", commits[0].Message)
		assert.Equal(t, "First feature commit", commits[1].Message)
		// Hashes should be valid 40-char hex strings
		assert.Len(t, commits[0].Hash, 40)
		assert.Len(t, commits[1].Hash, 40)
	})

	t.Run("returns empty slice when no commits in range", func(t *testing.T) {
		t.Parallel()
		dir := setupTestRepo(t)

		head := strings.TrimSpace(runGit(t, dir, "rev-parse", "HEAD"))

		runner := git.NewRunner()
		ctx := context.Background()

		commits, err := runner.CommitsInRange(ctx, dir, head, head)

		require.NoError(t, err)
		assert.Empty(t, commits)
	})
}

func TestRunner_DiffRange(t *testing.T) {
	t.Parallel()

	t.Run("returns diff between base and head", func(t *testing.T) {
		t.Parallel()
		dir := setupTestRepo(t)

		// Create a feature branch with changes
		runGit(t, dir, "checkout", "-b", "feature")
		writeFile(t, dir, "newfile.txt", "new content\n")
		runGit(t, dir, "add", ".")
		runGit(t, dir, "commit", "-m", "Add newfile")

		runner := git.NewRunner()
		ctx := context.Background()

		diff, err := runner.DiffRange(ctx, dir, "main", "feature")

		require.NoError(t, err)
		assert.Contains(t, diff, "newfile.txt")
		assert.Contains(t, diff, "+new content")
	})

	t.Run("returns empty diff when no changes", func(t *testing.T) {
		t.Parallel()
		dir := setupTestRepo(t)

		runner := git.NewRunner()
		ctx := context.Background()

		diff, err := runner.DiffRange(ctx, dir, "main", "main")

		require.NoError(t, err)
		assert.Empty(t, diff)
	})
}

func TestRunner_CurrentBranch(t *testing.T) {
	t.Parallel()

	t.Run("returns current branch name", func(t *testing.T) {
		t.Parallel()
		dir := setupTestRepo(t)

		runner := git.NewRunner()
		ctx := context.Background()

		branch, err := runner.CurrentBranch(ctx, dir)

		require.NoError(t, err)
		assert.Equal(t, "main", branch)
	})

	t.Run("returns feature branch when checked out", func(t *testing.T) {
		t.Parallel()
		dir := setupTestRepo(t)

		runGit(t, dir, "checkout", "-b", "my-feature")

		runner := git.NewRunner()
		ctx := context.Background()

		branch, err := runner.CurrentBranch(ctx, dir)

		require.NoError(t, err)
		assert.Equal(t, "my-feature", branch)
	})
}

func TestRunner_MergeBase(t *testing.T) {
	t.Parallel()

	t.Run("returns common ancestor of two refs", func(t *testing.T) {
		t.Parallel()
		dir := setupTestRepo(t)

		// Save main HEAD
		mainHead := strings.TrimSpace(runGit(t, dir, "rev-parse", "HEAD"))

		// Create a feature branch with changes
		runGit(t, dir, "checkout", "-b", "feature")
		writeFile(t, dir, "feature.txt", "feature content\n")
		runGit(t, dir, "add", ".")
		runGit(t, dir, "commit", "-m", "Feature commit")

		runner := git.NewRunner()
		ctx := context.Background()

		base, err := runner.MergeBase(ctx, dir, "main", "feature")

		require.NoError(t, err)
		assert.Equal(t, mainHead, base)
	})

	t.Run("returns same commit when refs are identical", func(t *testing.T) {
		t.Parallel()
		dir := setupTestRepo(t)

		head := strings.TrimSpace(runGit(t, dir, "rev-parse", "HEAD"))

		runner := git.NewRunner()
		ctx := context.Background()

		base, err := runner.MergeBase(ctx, dir, "main", "main")

		require.NoError(t, err)
		assert.Equal(t, head, base)
	})
}

func TestRunner_DefaultBranch(t *testing.T) {
	t.Parallel()

	t.Run("returns default branch from origin/HEAD", func(t *testing.T) {
		t.Parallel()
		// Create a "remote" repo
		remoteDir := t.TempDir()
		runGit(t, remoteDir, "init", "-b", "main", "--bare")

		// Create local repo and add remote
		dir := setupTestRepo(t)
		runGit(t, dir, "remote", "add", "origin", remoteDir)
		runGit(t, dir, "push", "-u", "origin", "main")

		// Set origin/HEAD to point to main (simulates what GitHub does)
		runGit(t, dir, "remote", "set-head", "origin", "main")

		runner := git.NewRunner()
		ctx := context.Background()

		branch, err := runner.DefaultBranch(ctx, dir)

		require.NoError(t, err)
		assert.Equal(t, "main", branch)
	})

	t.Run("returns master when that is the default branch", func(t *testing.T) {
		t.Parallel()
		// Create a "remote" repo with master as default
		remoteDir := t.TempDir()
		runGit(t, remoteDir, "init", "-b", "master", "--bare")

		// Create local repo with master
		dir := t.TempDir()
		runGit(t, dir, "init", "-b", "master")
		runGit(t, dir, "config", "user.email", "test@example.com")
		runGit(t, dir, "config", "user.name", "Test User")
		writeFile(t, dir, "README.md", "# Test\n")
		runGit(t, dir, "add", ".")
		runGit(t, dir, "commit", "-m", "Initial commit")
		runGit(t, dir, "remote", "add", "origin", remoteDir)
		runGit(t, dir, "push", "-u", "origin", "master")
		runGit(t, dir, "remote", "set-head", "origin", "master")

		runner := git.NewRunner()
		ctx := context.Background()

		branch, err := runner.DefaultBranch(ctx, dir)

		require.NoError(t, err)
		assert.Equal(t, "master", branch)
	})

	t.Run("returns error when no remote configured", func(t *testing.T) {
		t.Parallel()
		dir := setupTestRepo(t)
		// No remote added

		runner := git.NewRunner()
		ctx := context.Background()

		_, err := runner.DefaultBranch(ctx, dir)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no remote")
	})

	t.Run("returns error when remote exists but origin/HEAD not set", func(t *testing.T) {
		t.Parallel()
		// Create a "remote" repo
		remoteDir := t.TempDir()
		runGit(t, remoteDir, "init", "-b", "main", "--bare")

		// Create local repo, add remote, push, but DON'T set origin/HEAD
		dir := setupTestRepo(t)
		runGit(t, dir, "remote", "add", "origin", remoteDir)
		runGit(t, dir, "push", "-u", "origin", "main")
		// Note: NOT calling "git remote set-head origin main"

		runner := git.NewRunner()
		ctx := context.Background()

		_, err := runner.DefaultBranch(ctx, dir)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "origin/HEAD not set")
	})
}
