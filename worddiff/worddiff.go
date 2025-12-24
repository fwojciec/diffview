// Package worddiff computes word-level differences between strings.
package worddiff

import (
	"github.com/fwojciec/diffview"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// Compile-time interface verification.
var _ diffview.WordDiffer = (*Differ)(nil)

// Differ computes word-level differences between strings.
type Differ struct {
	dmp *diffmatchpatch.DiffMatchPatch
}

// NewDiffer creates a new Differ.
func NewDiffer() *Differ {
	return &Differ{
		dmp: diffmatchpatch.New(),
	}
}

// Diff returns segments for both the old and new strings,
// marking which portions changed between them.
func (d *Differ) Diff(old, new string) (oldSegs, newSegs []diffview.Segment) {
	// Handle empty strings
	if old == "" && new == "" {
		return nil, nil
	}
	if old == "" {
		return nil, []diffview.Segment{{Text: new, Changed: true}}
	}
	if new == "" {
		return []diffview.Segment{{Text: old, Changed: true}}, nil
	}

	// Compute character-level diff
	diffs := d.dmp.DiffMain(old, new, false)
	diffs = d.dmp.DiffCleanupSemantic(diffs)

	// Convert diffs to segments for old and new strings
	oldSegs = diffsToSegments(diffs, diffmatchpatch.DiffDelete)
	newSegs = diffsToSegments(diffs, diffmatchpatch.DiffInsert)

	return oldSegs, newSegs
}

// diffsToSegments converts diff operations to segments for one side (old or new).
// changeOp specifies which operation type represents "changed" text for this side
// (DiffDelete for old side, DiffInsert for new side).
func diffsToSegments(diffs []diffmatchpatch.Diff, changeOp diffmatchpatch.Operation) []diffview.Segment {
	var segments []diffview.Segment

	for _, diff := range diffs {
		switch diff.Type {
		case diffmatchpatch.DiffEqual:
			// Equal text appears in both old and new, unchanged
			segments = append(segments, diffview.Segment{
				Text:    diff.Text,
				Changed: false,
			})
		case changeOp:
			// This is the "changed" operation for this side
			segments = append(segments, diffview.Segment{
				Text:    diff.Text,
				Changed: true,
			})
			// The other operation (Insert for old, Delete for new) is skipped
			// because that text doesn't exist in this side's string
		}
	}

	// Merge adjacent segments with the same Changed status
	return mergeSegments(segments)
}

// mergeSegments combines adjacent segments with the same Changed status.
func mergeSegments(segments []diffview.Segment) []diffview.Segment {
	if len(segments) == 0 {
		return nil
	}

	merged := make([]diffview.Segment, 0, len(segments))
	current := segments[0]

	for i := 1; i < len(segments); i++ {
		if segments[i].Changed == current.Changed {
			// Same status, merge text
			current.Text += segments[i].Text
		} else {
			// Different status, start new segment
			merged = append(merged, current)
			current = segments[i]
		}
	}
	merged = append(merged, current)

	return merged
}
