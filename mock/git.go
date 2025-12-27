package mock

import (
	"context"

	"github.com/fwojciec/diffview"
)

// Compile-time interface verification.
var _ diffview.GitRunner = (*GitRunner)(nil)

// GitRunner is a mock implementation of diffview.GitRunner.
type GitRunner struct {
	LogFn  func(ctx context.Context, repoPath string, limit int) ([]string, error)
	ShowFn func(ctx context.Context, repoPath string, hash string) (string, error)
}

func (g *GitRunner) Log(ctx context.Context, repoPath string, limit int) ([]string, error) {
	return g.LogFn(ctx, repoPath, limit)
}

func (g *GitRunner) Show(ctx context.Context, repoPath string, hash string) (string, error) {
	return g.ShowFn(ctx, repoPath, hash)
}
