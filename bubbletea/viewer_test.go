package bubbletea_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/fwojciec/diffview"
	"github.com/fwojciec/diffview/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time check that Viewer implements diffview.Viewer.
var _ diffview.Viewer = (*bubbletea.Viewer)(nil)

func TestModel_Init(t *testing.T) {
	t.Parallel()

	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				OldPath:   "a/file.go",
				NewPath:   "b/file.go",
				Operation: diffview.FileModified,
				Hunks: []diffview.Hunk{
					{
						OldStart: 1,
						OldCount: 3,
						NewStart: 1,
						NewCount: 4,
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "context line"},
							{Type: diffview.LineDeleted, Content: "deleted line"},
							{Type: diffview.LineAdded, Content: "added line 1"},
							{Type: diffview.LineAdded, Content: "added line 2"},
						},
					},
				},
			},
		},
	}

	m := bubbletea.NewModel(diff)
	cmd := m.Init()

	assert.Nil(t, cmd, "Init should return nil command")
}

func TestModel_ViewBeforeReady(t *testing.T) {
	t.Parallel()

	diff := &diffview.Diff{}
	m := bubbletea.NewModel(diff)

	view := m.View()

	assert.Contains(t, view, "Loading", "View should show loading state before WindowSizeMsg")
}

func TestModel_ViewAfterReady(t *testing.T) {
	t.Parallel()

	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				OldPath:   "a/test.go",
				NewPath:   "b/test.go",
				Operation: diffview.FileModified,
				Hunks: []diffview.Hunk{
					{
						OldStart: 1,
						OldCount: 1,
						NewStart: 1,
						NewCount: 1,
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "test content"},
						},
					},
				},
			},
		},
	}

	m := bubbletea.NewModel(diff)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Wait for content to appear - this verifies the view is rendered correctly
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("test content"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_QuitOnQ(t *testing.T) {
	t.Parallel()

	diff := &diffview.Diff{}
	m := bubbletea.NewModel(diff)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_QuitOnCtrlC(t *testing.T) {
	t.Parallel()

	diff := &diffview.Diff{}
	m := bubbletea.NewModel(diff)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})

	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_WindowResize(t *testing.T) {
	t.Parallel()

	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				OldPath:   "a/resize.go",
				NewPath:   "b/resize.go",
				Operation: diffview.FileModified,
				Hunks: []diffview.Hunk{
					{
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "resize test"},
						},
					},
				},
			},
		},
	}

	m := bubbletea.NewModel(diff)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("resize test"))
	})

	// Resize window
	tm.Send(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Content should still be visible
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("resize test"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_GotoBottomOnG(t *testing.T) {
	t.Parallel()

	// Create content with many lines so we can scroll
	lines := make([]diffview.Line, 100)
	for i := range lines {
		lines[i] = diffview.Line{Type: diffview.LineContext, Content: "line content"}
	}
	// Add unique markers at top and bottom
	lines[0] = diffview.Line{Type: diffview.LineContext, Content: "FIRST_LINE_MARKER"}
	lines[99] = diffview.Line{Type: diffview.LineContext, Content: "LAST_LINE_MARKER"}

	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				Hunks: []diffview.Hunk{{Lines: lines}},
			},
		},
	}

	m := bubbletea.NewModel(diff)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 10), // Small height to enable scrolling
	)

	// Wait for initial render with first line visible
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("FIRST_LINE_MARKER"))
	})

	// Scroll down with G (go to bottom)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})

	// Wait for last line to be visible
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("LAST_LINE_MARKER"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_GotoTopOnGG(t *testing.T) {
	t.Parallel()

	// Create content with many lines so we can scroll
	lines := make([]diffview.Line, 100)
	for i := range lines {
		lines[i] = diffview.Line{Type: diffview.LineContext, Content: "line content"}
	}
	// Add unique markers at top and bottom
	lines[0] = diffview.Line{Type: diffview.LineContext, Content: "FIRST_LINE_MARKER"}
	lines[99] = diffview.Line{Type: diffview.LineContext, Content: "LAST_LINE_MARKER"}

	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				Hunks: []diffview.Hunk{{Lines: lines}},
			},
		},
	}

	m := bubbletea.NewModel(diff)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 10), // Small height to enable scrolling
	)

	// Wait for initial render with first line visible
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("FIRST_LINE_MARKER"))
	})

	// First scroll to bottom with G (setup for testing gg)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})

	// Wait for last line to be visible
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("LAST_LINE_MARKER"))
	})

	// Now press gg to go back to top
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})

	// Wait for first line to be visible again
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("FIRST_LINE_MARKER"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_PendingGClearedOnOtherKey(t *testing.T) {
	t.Parallel()

	// This test verifies that pressing 'g' followed by a non-'g' key
	// clears the pending state and doesn't trigger GotoTop.
	// We test this by pressing 'g' then 'q' - if pending wasn't cleared
	// properly, the program might not quit.

	diff := &diffview.Diff{}
	m := bubbletea.NewModel(diff)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Press 'g' then 'q' - should quit (not wait for another 'g')
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_NextHunkNavigation(t *testing.T) {
	t.Parallel()

	// Create content with 3 hunks spread out
	// File 1: Hunk 1 (lines 0-9), Hunk 2 (lines 10-19)
	// File 2: Hunk 3 (lines 20-29)
	lines1 := make([]diffview.Line, 10)
	for i := range lines1 {
		lines1[i] = diffview.Line{Type: diffview.LineContext, Content: "file1 hunk1 line"}
	}
	lines1[0].Content = "HUNK1_START"

	lines2 := make([]diffview.Line, 10)
	for i := range lines2 {
		lines2[i] = diffview.Line{Type: diffview.LineContext, Content: "file1 hunk2 line"}
	}
	lines2[0].Content = "HUNK2_START"

	lines3 := make([]diffview.Line, 10)
	for i := range lines3 {
		lines3[i] = diffview.Line{Type: diffview.LineContext, Content: "file2 hunk1 line"}
	}
	lines3[0].Content = "HUNK3_START"

	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				Hunks: []diffview.Hunk{
					{Lines: lines1},
					{Lines: lines2},
				},
			},
			{
				Hunks: []diffview.Hunk{
					{Lines: lines3},
				},
			},
		},
	}

	m := bubbletea.NewModel(diff)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 5), // Small height to enable scrolling
	)

	// Wait for initial render - should show first hunk
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("HUNK1_START"))
	})

	// Press 'n' to go to next hunk
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	// Should now show second hunk
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("HUNK2_START"))
	})

	// Press 'n' again to go to third hunk
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	// Should now show third hunk
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("HUNK3_START"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_PrevHunkNavigation(t *testing.T) {
	t.Parallel()

	// Create content with 3 hunks spread out
	lines1 := make([]diffview.Line, 10)
	for i := range lines1 {
		lines1[i] = diffview.Line{Type: diffview.LineContext, Content: "file1 hunk1 line"}
	}
	lines1[0].Content = "HUNK1_START"

	lines2 := make([]diffview.Line, 10)
	for i := range lines2 {
		lines2[i] = diffview.Line{Type: diffview.LineContext, Content: "file1 hunk2 line"}
	}
	lines2[0].Content = "HUNK2_START"

	lines3 := make([]diffview.Line, 10)
	for i := range lines3 {
		lines3[i] = diffview.Line{Type: diffview.LineContext, Content: "file2 hunk1 line"}
	}
	lines3[0].Content = "HUNK3_START"

	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				Hunks: []diffview.Hunk{
					{Lines: lines1},
					{Lines: lines2},
				},
			},
			{
				Hunks: []diffview.Hunk{
					{Lines: lines3},
				},
			},
		},
	}

	m := bubbletea.NewModel(diff)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 5),
	)

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("HUNK1_START"))
	})

	// Navigate to hunk 3 using next hunk
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("HUNK2_START"))
	})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("HUNK3_START"))
	})

	// Press 'N' to go to previous hunk
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})

	// Should now show second hunk
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("HUNK2_START"))
	})

	// Press 'N' again to go to first hunk
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})

	// Should now show first hunk
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("HUNK1_START"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_NextFileNavigation(t *testing.T) {
	t.Parallel()

	// Create content with 3 files
	lines1 := make([]diffview.Line, 10)
	for i := range lines1 {
		lines1[i] = diffview.Line{Type: diffview.LineContext, Content: "file1 content"}
	}
	lines1[0].Content = "FILE1_START"

	lines2 := make([]diffview.Line, 10)
	for i := range lines2 {
		lines2[i] = diffview.Line{Type: diffview.LineContext, Content: "file2 content"}
	}
	lines2[0].Content = "FILE2_START"

	lines3 := make([]diffview.Line, 10)
	for i := range lines3 {
		lines3[i] = diffview.Line{Type: diffview.LineContext, Content: "file3 content"}
	}
	lines3[0].Content = "FILE3_START"

	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{Hunks: []diffview.Hunk{{Lines: lines1}}},
			{Hunks: []diffview.Hunk{{Lines: lines2}}},
			{Hunks: []diffview.Hunk{{Lines: lines3}}},
		},
	}

	m := bubbletea.NewModel(diff)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 5),
	)

	// Wait for initial render - should show first file
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("FILE1_START"))
	})

	// Press ']' to go to next file
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})

	// Should now show second file
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("FILE2_START"))
	})

	// Press ']' again to go to third file
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})

	// Should now show third file
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("FILE3_START"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_PrevFileNavigation(t *testing.T) {
	t.Parallel()

	// Create content with 3 files
	lines1 := make([]diffview.Line, 10)
	for i := range lines1 {
		lines1[i] = diffview.Line{Type: diffview.LineContext, Content: "file1 content"}
	}
	lines1[0].Content = "FILE1_START"

	lines2 := make([]diffview.Line, 10)
	for i := range lines2 {
		lines2[i] = diffview.Line{Type: diffview.LineContext, Content: "file2 content"}
	}
	lines2[0].Content = "FILE2_START"

	lines3 := make([]diffview.Line, 10)
	for i := range lines3 {
		lines3[i] = diffview.Line{Type: diffview.LineContext, Content: "file3 content"}
	}
	lines3[0].Content = "FILE3_START"

	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{Hunks: []diffview.Hunk{{Lines: lines1}}},
			{Hunks: []diffview.Hunk{{Lines: lines2}}},
			{Hunks: []diffview.Hunk{{Lines: lines3}}},
		},
	}

	m := bubbletea.NewModel(diff)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 5),
	)

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("FILE1_START"))
	})

	// Navigate to file 3 using next file
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("FILE2_START"))
	})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("FILE3_START"))
	})

	// Press '[' to go to previous file
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})

	// Should now show second file
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("FILE2_START"))
	})

	// Press '[' again to go to first file
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})

	// Should now show first file
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("FILE1_START"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_NavigationWithEmptyDiff(t *testing.T) {
	t.Parallel()

	diff := &diffview.Diff{Files: []diffview.FileDiff{}}
	m := bubbletea.NewModel(diff)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Navigation keys should not panic or cause issues with empty diff
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})

	// Should still be able to quit normally
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_NavigationAtBoundaries(t *testing.T) {
	t.Parallel()

	// Create two hunks so we can test navigating to a boundary and staying there
	lines1 := make([]diffview.Line, 10)
	for i := range lines1 {
		lines1[i] = diffview.Line{Type: diffview.LineContext, Content: "hunk1 content"}
	}
	lines1[0].Content = "FIRST_HUNK"

	lines2 := make([]diffview.Line, 10)
	for i := range lines2 {
		lines2[i] = diffview.Line{Type: diffview.LineContext, Content: "hunk2 content"}
	}
	lines2[0].Content = "LAST_HUNK"

	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{Hunks: []diffview.Hunk{{Lines: lines1}, {Lines: lines2}}},
		},
	}

	m := bubbletea.NewModel(diff)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 5),
	)

	// Wait for initial render at first hunk
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("FIRST_HUNK"))
	})

	// Press 'N' at first hunk - should stay at first hunk (no previous)
	// Then press 'n' to go to second hunk
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	// Should now be at last hunk
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("LAST_HUNK"))
	})

	// Press 'n' at last hunk - should stay (no next)
	// Then press 'N' to go back to first hunk
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})

	// Should be back at first hunk
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("FIRST_HUNK"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_TracksHunkPositions(t *testing.T) {
	t.Parallel()

	// Create diff with 2 files, each with 2 hunks
	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				OldPath: "a/file1.go",
				NewPath: "b/file1.go",
				Hunks: []diffview.Hunk{
					{
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "file1 hunk1 line1"},
							{Type: diffview.LineContext, Content: "file1 hunk1 line2"},
						},
					},
					{
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "file1 hunk2 line1"},
						},
					},
				},
			},
			{
				OldPath: "a/file2.go",
				NewPath: "b/file2.go",
				Hunks: []diffview.Hunk{
					{
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "file2 hunk1 line1"},
						},
					},
				},
			},
		},
	}

	m := bubbletea.NewModel(diff)

	// Hunk positions should be tracked: hunk 0 at line 0, hunk 1 at line 2, hunk 2 at line 3
	hunkPositions := m.HunkPositions()
	assert.Len(t, hunkPositions, 3, "should track 3 hunks")
	assert.Equal(t, 0, hunkPositions[0], "first hunk at line 0")
	assert.Equal(t, 2, hunkPositions[1], "second hunk at line 2")
	assert.Equal(t, 3, hunkPositions[2], "third hunk at line 3")

	// File positions should be tracked: file 0 at line 0, file 1 at line 3
	filePositions := m.FilePositions()
	assert.Len(t, filePositions, 2, "should track 2 files")
	assert.Equal(t, 0, filePositions[0], "first file at line 0")
	assert.Equal(t, 3, filePositions[1], "second file at line 3")
}

func TestModel_SkipsFilesWithNoHunks(t *testing.T) {
	t.Parallel()

	// Create diff with a mix of files with and without hunks (e.g., binary files)
	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				OldPath: "a/file1.go",
				NewPath: "b/file1.go",
				Hunks: []diffview.Hunk{
					{
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "file1 content"},
						},
					},
				},
			},
			{
				// Binary file with no hunks
				OldPath:  "a/image.png",
				NewPath:  "b/image.png",
				IsBinary: true,
				Hunks:    nil,
			},
			{
				OldPath: "a/file2.go",
				NewPath: "b/file2.go",
				Hunks: []diffview.Hunk{
					{
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "file2 content"},
						},
					},
				},
			},
		},
	}

	m := bubbletea.NewModel(diff)

	// Should only track files with hunks (skip binary file)
	filePositions := m.FilePositions()
	assert.Len(t, filePositions, 2, "should only track 2 files with hunks")
	assert.Equal(t, 0, filePositions[0], "first file at line 0")
	assert.Equal(t, 1, filePositions[1], "second file at line 1")

	// Hunks should still be tracked correctly
	hunkPositions := m.HunkPositions()
	assert.Len(t, hunkPositions, 2, "should track 2 hunks")
}

func TestViewer_ContextCancellation(t *testing.T) {
	t.Parallel()

	// Create a context with cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	diff := &diffview.Diff{}

	// Create viewer with custom IO to avoid TTY requirement
	var in bytes.Buffer
	var out bytes.Buffer
	viewer := bubbletea.NewViewer(
		bubbletea.WithProgramOptions(
			tea.WithInput(&in),
			tea.WithOutput(&out),
		),
	)

	// Run viewer in goroutine
	done := make(chan error, 1)
	go func() {
		done <- viewer.View(ctx, diff)
	}()

	// Give viewer time to start
	time.Sleep(50 * time.Millisecond)

	// Cancel context - this should terminate the viewer
	cancel()

	// Viewer should exit within reasonable time
	select {
	case err := <-done:
		// Verify context cancellation causes exit with context.Canceled error
		require.ErrorIs(t, err, context.Canceled, "viewer should return context.Canceled on cancellation")
	case <-time.After(1 * time.Second):
		t.Fatal("viewer did not exit after context cancellation")
	}
}

func TestViewer_ContextAlreadyCancelled(t *testing.T) {
	t.Parallel()

	// Create an already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	diff := &diffview.Diff{}

	var in bytes.Buffer
	var out bytes.Buffer
	viewer := bubbletea.NewViewer(
		bubbletea.WithProgramOptions(
			tea.WithInput(&in),
			tea.WithOutput(&out),
		),
	)

	// Viewer should exit immediately with already-cancelled context
	err := viewer.View(ctx, diff)
	require.ErrorIs(t, err, context.Canceled, "viewer should return context.Canceled for pre-cancelled context")
}
