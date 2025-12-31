package bubbletea_test

import (
	"bytes"
	"io"
	"sync"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/fwojciec/diffstory"
	"github.com/fwojciec/diffstory/bubbletea"
	dv "github.com/fwojciec/diffstory/lipgloss"
	"github.com/muesli/termenv"
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

func TestStoryModel_NoToggleCollapseKey(t *testing.T) {
	t.Parallel()

	// Verify ToggleCollapseAll is bound to 'z'.
	// Note: The 'o' keybinding (ToggleCollapse) was removed from the struct,
	// which is enforced at compile time - any code referencing the field won't compile.
	keymap := bubbletea.DefaultStoryKeyMap()

	if keymap.ToggleCollapseAll.Help().Key != "z" {
		t.Errorf("expected ToggleCollapseAll to be bound to 'z', got %q", keymap.ToggleCollapseAll.Help().Key)
	}
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

func TestStoryModel_ZKeyOnlyTogglesLLMCollapsedHunks(t *testing.T) {
	t.Parallel()

	// Create diff with two hunks in one section:
	// - Hunk 0: LLM-collapsed (Collapsed: true) - should toggle with 'z'
	// - Hunk 1: Never collapsed (Collapsed: false) - should NOT toggle with 'z'
	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				NewPath:   "b/file.go",
				Operation: diffview.FileModified,
				Hunks: []diffview.Hunk{
					{
						OldStart: 1, OldCount: 2, NewStart: 1, NewCount: 2,
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "COLLAPSED_HUNK_CONTENT"},
							{Type: diffview.LineContext, Content: "more collapsed content"},
						},
					},
					{
						OldStart: 10, OldCount: 2, NewStart: 10, NewCount: 2,
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "NORMAL_HUNK_CONTENT"},
							{Type: diffview.LineContext, Content: "more normal content"},
						},
					},
				},
			},
		},
	}

	story := &diffview.StoryClassification{
		Sections: []diffview.Section{
			{
				Role:  "mixed",
				Title: "Mixed Hunks",
				Hunks: []diffview.HunkRef{
					{
						File:         "file.go",
						HunkIndex:    0,
						Category:     "noise",
						Collapsed:    true, // LLM says collapse this
						CollapseText: "Collapsed summary",
					},
					{
						File:         "file.go",
						HunkIndex:    1,
						Category:     "core",
						Collapsed:    false, // LLM says show this expanded
						CollapseText: "Core summary",
					},
				},
			},
		},
	}

	m := bubbletea.NewStoryModel(diff, story)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 40),
	)

	// Initial state:
	// - Collapsed hunk: shows "Collapsed summary", no content
	// - Normal hunk: shows "NORMAL_HUNK_CONTENT" (expanded)
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		collapsedHunkHidden := !bytes.Contains(out, []byte("COLLAPSED_HUNK_CONTENT"))
		collapsedSummaryVisible := bytes.Contains(out, []byte("Collapsed summary"))
		normalHunkExpanded := bytes.Contains(out, []byte("NORMAL_HUNK_CONTENT"))
		return collapsedHunkHidden && collapsedSummaryVisible && normalHunkExpanded
	})

	// Press 'z' to toggle LLM-collapsed hunks (expands the collapsed one)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})

	// After first 'z':
	// - Previously collapsed hunk: now expanded, shows content
	// - Normal hunk: still expanded (unchanged)
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		collapsedHunkNowExpanded := bytes.Contains(out, []byte("COLLAPSED_HUNK_CONTENT"))
		normalHunkStillExpanded := bytes.Contains(out, []byte("NORMAL_HUNK_CONTENT"))
		return collapsedHunkNowExpanded && normalHunkStillExpanded
	})

	// Press 'z' again to toggle back (collapses the LLM-collapsed one)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})

	// After second 'z':
	// - LLM-collapsed hunk: back to collapsed
	// - Normal hunk: still expanded (never affected by 'z')
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		collapsedHunkHidden := !bytes.Contains(out, []byte("COLLAPSED_HUNK_CONTENT"))
		collapsedSummaryVisible := bytes.Contains(out, []byte("Collapsed summary"))
		normalHunkStillExpanded := bytes.Contains(out, []byte("NORMAL_HUNK_CONTENT"))
		return collapsedHunkHidden && collapsedSummaryVisible && normalHunkStillExpanded
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
						Category:  "refactoring", // Dimmed when collapsed, full styling when expanded
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

func TestStoryModel_IntroSlide_StartsAtIntro(t *testing.T) {
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
							{Type: diffview.LineContext, Content: "CODE_CONTENT"},
						},
					},
				},
			},
		},
	}

	story := &diffview.StoryClassification{
		Summary: "Test summary for intro slide",
		Sections: []diffview.Section{
			{
				Role:  "core",
				Title: "Core Changes",
				Hunks: []diffview.HunkRef{
					{File: "file.go", HunkIndex: 0, Category: "core"},
				},
			},
		},
	}

	// With intro slide enabled, should start at intro (not code)
	m := bubbletea.NewStoryModel(diff, story, bubbletea.WithIntroSlide())
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Should show intro slide content (summary and overview indicator)
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasOverview := bytes.Contains(out, []byte("overview"))
		hasSummary := bytes.Contains(out, []byte("Test summary for intro slide"))
		noCode := !bytes.Contains(out, []byte("CODE_CONTENT"))
		return hasOverview && hasSummary && noCode
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestStoryModel_WithoutIntroSlide_StartsAtCodeSection(t *testing.T) {
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
							{Type: diffview.LineContext, Content: "CODE_CONTENT"},
						},
					},
				},
			},
		},
	}

	story := &diffview.StoryClassification{
		Summary: "Test summary",
		Sections: []diffview.Section{
			{
				Role:  "core",
				Title: "Core Changes",
				Hunks: []diffview.HunkRef{
					{File: "file.go", HunkIndex: 0, Category: "core"},
				},
			},
		},
	}

	// WITHOUT WithIntroSlide - should start at code section
	m := bubbletea.NewStoryModel(diff, story)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Should show code content and "section 1/1: Core Changes" (no intro)
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasCode := bytes.Contains(out, []byte("CODE_CONTENT"))
		hasSection := bytes.Contains(out, []byte("section 1/1: Core Changes"))
		noOverview := !bytes.Contains(out, []byte("overview"))
		return hasCode && hasSection && noOverview
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestStoryModel_IntroSlide_NavigationToCodeAndBack(t *testing.T) {
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
							{Type: diffview.LineContext, Content: "CODE_CONTENT"},
						},
					},
				},
			},
		},
	}

	story := &diffview.StoryClassification{
		Summary: "Test summary",
		Sections: []diffview.Section{
			{
				Role:  "core",
				Title: "Core Changes",
				Hunks: []diffview.HunkRef{
					{File: "file.go", HunkIndex: 0, Category: "core"},
				},
			},
		},
	}

	m := bubbletea.NewStoryModel(diff, story, bubbletea.WithIntroSlide())
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Should start at intro with "section 1/2: overview"
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("section 1/2: overview"))
	})

	// Press 's' to advance to first code section
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	// Should show code content and "section 2/2: Core Changes"
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasCode := bytes.Contains(out, []byte("CODE_CONTENT"))
		hasSection := bytes.Contains(out, []byte("section 2/2: Core Changes"))
		return hasCode && hasSection
	})

	// Press 'S' to go back to intro
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})

	// Should be back at intro with "section 1/2: overview"
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasOverview := bytes.Contains(out, []byte("section 1/2: overview"))
		noCode := !bytes.Contains(out, []byte("CODE_CONTENT"))
		return hasOverview && noCode
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestStoryModel_SectionFiltering(t *testing.T) {
	t.Parallel()

	// Create diff with two files, each belonging to a different section
	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				NewPath:   "b/first.go",
				Operation: diffview.FileModified,
				Hunks: []diffview.Hunk{
					{
						OldStart: 1, OldCount: 1, NewStart: 1, NewCount: 1,
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "SECTION_ONE_CONTENT"},
						},
					},
				},
			},
			{
				NewPath:   "b/second.go",
				Operation: diffview.FileModified,
				Hunks: []diffview.Hunk{
					{
						OldStart: 1, OldCount: 1, NewStart: 1, NewCount: 1,
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "SECTION_TWO_CONTENT"},
						},
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

	// Wait for initial render - should show only section 1 content
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasSection1Content := bytes.Contains(out, []byte("SECTION_ONE_CONTENT"))
		noSection2Content := !bytes.Contains(out, []byte("SECTION_TWO_CONTENT"))
		hasSection1Indicator := bytes.Contains(out, []byte("section 1/2"))
		return hasSection1Content && noSection2Content && hasSection1Indicator
	})

	// Navigate to section 2
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	// Should show only section 2 content
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasSection2Content := bytes.Contains(out, []byte("SECTION_TWO_CONTENT"))
		noSection1Content := !bytes.Contains(out, []byte("SECTION_ONE_CONTENT"))
		hasSection2Indicator := bytes.Contains(out, []byte("section 2/2"))
		return hasSection2Content && noSection1Content && hasSection2Indicator
	})

	// Navigate back to section 1
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})

	// Should show only section 1 content again
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasSection1Content := bytes.Contains(out, []byte("SECTION_ONE_CONTENT"))
		noSection2Content := !bytes.Contains(out, []byte("SECTION_TWO_CONTENT"))
		hasSection1Indicator := bytes.Contains(out, []byte("section 1/2"))
		return hasSection1Content && noSection2Content && hasSection1Indicator
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

// storyTrueColorRenderer creates a lipgloss renderer that outputs true colors.
func storyTrueColorRenderer() *lipgloss.Renderer {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.TrueColor)
	return r
}

// storyMockTokenizer implements diffview.Tokenizer for testing.
type storyMockTokenizer struct {
	TokenizeFn      func(language, source string) []diffview.Token
	TokenizeLinesFn func(language, source string) [][]diffview.Token
}

func (m *storyMockTokenizer) Tokenize(language, source string) []diffview.Token {
	return m.TokenizeFn(language, source)
}

func (m *storyMockTokenizer) TokenizeLines(language, source string) [][]diffview.Token {
	if m.TokenizeLinesFn != nil {
		return m.TokenizeLinesFn(language, source)
	}
	return nil
}

// storyMockLanguageDetector implements diffview.LanguageDetector for testing.
type storyMockLanguageDetector struct {
	DetectFromPathFn func(path string) string
}

func (m *storyMockLanguageDetector) DetectFromPath(path string) string {
	return m.DetectFromPathFn(path)
}

func TestStoryModel_ExpandedHunksGetFullStyling(t *testing.T) {
	t.Parallel()

	// Create a diff with Go code that will be syntax highlighted
	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				OldPath:   "a/main.go",
				NewPath:   "b/main.go",
				Operation: diffview.FileModified,
				Hunks: []diffview.Hunk{
					{
						OldStart: 1, OldCount: 1, NewStart: 1, NewCount: 2,
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "package main", OldLineNum: 1, NewLineNum: 1},
							{Type: diffview.LineAdded, Content: "func main() {}", OldLineNum: 0, NewLineNum: 2},
						},
					},
				},
			},
		},
	}

	// Story with refactoring category (starts collapsed and dimmed)
	story := &diffview.StoryClassification{
		Sections: []diffview.Section{
			{
				Role:  "supporting",
				Title: "Refactoring",
				Hunks: []diffview.HunkRef{
					{
						File:         "main.go",
						HunkIndex:    0,
						Category:     "refactoring",
						Collapsed:    true,
						CollapseText: "COLLAPSED_SUMMARY",
					},
				},
			},
		},
	}

	// Use TestTheme with predictable colors:
	// - Keyword: #ff00ff (magenta) -> "38;2;255;0;255"
	// - Added background: 15% blend of #00ff00 into #000000 -> "48;2;0;38;0"
	theme := dv.TestTheme()

	// Mock tokenizer that returns magenta-colored keywords
	tokenizer := &storyMockTokenizer{
		TokenizeLinesFn: func(language, source string) [][]diffview.Token {
			if language != "Go" {
				return nil
			}
			// For "package main\nfunc main() {}" return tokens for both lines
			if source == "package main\nfunc main() {}" {
				return [][]diffview.Token{
					{
						{Text: "package", Style: diffview.Style{Foreground: "#ff00ff", Bold: true}},
						{Text: " ", Style: diffview.Style{}},
						{Text: "main", Style: diffview.Style{}},
					},
					{
						{Text: "func", Style: diffview.Style{Foreground: "#ff00ff", Bold: true}},
						{Text: " ", Style: diffview.Style{}},
						{Text: "main", Style: diffview.Style{Foreground: "#0000ff"}},
						{Text: "()", Style: diffview.Style{}},
						{Text: " {}", Style: diffview.Style{}},
					},
				}
			}
			return nil
		},
	}

	// Mock detector that returns "Go" for .go files
	detector := &storyMockLanguageDetector{
		DetectFromPathFn: func(path string) string {
			if len(path) >= 3 && path[len(path)-3:] == ".go" {
				return "Go"
			}
			return ""
		},
	}

	m := bubbletea.NewStoryModel(diff, story,
		bubbletea.WithStoryTheme(theme),
		bubbletea.WithStoryRenderer(storyTrueColorRenderer()),
		bubbletea.WithStoryLanguageDetector(detector),
		bubbletea.WithStoryTokenizer(tokenizer),
	)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Step 1: Verify collapsed state shows collapse text, not the code
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasCollapseText := bytes.Contains(out, []byte("COLLAPSED_SUMMARY"))
		noCodeContent := !bytes.Contains(out, []byte("func main"))
		return hasCollapseText && noCodeContent
	})

	// Step 2: Press 'z' to toggle all LLM-collapsed hunks (expands them)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})

	// Step 3: Verify expanded state has full styling
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		// Code content is now visible (tokens are styled separately, so check individually)
		hasFuncKeyword := bytes.Contains(out, []byte("func"))
		hasMainIdent := bytes.Contains(out, []byte("main"))

		// Syntax highlighting: magenta keyword "func" -> RGB(255, 0, 255)
		// May appear as "1;38;2;255;0;255" (bold+fg) or "38;2;255;0;255" (fg only)
		hasSyntaxHighlighting := bytes.Contains(out, []byte("255;0;255"))

		// Added line background: 15% green blend -> RGB(0, 38, 0) -> "48;2;0;38;0"
		hasAddedBackground := bytes.Contains(out, []byte("48;2;0;38;0"))

		return hasFuncKeyword && hasMainIdent && hasSyntaxHighlighting && hasAddedBackground
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestStoryModel_SaveCaseToEvalDataset(t *testing.T) {
	t.Parallel()

	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				NewPath:   "main.go",
				Operation: diffview.FileModified,
				Hunks: []diffview.Hunk{
					{
						OldStart: 1, OldCount: 1, NewStart: 1, NewCount: 2,
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "package main"},
							{Type: diffview.LineAdded, Content: "// new comment"},
						},
					},
				},
			},
		},
	}

	story := &diffview.StoryClassification{
		ChangeType: "feature",
		Summary:    "Added a comment",
		Sections: []diffview.Section{
			{Role: "core", Title: "Main Changes", Hunks: []diffview.HunkRef{{File: "main.go", HunkIndex: 0}}},
		},
	}

	input := diffview.ClassificationInput{
		Repo:   "test-repo",
		Branch: "feature-branch",
		Commits: []diffview.CommitBrief{
			{Hash: "abc123", Message: "Add comment"},
		},
		Diff: *diff,
	}

	// Create mock saver
	mockSaver := &storyCaseSaver{}
	m := bubbletea.NewStoryModel(diff, story,
		bubbletea.WithStoryInput(input),
		bubbletea.WithStoryCaseSaver(mockSaver, "/tmp/curated.jsonl"),
	)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(100, 40),
	)

	// Wait for model to be ready
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("main.go"))
	})

	// Press 'e' to save case
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

	// Wait for save to happen
	teatest.WaitFor(t, tm.Output(), func(_ []byte) bool {
		return mockSaver.Saved()
	})

	// Verify the saved case
	savedCase := mockSaver.SavedCase()
	if savedCase == nil {
		t.Fatal("expected case to be saved")
	}
	if savedCase.Input.Repo != "test-repo" {
		t.Errorf("expected repo 'test-repo', got %q", savedCase.Input.Repo)
	}
	if savedCase.Story.ChangeType != "feature" {
		t.Errorf("expected change type 'feature', got %q", savedCase.Story.ChangeType)
	}
	if mockSaver.SavedPath() != "/tmp/curated.jsonl" {
		t.Errorf("expected path '/tmp/curated.jsonl', got %q", mockSaver.SavedPath())
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

// storyCaseSaver is a mock for testing case saving in StoryModel.
type storyCaseSaver struct {
	mu        sync.Mutex
	saved     bool
	savedCase *diffview.EvalCase
	savedPath string
}

func (s *storyCaseSaver) Save(path string, c diffview.EvalCase) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.saved = true
	s.savedCase = &c
	s.savedPath = path
	return nil
}

func (s *storyCaseSaver) Saved() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saved
}

func (s *storyCaseSaver) SavedCase() *diffview.EvalCase {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.savedCase
}

func (s *storyCaseSaver) SavedPath() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.savedPath
}

func TestStoryModel_FilteredDiffUsesOriginalHunkIndices(t *testing.T) {
	t.Parallel()

	// Bug: When a section references non-first hunks from a file, collapsed state,
	// categories, and collapse text are incorrectly looked up because render uses
	// filtered slice indices instead of original hunk indices.
	//
	// This test creates:
	// - A file with 2 hunks (indices 0 and 1)
	// - A section that only references hunk 1 (not hunk 0)
	// - Hunk 0 has category "systematic" and collapse text "Hunk zero text"
	// - Hunk 1 has category "core" and collapse text "Hunk one text"
	//
	// The bug: When rendering the section, the filtered diff only contains hunk 1,
	// but at position 0 in the filtered slice. The render code uses the position (0)
	// to lookup the category/collapse text, getting hunk 0's data instead of hunk 1's.

	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				NewPath:   "b/file.go",
				Operation: diffview.FileModified,
				Hunks: []diffview.Hunk{
					{
						// Hunk 0: NOT in section
						OldStart: 1, OldCount: 2, NewStart: 1, NewCount: 2,
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "HUNK_ZERO_CONTENT"},
							{Type: diffview.LineContext, Content: "more hunk zero"},
						},
					},
					{
						// Hunk 1: IS in section
						OldStart: 100, OldCount: 2, NewStart: 100, NewCount: 2,
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "HUNK_ONE_CONTENT"},
							{Type: diffview.LineContext, Content: "more hunk one"},
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
				Title: "Core Changes",
				Hunks: []diffview.HunkRef{
					// Section only includes hunk 1, not hunk 0
					{
						File:         "file.go",
						HunkIndex:    1, // Original index is 1
						Category:     "core",
						Collapsed:    true,
						CollapseText: "Hunk one text", // Should show this
					},
				},
			},
		},
	}

	// Also register hunk 0's metadata (which the bug would incorrectly use)
	// We do this by making the story aware of hunk 0 in a different section
	// Actually, the maps are built from ALL hunks in the story, so we need
	// to ensure hunk 0's data exists in the maps too. Let's add a second section.
	story.Sections = append(story.Sections, diffview.Section{
		Role:  "supporting",
		Title: "Supporting",
		Hunks: []diffview.HunkRef{
			{
				File:         "file.go",
				HunkIndex:    0,
				Category:     "systematic",
				Collapsed:    true,
				CollapseText: "Hunk zero text", // Bug would incorrectly show this
			},
		},
	})

	m := bubbletea.NewStoryModel(diff, story)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// On section 1, we should see:
	// - "Hunk one text" (the collapse text for hunk 1)
	// - NOT "Hunk zero text" (that's for hunk 0, which isn't in this section)
	// - NOT "HUNK_ZERO_CONTENT" (hunk 0 content, filtered out)
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasCorrectCollapseText := bytes.Contains(out, []byte("Hunk one text"))
		noWrongCollapseText := !bytes.Contains(out, []byte("Hunk zero text"))
		noHunkZeroContent := !bytes.Contains(out, []byte("HUNK_ZERO_CONTENT"))
		return hasCorrectCollapseText && noWrongCollapseText && noHunkZeroContent
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestStoryModel_IntroSlide_ShowsChangeTypePrefix(t *testing.T) {
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

	story := &diffview.StoryClassification{
		ChangeType: "bugfix",
		Summary:    "Fix null pointer exception in handler",
		Sections: []diffview.Section{
			{
				Role:  "fix",
				Title: "The Fix",
				Hunks: []diffview.HunkRef{
					{File: "file.go", HunkIndex: 0, Category: "core"},
				},
			},
		},
	}

	m := bubbletea.NewStoryModel(diff, story, bubbletea.WithIntroSlide())
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Should show summary with [bugfix] prefix
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("[bugfix]")) &&
			bytes.Contains(out, []byte("Fix null pointer exception in handler"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestStoryModel_IntroSlide_ShowsNarrativeDiagram(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		narrative string
		sections  []diffview.Section
		expected  []string // all must be present
	}{
		{
			name:      "cause-effect shows role diagram",
			narrative: "cause-effect",
			sections: []diffview.Section{
				{Role: "problem", Title: "The Bug", Hunks: []diffview.HunkRef{{File: "file.go", HunkIndex: 0}}},
				{Role: "fix", Title: "The Fix", Hunks: []diffview.HunkRef{{File: "file.go", HunkIndex: 0}}},
				{Role: "test", Title: "Tests", Hunks: []diffview.HunkRef{{File: "file.go", HunkIndex: 0}}},
			},
			expected: []string{"problem", "fix", "test", "→"},
		},
		{
			name:      "core-periphery shows text explanation",
			narrative: "core-periphery",
			sections: []diffview.Section{
				{Role: "core", Title: "Core", Hunks: []diffview.HunkRef{{File: "file.go", HunkIndex: 0}}},
			},
			expected: []string{"core change → ripple effects"},
		},
		{
			name:      "before-after shows role diagram",
			narrative: "before-after",
			sections: []diffview.Section{
				{Role: "cleanup", Title: "Remove old", Hunks: []diffview.HunkRef{{File: "file.go", HunkIndex: 0}}},
				{Role: "core", Title: "Add new", Hunks: []diffview.HunkRef{{File: "file.go", HunkIndex: 0}}},
			},
			expected: []string{"cleanup", "core", "→"},
		},
		{
			name:      "entry-implementation shows role diagram",
			narrative: "entry-implementation",
			sections: []diffview.Section{
				{Role: "entry", Title: "API", Hunks: []diffview.HunkRef{{File: "file.go", HunkIndex: 0}}},
				{Role: "implementation", Title: "Impl", Hunks: []diffview.HunkRef{{File: "file.go", HunkIndex: 0}}},
			},
			expected: []string{"entry", "implementation", "→"},
		},
		{
			name:      "rule-instances shows role diagram",
			narrative: "rule-instances",
			sections: []diffview.Section{
				{Role: "pattern", Title: "The Pattern", Hunks: []diffview.HunkRef{{File: "file.go", HunkIndex: 0}}},
				{Role: "instance", Title: "Application", Hunks: []diffview.HunkRef{{File: "file.go", HunkIndex: 0}}},
			},
			expected: []string{"pattern", "instance", "→"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			story := &diffview.StoryClassification{
				ChangeType: "feature",
				Narrative:  tt.narrative,
				Summary:    "Test summary",
				Sections:   tt.sections,
			}

			m := bubbletea.NewStoryModel(diff, story, bubbletea.WithIntroSlide())
			tm := teatest.NewTestModel(t, m,
				teatest.WithInitialTermSize(80, 24),
			)

			// Should show narrative diagram or explanation
			teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
				for _, exp := range tt.expected {
					if !bytes.Contains(out, []byte(exp)) {
						return false
					}
				}
				return true
			})

			tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
			tm.WaitFinished(t, teatest.WithFinalTimeout(0))
		})
	}
}

func TestStoryModel_IntroSlide_ShowsSectionRolePrefixes(t *testing.T) {
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
							{Type: diffview.LineContext, Content: "content1"},
						},
					},
					{
						OldStart: 10, OldCount: 1, NewStart: 10, NewCount: 1,
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "content2"},
						},
					},
					{
						OldStart: 20, OldCount: 1, NewStart: 20, NewCount: 1,
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "content3"},
						},
					},
				},
			},
		},
	}

	story := &diffview.StoryClassification{
		ChangeType: "bugfix",
		Narrative:  "cause-effect",
		Summary:    "Fix crash on empty input",
		Sections: []diffview.Section{
			{
				Role:  "problem",
				Title: "The Bug Location",
				Hunks: []diffview.HunkRef{
					{File: "file.go", HunkIndex: 0, Category: "core"},
				},
			},
			{
				Role:  "fix",
				Title: "The Fix",
				Hunks: []diffview.HunkRef{
					{File: "file.go", HunkIndex: 1, Category: "core"},
				},
			},
			{
				Role:  "test",
				Title: "Tests",
				Hunks: []diffview.HunkRef{
					{File: "file.go", HunkIndex: 2, Category: "core"},
				},
			},
		},
	}

	m := bubbletea.NewStoryModel(diff, story, bubbletea.WithIntroSlide())
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Should show section list with [role] prefixes
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasProblem := bytes.Contains(out, []byte("[problem]")) &&
			bytes.Contains(out, []byte("The Bug Location"))
		hasFix := bytes.Contains(out, []byte("[fix]")) &&
			bytes.Contains(out, []byte("The Fix"))
		hasTest := bytes.Contains(out, []byte("[test]")) &&
			bytes.Contains(out, []byte("Tests"))
		return hasProblem && hasFix && hasTest
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestStoryModel_IntroSlide_UnknownNarrativeFallback(t *testing.T) {
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

	story := &diffview.StoryClassification{
		ChangeType: "feature",
		Narrative:  "unknown-narrative-type",
		Summary:    "Some feature implementation",
		Sections: []diffview.Section{
			{
				Role:  "core",
				Title: "The Feature",
				Hunks: []diffview.HunkRef{
					{File: "file.go", HunkIndex: 0, Category: "core"},
				},
			},
		},
	}

	m := bubbletea.NewStoryModel(diff, story, bubbletea.WithIntroSlide())
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Should show summary and sections but NO "Story:" line for unknown narrative
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasSummary := bytes.Contains(out, []byte("[feature]")) &&
			bytes.Contains(out, []byte("Some feature implementation"))
		hasSections := bytes.Contains(out, []byte("Sections:"))
		noStoryLine := !bytes.Contains(out, []byte("Story:"))
		return hasSummary && hasSections && noStoryLine
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}
