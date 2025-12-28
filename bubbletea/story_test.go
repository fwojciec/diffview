package bubbletea_test

import (
	"bytes"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/fwojciec/diffview"
	"github.com/fwojciec/diffview/bubbletea"
)

func TestStoryModel_BasicRendering(t *testing.T) {
	t.Parallel()

	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				OldPath:   "a/main.go",
				NewPath:   "b/main.go",
				Operation: diffview.FileModified,
				Hunks: []diffview.Hunk{
					{
						OldStart: 1,
						OldCount: 3,
						NewStart: 1,
						NewCount: 4,
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "package main"},
							{Type: diffview.LineDeleted, Content: "old line"},
							{Type: diffview.LineAdded, Content: "new line"},
							{Type: diffview.LineAdded, Content: "another new line"},
						},
					},
				},
			},
		},
	}

	story := &diffview.StoryClassification{
		ChangeType: "feature",
		Summary:    "Test change",
		Sections: []diffview.Section{
			{
				Role:  "core",
				Title: "Main Changes",
				Hunks: []diffview.HunkRef{
					{File: "main.go", HunkIndex: 0, Category: "core"},
				},
			},
		},
	}

	m := bubbletea.NewStoryModel(diff, story)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Wait for content to appear - file header should show the filename
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("main.go"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestStoryModel_QuitOnQ(t *testing.T) {
	t.Parallel()

	diff := &diffview.Diff{}
	story := &diffview.StoryClassification{}

	m := bubbletea.NewStoryModel(diff, story)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestStoryModel_SectionNavigation(t *testing.T) {
	t.Parallel()

	// Create many lines per file so sections don't all fit on screen
	firstFileLines := make([]diffview.Line, 20)
	for i := range firstFileLines {
		firstFileLines[i] = diffview.Line{Type: diffview.LineContext, Content: "first file content line"}
	}
	firstFileLines[0] = diffview.Line{Type: diffview.LineContext, Content: "FIRST_SECTION_MARKER"}

	secondFileLines := make([]diffview.Line, 20)
	for i := range secondFileLines {
		secondFileLines[i] = diffview.Line{Type: diffview.LineContext, Content: "second file content line"}
	}
	secondFileLines[0] = diffview.Line{Type: diffview.LineContext, Content: "SECOND_SECTION_MARKER"}

	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				NewPath:   "b/first.go",
				Operation: diffview.FileModified,
				Hunks: []diffview.Hunk{
					{
						OldStart: 1, OldCount: 20, NewStart: 1, NewCount: 20,
						Lines: firstFileLines,
					},
				},
			},
			{
				NewPath:   "b/second.go",
				Operation: diffview.FileModified,
				Hunks: []diffview.Hunk{
					{
						OldStart: 1, OldCount: 20, NewStart: 1, NewCount: 20,
						Lines: secondFileLines,
					},
				},
			},
		},
	}

	story := &diffview.StoryClassification{
		Sections: []diffview.Section{
			{
				Role:  "first",
				Title: "First Section",
				Hunks: []diffview.HunkRef{
					{File: "first.go", HunkIndex: 0, Category: "core"},
				},
			},
			{
				Role:  "second",
				Title: "Second Section",
				Hunks: []diffview.HunkRef{
					{File: "second.go", HunkIndex: 0, Category: "core"},
				},
			},
		},
	}

	m := bubbletea.NewStoryModel(diff, story)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Wait for initial render with first section marker visible
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("FIRST_SECTION_MARKER"))
	})

	// Press 's' to go to next section
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	// Wait for second section marker to become visible
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("SECOND_SECTION_MARKER"))
	})

	// Press 'S' to go back to previous section
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})

	// Wait for first section marker to be visible again
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("FIRST_SECTION_MARKER"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestStoryModel_HunkCollapsing(t *testing.T) {
	t.Parallel()

	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				NewPath:   "b/file.go",
				Operation: diffview.FileModified,
				Hunks: []diffview.Hunk{
					{
						OldStart: 1, OldCount: 5, NewStart: 1, NewCount: 5,
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "HUNK_CONTENT_MARKER"},
							{Type: diffview.LineContext, Content: "line 2"},
							{Type: diffview.LineContext, Content: "line 3"},
							{Type: diffview.LineContext, Content: "line 4"},
							{Type: diffview.LineContext, Content: "line 5"},
						},
					},
				},
			},
		},
	}

	story := &diffview.StoryClassification{
		Sections: []diffview.Section{
			{
				Role:  "core",
				Title: "Changes",
				Hunks: []diffview.HunkRef{
					{
						File:         "file.go",
						HunkIndex:    0,
						Category:     "core",
						Collapsed:    false,
						CollapseText: "Core changes summary",
					},
				},
			},
		},
	}

	m := bubbletea.NewStoryModel(diff, story)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Wait for initial render with hunk content visible
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("HUNK_CONTENT_MARKER"))
	})

	// Press 'o' to toggle collapse (should collapse the hunk)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})

	// Wait for collapse text to appear (hunk content should be hidden)
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasCollapseText := bytes.Contains(out, []byte("Core changes summary"))
		noContent := !bytes.Contains(out, []byte("HUNK_CONTENT_MARKER"))
		return hasCollapseText && noContent
	})

	// Press 'o' again to expand
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})

	// Wait for hunk content to reappear
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("HUNK_CONTENT_MARKER"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestStoryModel_NoiseHunksCollapsedByDefault(t *testing.T) {
	t.Parallel()

	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				NewPath:   "b/file.go",
				Operation: diffview.FileModified,
				Hunks: []diffview.Hunk{
					{
						OldStart: 1, OldCount: 2, NewStart: 1, NewCount: 2,
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "NOISE_CONTENT_MARKER"},
							{Type: diffview.LineContext, Content: "more noise"},
						},
					},
				},
			},
		},
	}

	story := &diffview.StoryClassification{
		Sections: []diffview.Section{
			{
				Role:  "noise",
				Title: "Noise",
				Hunks: []diffview.HunkRef{
					{
						File:         "file.go",
						HunkIndex:    0,
						Category:     "noise",
						CollapseText: "Noise hunk",
					},
				},
			},
		},
	}

	m := bubbletea.NewStoryModel(diff, story)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Noise hunks should be collapsed by default
	// The content should NOT be visible, but collapse text should be
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasCollapseText := bytes.Contains(out, []byte("Noise hunk"))
		noContent := !bytes.Contains(out, []byte("NOISE_CONTENT_MARKER"))
		return hasCollapseText && noContent
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestStoryModel_CategoryStyling(t *testing.T) {
	t.Parallel()

	// Create a diff with hunks of different categories
	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				NewPath:   "b/file.go",
				Operation: diffview.FileModified,
				Hunks: []diffview.Hunk{
					{
						OldStart: 1, OldCount: 1, NewStart: 1, NewCount: 1,
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "refactoring content"},
						},
					},
				},
			},
		},
	}

	story := &diffview.StoryClassification{
		Sections: []diffview.Section{
			{
				Role:  "supporting",
				Title: "Refactoring",
				Hunks: []diffview.HunkRef{
					{
						File:      "file.go",
						HunkIndex: 0,
						Category:  "refactoring", // This should be rendered with dimmed style
					},
				},
			},
		},
	}

	m := bubbletea.NewStoryModel(diff, story)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Wait for render - the content should be visible (just styled differently)
	// We verify the content is rendered, not the specific styling (styling is harder to test)
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("refactoring content"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestStoryModel_SectionIndicator_NoSections(t *testing.T) {
	t.Parallel()

	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				NewPath:   "b/file.go",
				Operation: diffview.FileModified,
				Hunks: []diffview.Hunk{
					{
						OldStart: 1, OldCount: 1, NewStart: 1, NewCount: 1,
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "content"},
						},
					},
				},
			},
		},
	}

	// Story with empty sections - should not show section indicator
	story := &diffview.StoryClassification{
		Sections: []diffview.Section{},
	}

	m := bubbletea.NewStoryModel(diff, story)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Wait for render - should show file/hunk but NOT "section X/Y:"
	// (Note: status bar still shows "s/S:section" in help, so we check for the pattern)
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasFile := bytes.Contains(out, []byte("file 1/1"))
		hasHunk := bytes.Contains(out, []byte("hunk 1/1"))
		noSectionIndicator := !bytes.Contains(out, []byte("section 1/"))
		return hasFile && hasHunk && noSectionIndicator
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestStoryModel_SectionIndicator(t *testing.T) {
	t.Parallel()

	// Create many lines per file so sections don't all fit on screen
	firstFileLines := make([]diffview.Line, 20)
	for i := range firstFileLines {
		firstFileLines[i] = diffview.Line{Type: diffview.LineContext, Content: "first file content line"}
	}

	secondFileLines := make([]diffview.Line, 20)
	for i := range secondFileLines {
		secondFileLines[i] = diffview.Line{Type: diffview.LineContext, Content: "second file content line"}
	}

	thirdFileLines := make([]diffview.Line, 20)
	for i := range thirdFileLines {
		thirdFileLines[i] = diffview.Line{Type: diffview.LineContext, Content: "third file content line"}
	}

	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				NewPath:   "b/first.go",
				Operation: diffview.FileModified,
				Hunks: []diffview.Hunk{
					{OldStart: 1, OldCount: 20, NewStart: 1, NewCount: 20, Lines: firstFileLines},
				},
			},
			{
				NewPath:   "b/second.go",
				Operation: diffview.FileModified,
				Hunks: []diffview.Hunk{
					{OldStart: 1, OldCount: 20, NewStart: 1, NewCount: 20, Lines: secondFileLines},
				},
			},
			{
				NewPath:   "b/third.go",
				Operation: diffview.FileModified,
				Hunks: []diffview.Hunk{
					{OldStart: 1, OldCount: 20, NewStart: 1, NewCount: 20, Lines: thirdFileLines},
				},
			},
		},
	}

	story := &diffview.StoryClassification{
		Sections: []diffview.Section{
			{
				Role:  "first",
				Title: "Core Changes",
				Hunks: []diffview.HunkRef{
					{File: "first.go", HunkIndex: 0, Category: "core"},
				},
			},
			{
				Role:  "second",
				Title: "Supporting Work",
				Hunks: []diffview.HunkRef{
					{File: "second.go", HunkIndex: 0, Category: "core"},
				},
			},
			{
				Role:  "third",
				Title: "Tests",
				Hunks: []diffview.HunkRef{
					{File: "third.go", HunkIndex: 0, Category: "core"},
				},
			},
		},
	}

	m := bubbletea.NewStoryModel(diff, story)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Wait for initial render - should show "section 1/3: Core Changes"
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("section 1/3: Core Changes"))
	})

	// Navigate to second section
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	// Should show "section 2/3: Supporting Work"
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("section 2/3: Supporting Work"))
	})

	// Navigate to third section
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	// Should show "section 3/3: Tests"
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("section 3/3: Tests"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}
