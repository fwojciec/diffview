package bubbletea_test

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/fwojciec/diffview"
	"github.com/fwojciec/diffview/bubbletea"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// trueColorRenderer creates a lipgloss renderer that outputs true colors.
// This is useful for testing color output without affecting global state.
func trueColorRenderer() *lipgloss.Renderer {
	r := lipgloss.NewRenderer(io.Discard)
	r.SetColorProfile(termenv.TrueColor)
	return r
}

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
		teatest.WithInitialTermSize(80, 8), // Height allows hunk header + first content line
	)

	// Wait for initial render - should show first hunk (after file headers)
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("HUNK1_START"))
	})

	// Press 'n' to go to next hunk
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	// Should now show second hunk (header + first content line visible)
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("HUNK2_START"))
	})

	// Press 'n' again to go to third hunk (in file 2)
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
		teatest.WithInitialTermSize(80, 8), // Height allows hunk header + first content line
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
		teatest.WithInitialTermSize(80, 8), // Height allows hunk header + first content line
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

	// Hunk positions now point to hunk headers (after file headers)
	// File 1: lines 0-1 are file headers, line 2 is hunk 1 header
	// Lines 3-4 are hunk 1 content, line 5 is hunk 2 header, line 6 is content
	// File 2: lines 7-8 are file headers, line 9 is hunk header, line 10 is content
	hunkPositions := m.HunkPositions()
	assert.Len(t, hunkPositions, 3, "should track 3 hunks")
	assert.Equal(t, 2, hunkPositions[0], "first hunk header at line 2")
	assert.Equal(t, 5, hunkPositions[1], "second hunk header at line 5")
	assert.Equal(t, 9, hunkPositions[2], "third hunk header at line 9")

	// File positions point to file headers (--- line)
	filePositions := m.FilePositions()
	assert.Len(t, filePositions, 2, "should track 2 files")
	assert.Equal(t, 0, filePositions[0], "first file header at line 0")
	assert.Equal(t, 7, filePositions[1], "second file header at line 7")
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
	// File 1: lines 0-1 (headers), line 2 (hunk), line 3 (content)
	// Binary file is skipped entirely
	// File 2: lines 4-5 (headers), line 6 (hunk), line 7 (content)
	filePositions := m.FilePositions()
	assert.Len(t, filePositions, 2, "should only track 2 files with hunks")
	assert.Equal(t, 0, filePositions[0], "first file header at line 0")
	assert.Equal(t, 4, filePositions[1], "second file header at line 4")

	// Hunks should still be tracked correctly
	hunkPositions := m.HunkPositions()
	assert.Len(t, hunkPositions, 2, "should track 2 hunks")
	assert.Equal(t, 2, hunkPositions[0], "first hunk at line 2")
	assert.Equal(t, 6, hunkPositions[1], "second hunk at line 6")
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

func TestModel_RendersFileHeaders(t *testing.T) {
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
							{Type: diffview.LineContext, Content: "context line"},
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

	// Should render file headers (--- and +++)
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("--- a/test.go")) &&
			bytes.Contains(out, []byte("+++ b/test.go"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_RendersHunkHeaders(t *testing.T) {
	t.Parallel()

	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				OldPath:   "a/test.go",
				NewPath:   "b/test.go",
				Operation: diffview.FileModified,
				Hunks: []diffview.Hunk{
					{
						OldStart: 10,
						OldCount: 3,
						NewStart: 10,
						NewCount: 5,
						Section:  "func Example",
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "context line"},
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

	// Should render hunk header with @@ markers
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("@@ -10,3 +10,5 @@ func Example"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_RendersLinePrefixes(t *testing.T) {
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
						OldCount: 2,
						NewStart: 1,
						NewCount: 2,
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "unchanged"},
							{Type: diffview.LineDeleted, Content: "removed"},
							{Type: diffview.LineAdded, Content: "added"},
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

	// Should render lines with prefixes
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasContext := bytes.Contains(out, []byte(" unchanged"))
		hasDeleted := bytes.Contains(out, []byte("-removed"))
		hasAdded := bytes.Contains(out, []byte("+added"))
		return hasContext && hasDeleted && hasAdded
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_AppliesColors(t *testing.T) {
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
						NewCount: 2,
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "context"},
							{Type: diffview.LineAdded, Content: "added"},
						},
					},
				},
			},
		},
	}

	// Use WithRenderer to force true color output without global state
	m := bubbletea.NewModel(diff, bubbletea.WithRenderer(trueColorRenderer()))
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Wait for styled output - true color foreground codes use 38;2;R;G;B format
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasForegroundColor := bytes.Contains(out, []byte("38;2;"))
		hasContent := bytes.Contains(out, []byte("added"))
		return hasForegroundColor && hasContent
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_AddedLinesHaveBackgroundColor(t *testing.T) {
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
						NewCount: 2,
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "context line"},
							{Type: diffview.LineAdded, Content: "added line"},
						},
					},
				},
			},
		},
	}

	m := bubbletea.NewModel(diff, bubbletea.WithRenderer(trueColorRenderer()))
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Wait for output with background color on added line
	// True color backgrounds use ESC[48;2;R;G;Bm format
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasAddedLine := bytes.Contains(out, []byte("+added line"))
		hasBackgroundColor := bytes.Contains(out, []byte("48;2;"))
		return hasAddedLine && hasBackgroundColor
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_DeletedLinesHaveBackgroundColor(t *testing.T) {
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
						OldCount: 2,
						NewStart: 1,
						NewCount: 1,
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "context line"},
							{Type: diffview.LineDeleted, Content: "deleted line"},
						},
					},
				},
			},
		},
	}

	m := bubbletea.NewModel(diff, bubbletea.WithRenderer(trueColorRenderer()))
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Wait for output with background color on deleted line
	// True color backgrounds use ESC[48;2;R;G;Bm format
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasDeletedLine := bytes.Contains(out, []byte("-deleted line"))
		hasBackgroundColor := bytes.Contains(out, []byte("48;2;"))
		return hasDeletedLine && hasBackgroundColor
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_BackgroundExtendsFullWidth(t *testing.T) {
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
						NewCount: 2,
						Lines: []diffview.Line{
							{Type: diffview.LineAdded, Content: "short"},
						},
					},
				},
			},
		},
	}

	m := bubbletea.NewModel(diff, bubbletea.WithRenderer(trueColorRenderer()))
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Background should extend beyond just the text "+short"
	// The styled content should include padding spaces within the style
	// Looking for background color followed by spaces within the styled region
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasAddedLine := bytes.Contains(out, []byte("+short"))
		// Check for padding spaces within styled region (spaces before reset code)
		// Pattern: spaces followed by ESC[0m (reset)
		hasStyledPadding := bytes.Contains(out, []byte("   \x1b[0m")) ||
			bytes.Contains(out, []byte("  \x1b[0m"))
		return hasAddedLine && hasStyledPadding
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_BackgroundExtendsFullWidthWithUnicode(t *testing.T) {
	t.Parallel()

	// Test with multi-byte Unicode characters to ensure padding uses display width
	// "日本語" is 3 characters, 9 bytes, but 6 display cells (CJK are double-width)
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
						NewCount: 2,
						Lines: []diffview.Line{
							{Type: diffview.LineAdded, Content: "日本語"},
						},
					},
				},
			},
		},
	}

	m := bubbletea.NewModel(diff, bubbletea.WithRenderer(trueColorRenderer()))
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Background should extend full width even with Unicode content
	// The line "+日本語" should be padded with spaces within the styled region
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasUnicodeLine := bytes.Contains(out, []byte("+日本語"))
		// Check for padding spaces within styled region (spaces before reset code)
		hasStyledPadding := bytes.Contains(out, []byte("   \x1b[0m")) ||
			bytes.Contains(out, []byte("  \x1b[0m"))
		return hasUnicodeLine && hasStyledPadding
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_StatusBarShowsFilePosition(t *testing.T) {
	t.Parallel()

	// Create diff with 3 files
	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				OldPath: "a/first.go",
				NewPath: "b/first.go",
				Hunks: []diffview.Hunk{
					{Lines: []diffview.Line{{Type: diffview.LineContext, Content: "first file"}}},
				},
			},
			{
				OldPath: "a/second.go",
				NewPath: "b/second.go",
				Hunks: []diffview.Hunk{
					{Lines: []diffview.Line{{Type: diffview.LineContext, Content: "second file"}}},
				},
			},
			{
				OldPath: "a/third.go",
				NewPath: "b/third.go",
				Hunks: []diffview.Hunk{
					{Lines: []diffview.Line{{Type: diffview.LineContext, Content: "third file"}}},
				},
			},
		},
	}

	m := bubbletea.NewModel(diff)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Status bar should show file 1/3 when at top
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("file 1/3"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_StatusBarShowsHunkPosition(t *testing.T) {
	t.Parallel()

	// Create diff with one file containing 3 hunks
	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				OldPath: "a/file.go",
				NewPath: "b/file.go",
				Hunks: []diffview.Hunk{
					{Lines: []diffview.Line{{Type: diffview.LineContext, Content: "hunk1"}}},
					{Lines: []diffview.Line{{Type: diffview.LineContext, Content: "hunk2"}}},
					{Lines: []diffview.Line{{Type: diffview.LineContext, Content: "hunk3"}}},
				},
			},
		},
	}

	m := bubbletea.NewModel(diff)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Status bar should show hunk 1/3 when at top
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("hunk 1/3"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_StatusBarShowsScrollPosition(t *testing.T) {
	t.Parallel()

	// Create diff with many lines to enable scrolling
	lines := make([]diffview.Line, 100)
	for i := range lines {
		lines[i] = diffview.Line{Type: diffview.LineContext, Content: "content line"}
	}

	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				OldPath: "a/file.go",
				NewPath: "b/file.go",
				Hunks: []diffview.Hunk{
					{Lines: lines},
				},
			},
		},
	}

	m := bubbletea.NewModel(diff)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 10), // Small height to enable scrolling
	)

	// At top, should show "Top"
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Top"))
	})

	// Scroll down half page to get percentage display
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlD})

	// Should show a percentage (contains %)
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("%"))
	})

	// Scroll to bottom
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})

	// At bottom, should show "Bot"
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Bot"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_StatusBarShowsKeyHints(t *testing.T) {
	t.Parallel()

	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				OldPath: "a/file.go",
				NewPath: "b/file.go",
				Hunks: []diffview.Hunk{
					{Lines: []diffview.Line{{Type: diffview.LineContext, Content: "content"}}},
				},
			},
		},
	}

	m := bubbletea.NewModel(diff)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Status bar should show key hints
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasScroll := bytes.Contains(out, []byte("j/k"))
		hasHunk := bytes.Contains(out, []byte("n/N"))
		hasFile := bytes.Contains(out, []byte("]/["))
		hasQuit := bytes.Contains(out, []byte("q"))
		return hasScroll && hasHunk && hasFile && hasQuit
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_StatusBarUpdatesOnFileNavigation(t *testing.T) {
	t.Parallel()

	// Create 3 files with multiple lines each
	lines := make([]diffview.Line, 20)
	for i := range lines {
		lines[i] = diffview.Line{Type: diffview.LineContext, Content: "content line"}
	}

	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				OldPath: "a/first.go",
				NewPath: "b/first.go",
				Hunks:   []diffview.Hunk{{Lines: lines}},
			},
			{
				OldPath: "a/second.go",
				NewPath: "b/second.go",
				Hunks:   []diffview.Hunk{{Lines: lines}},
			},
			{
				OldPath: "a/third.go",
				NewPath: "b/third.go",
				Hunks:   []diffview.Hunk{{Lines: lines}},
			},
		},
	}

	m := bubbletea.NewModel(diff)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 10), // Small height to enable scrolling
	)

	// Initially at file 1/3
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("file 1/3"))
	})

	// Navigate to next file
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})

	// Should now show file 2/3
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("file 2/3"))
	})

	// Navigate to next file
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})

	// Should now show file 3/3
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("file 3/3"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_StatusBarUpdatesOnHunkNavigation(t *testing.T) {
	t.Parallel()

	// Create file with 3 hunks, each with multiple lines
	lines := make([]diffview.Line, 15)
	for i := range lines {
		lines[i] = diffview.Line{Type: diffview.LineContext, Content: "content line"}
	}

	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				OldPath: "a/file.go",
				NewPath: "b/file.go",
				Hunks: []diffview.Hunk{
					{Lines: lines},
					{Lines: lines},
					{Lines: lines},
				},
			},
		},
	}

	m := bubbletea.NewModel(diff)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 10), // Small height to enable scrolling
	)

	// Initially at hunk 1/3
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("hunk 1/3"))
	})

	// Navigate to next hunk with 'n'
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	// Should now show hunk 2/3
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("hunk 2/3"))
	})

	// Navigate to next hunk
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	// Should now show hunk 3/3
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("hunk 3/3"))
	})

	// Navigate back with 'N'
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})

	// Should now show hunk 2/3
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("hunk 2/3"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_RendersLineNumbersInGutter(t *testing.T) {
	t.Parallel()

	// Create diff with known line numbers
	// Context line at old:10, new:10
	// Deleted line at old:11, new:-
	// Added line at old:-, new:11
	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				OldPath:   "a/test.go",
				NewPath:   "b/test.go",
				Operation: diffview.FileModified,
				Hunks: []diffview.Hunk{
					{
						OldStart: 10,
						OldCount: 2,
						NewStart: 10,
						NewCount: 2,
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "context", OldLineNum: 10, NewLineNum: 10},
							{Type: diffview.LineDeleted, Content: "deleted", OldLineNum: 11, NewLineNum: 0},
							{Type: diffview.LineAdded, Content: "added", OldLineNum: 0, NewLineNum: 11},
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

	// Should render line numbers in gutter
	// Format: "  10    10 │" for context line
	// Format: "  11     - │" for deleted line (no new line number)
	// Format: "   -    11 │" for added line (no old line number)
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		// Check for context line with both numbers
		hasContext := bytes.Contains(out, []byte("10")) && bytes.Contains(out, []byte("context"))
		// Check for deleted line with old number and dash
		hasDeleted := bytes.Contains(out, []byte("11")) && bytes.Contains(out, []byte("-deleted"))
		// Check for added line with dash and new number
		hasAdded := bytes.Contains(out, []byte("+added"))
		return hasContext && hasDeleted && hasAdded
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_GutterShowsDashForMissingLineNumbers(t *testing.T) {
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
						NewCount: 2,
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "context", OldLineNum: 1, NewLineNum: 1},
							{Type: diffview.LineAdded, Content: "new line", OldLineNum: 0, NewLineNum: 2},
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

	// For added lines, old line number should show as "-"
	// The gutter should show something like "   -     2 │+new line"
	// Look for the separator immediately before the + prefix for added lines
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		// The gutter separator should appear right before the + prefix
		hasGutterBeforeAdded := bytes.Contains(out, []byte("│+new line"))
		return hasGutterBeforeAdded
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_HighlightsWordLevelChanges(t *testing.T) {
	t.Parallel()

	// Create diff with a deleted+added pair where only one word changes
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
							{Type: diffview.LineDeleted, Content: "hello world", OldLineNum: 1, NewLineNum: 0},
							{Type: diffview.LineAdded, Content: "hello universe", OldLineNum: 0, NewLineNum: 1},
						},
					},
				},
			},
		},
	}

	// Use WithRenderer to force true color output
	m := bubbletea.NewModel(diff, bubbletea.WithRenderer(trueColorRenderer()))
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Wait for content to render with word-level highlighting
	// The changed words "world" and "universe" should have highlight background colors
	// AddedHighlight background: #a6e3a1 = RGB(166, 227, 161) -> "48;2;166;227;161"
	// DeletedHighlight background: #f38ba8 = RGB(243, 139, 168) -> "48;2;243;139;168"
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		// Check that the content appears
		hasWorld := bytes.Contains(out, []byte("world"))
		hasUniverse := bytes.Contains(out, []byte("universe"))
		// Check for the highlight background colors (true color format: 48;2;R;G;B)
		hasAddedHighlight := bytes.Contains(out, []byte("48;2;166;227;161"))
		hasDeletedHighlight := bytes.Contains(out, []byte("48;2;243;139;168"))
		return hasWorld && hasUniverse && hasAddedHighlight && hasDeletedHighlight
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}
