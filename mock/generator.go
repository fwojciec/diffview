package mock

import (
	"context"

	"github.com/fwojciec/diffview"
)

// Compile-time interface verification.
var _ diffview.StoryGenerator = (*StoryGenerator)(nil)

// StoryGenerator is a mock implementation of diffview.StoryGenerator.
type StoryGenerator struct {
	GenerateFn func(ctx context.Context, hunks []diffview.AnnotatedHunk) (*diffview.DiffAnalysis, error)
}

func (g *StoryGenerator) Generate(ctx context.Context, hunks []diffview.AnnotatedHunk) (*diffview.DiffAnalysis, error) {
	return g.GenerateFn(ctx, hunks)
}
