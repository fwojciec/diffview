package diffview_test

import (
	"testing"

	"github.com/fwojciec/diffstory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateClassification(t *testing.T) {
	t.Parallel()

	// Sample diff with two files, each with known hunk counts
	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				NewPath: "foo.go",
				Hunks:   make([]diffview.Hunk, 3), // indices 0, 1, 2 valid
			},
			{
				NewPath: "bar.go",
				Hunks:   make([]diffview.Hunk, 2), // indices 0, 1 valid
			},
		},
	}

	t.Run("valid classification passes", func(t *testing.T) {
		t.Parallel()

		classification := &diffview.StoryClassification{
			Sections: []diffview.Section{
				{
					Hunks: []diffview.HunkRef{
						{File: "foo.go", HunkIndex: 0},
						{File: "foo.go", HunkIndex: 2},
						{File: "bar.go", HunkIndex: 1},
					},
				},
			},
		}

		errs := diffview.ValidateClassification(diff, classification)
		assert.Empty(t, errs)
	})

	t.Run("invalid hunk index returns error", func(t *testing.T) {
		t.Parallel()

		classification := &diffview.StoryClassification{
			Sections: []diffview.Section{
				{
					Hunks: []diffview.HunkRef{
						{File: "foo.go", HunkIndex: 3}, // out of bounds (only 0-2 valid)
					},
				},
			},
		}

		errs := diffview.ValidateClassification(diff, classification)
		assert.Len(t, errs, 1)
		assert.Equal(t, "foo.go", errs[0].HunkRef.File)
		assert.Equal(t, 3, errs[0].HunkRef.HunkIndex)
		assert.Equal(t, diffview.ErrInvalidHunkIndex, errs[0].Reason)
	})

	t.Run("negative hunk index returns error", func(t *testing.T) {
		t.Parallel()

		classification := &diffview.StoryClassification{
			Sections: []diffview.Section{
				{
					Hunks: []diffview.HunkRef{
						{File: "foo.go", HunkIndex: -1},
					},
				},
			},
		}

		errs := diffview.ValidateClassification(diff, classification)
		assert.Len(t, errs, 1)
		assert.Equal(t, diffview.ErrInvalidHunkIndex, errs[0].Reason)
	})

	t.Run("file not found returns error", func(t *testing.T) {
		t.Parallel()

		classification := &diffview.StoryClassification{
			Sections: []diffview.Section{
				{
					Hunks: []diffview.HunkRef{
						{File: "nonexistent.go", HunkIndex: 0},
					},
				},
			},
		}

		errs := diffview.ValidateClassification(diff, classification)
		assert.Len(t, errs, 1)
		assert.Equal(t, "nonexistent.go", errs[0].HunkRef.File)
		assert.Equal(t, diffview.ErrFileNotFound, errs[0].Reason)
	})

	t.Run("multiple errors in different sections", func(t *testing.T) {
		t.Parallel()

		classification := &diffview.StoryClassification{
			Sections: []diffview.Section{
				{
					Hunks: []diffview.HunkRef{
						{File: "foo.go", HunkIndex: 10}, // out of bounds
					},
				},
				{
					Hunks: []diffview.HunkRef{
						{File: "missing.go", HunkIndex: 0}, // file not found
						{File: "bar.go", HunkIndex: 5},     // out of bounds
					},
				},
			},
		}

		errs := diffview.ValidateClassification(diff, classification)
		assert.Len(t, errs, 3)
	})

	t.Run("tracks section index in error", func(t *testing.T) {
		t.Parallel()

		classification := &diffview.StoryClassification{
			Sections: []diffview.Section{
				{
					Hunks: []diffview.HunkRef{
						{File: "foo.go", HunkIndex: 0}, // valid
					},
				},
				{
					Hunks: []diffview.HunkRef{
						{File: "foo.go", HunkIndex: 99}, // invalid
					},
				},
			},
		}

		errs := diffview.ValidateClassification(diff, classification)
		assert.Len(t, errs, 1)
		assert.Equal(t, 1, errs[0].Section)
	})
}

func TestValidationError_Error(t *testing.T) {
	t.Parallel()

	t.Run("invalid hunk index message", func(t *testing.T) {
		t.Parallel()

		err := diffview.ValidationError{
			Section:   0,
			HunkRef:   diffview.HunkRef{File: "foo.go", HunkIndex: 7},
			Reason:    diffview.ErrInvalidHunkIndex,
			HunkCount: 7,
		}

		msg := err.Error()
		assert.Contains(t, msg, "foo.go")
		assert.Contains(t, msg, "hunk_index 7")
		assert.Contains(t, msg, "valid: 0-6")
	})

	t.Run("file not found message", func(t *testing.T) {
		t.Parallel()

		err := diffview.ValidationError{
			Section: 0,
			HunkRef: diffview.HunkRef{File: "missing.go", HunkIndex: 0},
			Reason:  diffview.ErrFileNotFound,
		}

		msg := err.Error()
		assert.Contains(t, msg, "missing.go")
		assert.Contains(t, msg, "not found")
	})
}

// TestValidateClassification_PR83Case tests the real-world case from PR #83
// where the LLM incorrectly referenced hunk_index 7 for bubbletea/story.go
// which only had 7 hunks (valid indices 0-6).
func TestValidateClassification_PR83Case(t *testing.T) {
	t.Parallel()

	// Simplified diff matching the structure from eval-curated.jsonl case 2
	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{NewPath: "bubbletea/story.go", Hunks: make([]diffview.Hunk, 7)},        // 0-6 valid
			{NewPath: "bubbletea/story_test.go", Hunks: make([]diffview.Hunk, 8)},   // 0-7 valid
			{NewPath: "bubbletea/story_keymap.go", Hunks: make([]diffview.Hunk, 2)}, // 0-1 valid
			{NewPath: ".beads/issues.jsonl", Hunks: make([]diffview.Hunk, 1)},       // 0 valid
		},
	}

	// Classification with the bug: story.go hunk_index 7 is invalid
	classification := &diffview.StoryClassification{
		ChangeType: "feature",
		Narrative:  "core-periphery",
		Summary:    "Simplify collapse UX",
		Sections: []diffview.Section{
			{
				Role:  "core",
				Title: "Collapse Logic Refinement",
				Hunks: []diffview.HunkRef{
					{File: "bubbletea/story.go", HunkIndex: 0, Category: "core"},
					{File: "bubbletea/story.go", HunkIndex: 1, Category: "core"},
				},
			},
			{
				Role:  "supporting",
				Title: "Keybinding Update",
				Hunks: []diffview.HunkRef{
					{File: "bubbletea/story_keymap.go", HunkIndex: 0, Category: "core"},
					{File: "bubbletea/story_keymap.go", HunkIndex: 1, Category: "core"},
				},
			},
			{
				Role:  "test",
				Title: "Behavioral Validation",
				Hunks: []diffview.HunkRef{
					{File: "bubbletea/story_test.go", HunkIndex: 0, Category: "core"},
					{File: "bubbletea/story_test.go", HunkIndex: 7, Category: "core"}, // Valid (8 hunks in test file)
				},
			},
			{
				Role:  "supporting",
				Title: "UI Feedback",
				Hunks: []diffview.HunkRef{
					{File: "bubbletea/story.go", HunkIndex: 7, Category: "systematic"}, // INVALID! Only 0-6 valid
				},
			},
		},
	}

	errs := diffview.ValidateClassification(diff, classification)

	require.Len(t, errs, 1, "should catch exactly one invalid hunk reference")
	assert.Equal(t, "bubbletea/story.go", errs[0].HunkRef.File)
	assert.Equal(t, 7, errs[0].HunkRef.HunkIndex)
	assert.Equal(t, diffview.ErrInvalidHunkIndex, errs[0].Reason)
	assert.Equal(t, 7, errs[0].HunkCount, "should report file has 7 hunks")
	assert.Equal(t, 3, errs[0].Section, "should be in section 3 (UI Feedback)")

	// Verify error message is helpful
	errMsg := errs[0].Error()
	assert.Contains(t, errMsg, "story.go")
	assert.Contains(t, errMsg, "hunk_index 7")
	assert.Contains(t, errMsg, "valid: 0-6")
}
