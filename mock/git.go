package mock

import (
	"context"

	"github.com/fwojciec/diffview"
)

// Compile-time interface verification.
var _ diffview.GitRunner = (*GitRunner)(nil)

// GitRunner is a mock implementation of diffview.GitRunner.
type GitRunner struct {
	// Deprecated methods (for commit-level extraction)
	LogFn     func(ctx context.Context, repoPath string, limit int) ([]string, error)
	ShowFn    func(ctx context.Context, repoPath string, hash string) (string, error)
	MessageFn func(ctx context.Context, repoPath string, hash string) (string, error)

	// PR-level extraction methods
	MergeCommitsFn   func(ctx context.Context, repoPath string, limit int) ([]string, error)
	CommitsInRangeFn func(ctx context.Context, repoPath, base, head string) ([]diffview.CommitBrief, error)
	DiffRangeFn      func(ctx context.Context, repoPath, base, head string) (string, error)
	CurrentBranchFn  func(ctx context.Context, repoPath string) (string, error)
	MergeBaseFn      func(ctx context.Context, repoPath, ref1, ref2 string) (string, error)
	DefaultBranchFn  func(ctx context.Context, repoPath string) (string, error)
}

func (g *GitRunner) Log(ctx context.Context, repoPath string, limit int) ([]string, error) {
	return g.LogFn(ctx, repoPath, limit)
}

func (g *GitRunner) Show(ctx context.Context, repoPath string, hash string) (string, error) {
	return g.ShowFn(ctx, repoPath, hash)
}

func (g *GitRunner) Message(ctx context.Context, repoPath string, hash string) (string, error) {
	return g.MessageFn(ctx, repoPath, hash)
}

func (g *GitRunner) MergeCommits(ctx context.Context, repoPath string, limit int) ([]string, error) {
	return g.MergeCommitsFn(ctx, repoPath, limit)
}

func (g *GitRunner) CommitsInRange(ctx context.Context, repoPath, base, head string) ([]diffview.CommitBrief, error) {
	return g.CommitsInRangeFn(ctx, repoPath, base, head)
}

func (g *GitRunner) DiffRange(ctx context.Context, repoPath, base, head string) (string, error) {
	return g.DiffRangeFn(ctx, repoPath, base, head)
}

func (g *GitRunner) CurrentBranch(ctx context.Context, repoPath string) (string, error) {
	return g.CurrentBranchFn(ctx, repoPath)
}

func (g *GitRunner) MergeBase(ctx context.Context, repoPath, ref1, ref2 string) (string, error) {
	return g.MergeBaseFn(ctx, repoPath, ref1, ref2)
}

func (g *GitRunner) DefaultBranch(ctx context.Context, repoPath string) (string, error) {
	return g.DefaultBranchFn(ctx, repoPath)
}
