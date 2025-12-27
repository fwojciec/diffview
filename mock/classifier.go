package mock

import (
	"context"

	"github.com/fwojciec/diffview"
)

// Compile-time interface verification.
var _ diffview.StoryClassifier = (*StoryClassifier)(nil)

// StoryClassifier is a mock implementation of diffview.StoryClassifier.
type StoryClassifier struct {
	ClassifyFn func(ctx context.Context, input diffview.ClassificationInput) (*diffview.StoryClassification, error)
}

func (c *StoryClassifier) Classify(ctx context.Context, input diffview.ClassificationInput) (*diffview.StoryClassification, error) {
	return c.ClassifyFn(ctx, input)
}
