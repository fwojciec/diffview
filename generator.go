package diffview

import "context"

// StoryGenerator generates narrative analyses for code changes.
type StoryGenerator interface {
	// Generate creates a DiffAnalysis from annotated hunks.
	Generate(ctx context.Context, hunks []AnnotatedHunk) (*DiffAnalysis, error)
}
