package mock

import (
	"context"

	"github.com/fwojciec/diffview"
)

// Compile-time interface verification.
var _ diffview.Viewer = (*Viewer)(nil)

// Viewer is a mock implementation of diffview.Viewer.
type Viewer struct {
	ViewFn func(ctx context.Context, diff *diffview.Diff) error
}

func (v *Viewer) View(ctx context.Context, diff *diffview.Diff) error {
	return v.ViewFn(ctx, diff)
}
