package bubbletea_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/fwojciec/diffview"
	"github.com/fwojciec/diffview/bubbletea"
	dv "github.com/fwojciec/diffview/lipgloss"
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

	// Positions should be available immediately - no WindowSizeMsg needed!
	// Hunk positions point to hunk headers (after file headers)
	// File 1: line 0 is enhanced file header, line 1 is hunk 1 header
	// Lines 2-3 are hunk 1 content, line 4 is hunk 2 header, line 5 is content
	// File 2: line 6 is enhanced file header, line 7 is hunk header, line 8 is content
	hunkPositions := m.HunkPositions()
	assert.Len(t, hunkPositions, 3, "should track 3 hunks")
	assert.Equal(t, 1, hunkPositions[0], "first hunk header at line 1")
	assert.Equal(t, 4, hunkPositions[1], "second hunk header at line 4")
	assert.Equal(t, 7, hunkPositions[2], "third hunk header at line 7")

	// File positions point to enhanced file headers
	filePositions := m.FilePositions()
	assert.Len(t, filePositions, 2, "should track 2 files")
	assert.Equal(t, 0, filePositions[0], "first file header at line 0")
	assert.Equal(t, 6, filePositions[1], "second file header at line 6")
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

	// Positions should be available immediately - no WindowSizeMsg needed!
	// Should only track files with hunks (skip binary file)
	// File 1: line 0 (header), line 1 (hunk), line 2 (content)
	// Binary file is skipped entirely
	// File 2: line 3 (header), line 4 (hunk), line 5 (content)
	filePositions := m.FilePositions()
	assert.Len(t, filePositions, 2, "should only track 2 files with hunks")
	assert.Equal(t, 0, filePositions[0], "first file header at line 0")
	assert.Equal(t, 3, filePositions[1], "second file header at line 3")

	// Hunks should still be tracked correctly
	hunkPositions := m.HunkPositions()
	assert.Len(t, hunkPositions, 2, "should track 2 hunks")
	assert.Equal(t, 1, hunkPositions[0], "first hunk at line 1")
	assert.Equal(t, 4, hunkPositions[1], "second hunk at line 4")
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
		dv.TestTheme(),
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
		dv.TestTheme(),
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

	// Should render enhanced file header with box-drawing chars
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("── ")) &&
			bytes.Contains(out, []byte("test.go"))
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

	// Diff with all line types for comprehensive color testing
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
							{Type: diffview.LineContext, Content: "context"},
							{Type: diffview.LineDeleted, Content: "deleted"},
							{Type: diffview.LineAdded, Content: "added"},
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

	// Wait for output with both foreground and background colors
	// True color uses 38;2;R;G;B for foreground, 48;2;R;G;B for background
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasForegroundColor := bytes.Contains(out, []byte("38;2;"))
		hasBackgroundColor := bytes.Contains(out, []byte("48;2;"))
		hasAddedLine := bytes.Contains(out, []byte("+added"))
		hasDeletedLine := bytes.Contains(out, []byte("-deleted"))
		return hasForegroundColor && hasBackgroundColor && hasAddedLine && hasDeletedLine
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
		// Check for deleted line with old number and prefix
		hasDeleted := bytes.Contains(out, []byte("11")) && bytes.Contains(out, []byte("-deleted"))
		// Check for added line with new number and prefix
		hasAdded := bytes.Contains(out, []byte("+added"))
		return hasContext && hasDeleted && hasAdded
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_GutterUsesEmptySpaceForMissingLineNumbers(t *testing.T) {
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

	// For added lines, old line number should be empty space (not "-")
	// Gutter has no divider - color transition provides separation
	// The gutter directly precedes the line content: "    2 +new line"
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		// The gutter should NOT have divider character before the + prefix
		// (status bar uses │ as separator, so we check specifically for gutter)
		hasOldGutterFormat := bytes.Contains(out, []byte("│+new line"))
		hasContent := bytes.Contains(out, []byte("+new line"))
		// Also verify "-" placeholder is replaced with empty space
		hasDashPlaceholder := bytes.Contains(out, []byte("-    2"))
		return !hasOldGutterFormat && hasContent && !hasDashPlaceholder
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_GutterHasColoredBackgroundForAddedLines(t *testing.T) {
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
							{Type: diffview.LineAdded, Content: "added", OldLineNum: 0, NewLineNum: 2},
						},
					},
				},
			},
		},
	}

	// TestTheme has AddedGutter with background from blending #00ff00 with #000000 at 35%
	// Result: RGB(0, 89, 0) -> "48;2;0;89;0"
	theme := dv.TestTheme()
	m := bubbletea.NewModel(diff,
		bubbletea.WithTheme(theme),
		bubbletea.WithRenderer(trueColorRenderer()),
	)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// The gutter for added lines should have the AddedGutter background color
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasContent := bytes.Contains(out, []byte("+added"))
		// Check for the gutter background color (stronger green)
		hasGutterBackground := bytes.Contains(out, []byte("48;2;0;89;0"))
		return hasContent && hasGutterBackground
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_GutterHasColoredBackgroundForDeletedLines(t *testing.T) {
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
							{Type: diffview.LineContext, Content: "context", OldLineNum: 1, NewLineNum: 1},
							{Type: diffview.LineDeleted, Content: "deleted", OldLineNum: 2, NewLineNum: 0},
						},
					},
				},
			},
		},
	}

	// TestTheme has DeletedGutter with background from blending #ff0000 with #000000 at 35%
	// Result: RGB(89, 0, 0) -> "48;2;89;0;0"
	theme := dv.TestTheme()
	m := bubbletea.NewModel(diff,
		bubbletea.WithTheme(theme),
		bubbletea.WithRenderer(trueColorRenderer()),
	)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// The gutter for deleted lines should have the DeletedGutter background color
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasContent := bytes.Contains(out, []byte("-deleted"))
		// Check for the gutter background color (stronger red)
		hasGutterBackground := bytes.Contains(out, []byte("48;2;89;0;0"))
		return hasContent && hasGutterBackground
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_RendersFileHeaderWithStats(t *testing.T) {
	t.Parallel()

	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				OldPath:   "a/handler.go",
				NewPath:   "b/handler.go",
				Operation: diffview.FileModified,
				Hunks: []diffview.Hunk{
					{
						OldStart: 1,
						OldCount: 5,
						NewStart: 1,
						NewCount: 7,
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "context"},
							{Type: diffview.LineDeleted, Content: "old1"},
							{Type: diffview.LineDeleted, Content: "old2"},
							{Type: diffview.LineAdded, Content: "new1"},
							{Type: diffview.LineAdded, Content: "new2"},
							{Type: diffview.LineAdded, Content: "new3"},
							{Type: diffview.LineAdded, Content: "new4"},
							{Type: diffview.LineContext, Content: "context"},
						},
					},
				},
			},
		},
	}

	m := bubbletea.NewModel(diff, bubbletea.WithTheme(dv.TestTheme()))
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// File header should be enhanced with box-drawing and stats: ── file ─── +N -M ──
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		// Should have box-drawing prefix, filename, and stats
		return bytes.Contains(out, []byte("── ")) &&
			bytes.Contains(out, []byte("handler.go")) &&
			bytes.Contains(out, []byte("+4 -2"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_WithTheme(t *testing.T) {
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

	// TestTheme uses neutral foreground (#ffffff) with green-tinted background for added lines
	theme := dv.TestTheme()
	m := bubbletea.NewModel(diff,
		bubbletea.WithTheme(theme),
		bubbletea.WithRenderer(trueColorRenderer()),
	)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// TestTheme uses neutral foreground (#ffffff) with green-tinted background
	// Should see background color code with green tint (48;2;...)
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasContent := bytes.Contains(out, []byte("added"))
		// Check for any background color on the added line (48;2; prefix)
		hasBackground := bytes.Contains(out, []byte("48;2;"))
		return hasContent && hasBackground
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_StatusBarUsesThemeUIColors(t *testing.T) {
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

	// TestTheme has UIBackground=#333333 = RGB(51, 51, 51)
	// The status bar text "file 1/1" should have this background color
	theme := dv.TestTheme()
	m := bubbletea.NewModel(diff,
		bubbletea.WithTheme(theme),
		bubbletea.WithRenderer(trueColorRenderer()),
	)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Wait for the model to render and collect output
	var finalOutput []byte
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		if bytes.Contains(out, []byte("file 1/1")) {
			finalOutput = out
			return true
		}
		return false
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))

	// The status bar should use themed colors, which means there should be
	// color codes immediately before "file 1/1". Previously it used
	// lipgloss.NewStyle(), which ignored the renderer so the status bar
	// rendered without colors in tests; this test verifies it now has colors.
	//
	// Look for the pattern: background color code followed by "file 1/1"
	// TestTheme UIBackground is #333333 = RGB(51, 51, 51) -> "48;2;51;51;51"
	statusBarLine := extractLastLine(string(finalOutput))
	assert.Contains(t, statusBarLine, "48;2;51;51;51", "status bar should use TestTheme UIBackground color")
}

// extractLastLine returns the last non-empty line from the output.
func extractLastLine(s string) string {
	lines := bytes.Split([]byte(s), []byte("\n"))
	for i := len(lines) - 1; i >= 0; i-- {
		line := bytes.TrimSpace(lines[i])
		if len(line) > 0 {
			return string(lines[i])
		}
	}
	return ""
}

func TestModel_AppliesSyntaxHighlighting(t *testing.T) {
	t.Parallel()

	// Create a diff with Go code that will get syntax highlighted
	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				OldPath:   "a/main.go",
				NewPath:   "b/main.go",
				Operation: diffview.FileModified,
				Hunks: []diffview.Hunk{
					{
						OldStart: 1,
						OldCount: 1,
						NewStart: 1,
						NewCount: 2,
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "package main"},
							{Type: diffview.LineAdded, Content: "func main() {}", OldLineNum: 0, NewLineNum: 2},
						},
					},
				},
			},
		},
	}

	// Use TestTheme which has predictable colors:
	// Keyword: #ff00ff (magenta) = RGB(255, 0, 255) -> "38;2;255;0;255"
	theme := dv.TestTheme()

	// Create a mock tokenizer that returns tokens with keyword style
	tokenizer := &mockTokenizer{
		TokenizeFn: func(language, source string) []diffview.Token {
			if language != "Go" {
				return nil
			}
			// For "package main" return two tokens
			if source == "package main" {
				return []diffview.Token{
					{Text: "package", Style: diffview.Style{Foreground: "#ff00ff", Bold: true}},
					{Text: " ", Style: diffview.Style{}},
					{Text: "main", Style: diffview.Style{}},
				}
			}
			// For "func main() {}" return tokens
			if source == "func main() {}" {
				return []diffview.Token{
					{Text: "func", Style: diffview.Style{Foreground: "#ff00ff", Bold: true}},
					{Text: " ", Style: diffview.Style{}},
					{Text: "main", Style: diffview.Style{Foreground: "#0000ff"}},
					{Text: "()", Style: diffview.Style{}},
					{Text: " {}", Style: diffview.Style{}},
				}
			}
			return nil
		},
	}

	// Create a mock detector that returns "Go" for .go files
	detector := &mockLanguageDetector{
		DetectFromPathFn: func(path string) string {
			if len(path) >= 3 && path[len(path)-3:] == ".go" {
				return "Go"
			}
			return ""
		},
	}

	m := bubbletea.NewModel(diff,
		bubbletea.WithTheme(theme),
		bubbletea.WithRenderer(trueColorRenderer()),
		bubbletea.WithLanguageDetector(detector),
		bubbletea.WithTokenizer(tokenizer),
	)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Wait for output with syntax highlighting
	// The keyword "package" or "func" should have magenta foreground
	// RGB(255, 0, 255) -> "38;2;255;0;255"
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasContent := bytes.Contains(out, []byte("package"))
		hasMagentaKeyword := bytes.Contains(out, []byte("38;2;255;0;255"))
		return hasContent && hasMagentaKeyword
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

// mockTokenizer implements diffview.Tokenizer for testing.
type mockTokenizer struct {
	TokenizeFn func(language, source string) []diffview.Token
}

func (m *mockTokenizer) Tokenize(language, source string) []diffview.Token {
	return m.TokenizeFn(language, source)
}

// mockLanguageDetector implements diffview.LanguageDetector for testing.
type mockLanguageDetector struct {
	DetectFromPathFn func(path string) string
}

func (m *mockLanguageDetector) DetectFromPath(path string) string {
	return m.DetectFromPathFn(path)
}

func TestModel_PaddingBetweenGutterAndCodePrefix(t *testing.T) {
	t.Parallel()

	// Create a diff with added, deleted, and context lines
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
							{Type: diffview.LineContext, Content: "context", OldLineNum: 1, NewLineNum: 1},
							{Type: diffview.LineDeleted, Content: "deleted", OldLineNum: 2, NewLineNum: 0},
							{Type: diffview.LineAdded, Content: "added", OldLineNum: 0, NewLineNum: 2},
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

	// Wait for output with padding space between gutter and line prefix
	// The padding space appears between the gutter and the prefix character (+/-/space)
	// Due to ANSI color codes, the padding space may be separated from the prefix by escape sequences
	// We verify by checking that the rendered text shows " +added", " -deleted", "  context"
	// (padding space + prefix + content for each line type)
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		// After the gutter styling ends (reset code), we should see the padding space
		// followed by the prefix character. Check for space-prefix-content patterns.
		hasAddedWithPadding := bytes.Contains(out, []byte(" +added"))
		hasDeletedWithPadding := bytes.Contains(out, []byte(" -deleted"))
		// For context lines, the prefix is a space, so we get "  context" (padding + prefix + content)
		hasContextWithPadding := bytes.Contains(out, []byte("  context"))
		return hasAddedWithPadding && hasDeletedWithPadding && hasContextWithPadding
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_PaddingUsesCodeLineBackgroundColor(t *testing.T) {
	t.Parallel()

	// Create a diff with an added line to test padding background color
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
							{Type: diffview.LineAdded, Content: "added", OldLineNum: 0, NewLineNum: 2},
						},
					},
				},
			},
		},
	}

	// TestTheme has different colors for gutter vs line background:
	// AddedGutter background: RGB(0, 89, 0) -> "48;2;0;89;0" (stronger green)
	// Added line background: RGB(0, 38, 0) -> "48;2;0;38;0" (subtler green)
	theme := dv.TestTheme()
	m := bubbletea.NewModel(diff,
		bubbletea.WithTheme(theme),
		bubbletea.WithRenderer(trueColorRenderer()),
	)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// The padding space should use the line background color (0, 38, 0), not gutter (0, 89, 0)
	// The padding immediately follows the gutter, so we look for the pattern:
	// gutter ends with gutter-background -> padding has line-background
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasContent := bytes.Contains(out, []byte("+added"))
		// The padding space should have the line background color
		// Check that the output contains both the gutter background and line background colors
		hasGutterBackground := bytes.Contains(out, []byte("48;2;0;89;0"))
		hasLineBackground := bytes.Contains(out, []byte("48;2;0;38;0"))
		return hasContent && hasGutterBackground && hasLineBackground
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_WordDiffHighlighting(t *testing.T) {
	t.Parallel()

	// Create a diff with a paired delete/add line (a "replace" operation).
	// Word diff should highlight the changed portions within the lines.
	// "hello world" -> "hello universe"
	// - "world" should be highlighted in deleted line
	// - "universe" should be highlighted in added line
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

	// Create a mock word differ that returns segments
	wordDiffer := &mockWordDiffer{
		DiffFn: func(old, new string) (oldSegs, newSegs []diffview.Segment) {
			if old == "hello world" && new == "hello universe" {
				oldSegs = []diffview.Segment{
					{Text: "hello ", Changed: false},
					{Text: "world", Changed: true},
				}
				newSegs = []diffview.Segment{
					{Text: "hello ", Changed: false},
					{Text: "universe", Changed: true},
				}
			}
			return oldSegs, newSegs
		},
	}

	// TestTheme uses (GitHub-style - same foreground, gutter-intensity background):
	// AddedHighlight: gutter-intensity green background (35% blend) -> "48;2;0;89;0"
	// DeletedHighlight: gutter-intensity red background (35% blend) -> "48;2;89;0;0"
	// Added line (unchanged parts): dimmed green (15% blend) -> "48;2;0;38;0"
	// Deleted line (unchanged parts): dimmed red (15% blend) -> "48;2;38;0;0"
	theme := dv.TestTheme()
	m := bubbletea.NewModel(diff,
		bubbletea.WithTheme(theme),
		bubbletea.WithRenderer(trueColorRenderer()),
		bubbletea.WithWordDiffer(wordDiffer),
	)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Check that word-level highlighting is applied:
	// - Changed text should have gutter-intensity background (35% blend)
	// - Unchanged text should have dimmed background (15% blend)
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasDeletedLine := bytes.Contains(out, []byte("-hello"))
		hasAddedLine := bytes.Contains(out, []byte("+hello"))
		// Check for gutter-intensity highlight backgrounds (35% blend)
		hasDeletedHighlight := bytes.Contains(out, []byte("48;2;89;0;0")) // DeletedHighlight (gutter intensity)
		hasAddedHighlight := bytes.Contains(out, []byte("48;2;0;89;0"))   // AddedHighlight (gutter intensity)
		// Check for dimmed line backgrounds (15% blend)
		hasDimmedBackground := bytes.Contains(out, []byte("48;2;0;38;0")) || // Added dimmed
			bytes.Contains(out, []byte("48;2;38;0;0")) // Deleted dimmed
		return hasDeletedLine && hasAddedLine && hasDeletedHighlight && hasAddedHighlight && hasDimmedBackground
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

// mockWordDiffer implements diffview.WordDiffer for testing.
type mockWordDiffer struct {
	DiffFn func(old, new string) (oldSegs, newSegs []diffview.Segment)
}

func (m *mockWordDiffer) Diff(old, new string) (oldSegs, newSegs []diffview.Segment) {
	return m.DiffFn(old, new)
}

func TestModel_WordDiffHighlighting_NonPairedLinesNoHighlight(t *testing.T) {
	t.Parallel()

	// Non-paired lines (add without preceding delete) should NOT get word-level highlighting
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
							{Type: diffview.LineContext, Content: "unchanged", OldLineNum: 1, NewLineNum: 1},
							{Type: diffview.LineAdded, Content: "newly added", OldLineNum: 0, NewLineNum: 2},
						},
					},
				},
			},
		},
	}

	// Create a mock word differ that should NOT be called for non-paired lines
	wordDifferCalled := false
	wordDiffer := &mockWordDiffer{
		DiffFn: func(old, new string) (oldSegs, newSegs []diffview.Segment) {
			wordDifferCalled = true
			return nil, nil
		},
	}

	theme := dv.TestTheme()
	m := bubbletea.NewModel(diff,
		bubbletea.WithTheme(theme),
		bubbletea.WithRenderer(trueColorRenderer()),
		bubbletea.WithWordDiffer(wordDiffer),
	)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Wait for render - the added line should be present
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("+newly added"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))

	// Word differ should NOT have been called since there's no paired delete/add
	assert.False(t, wordDifferCalled, "WordDiffer should not be called for non-paired lines")
}

func TestModel_WordDiffHighlighting_MultiplePairs(t *testing.T) {
	t.Parallel()

	// Test multiple pairs in sequence
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
							// First pair
							{Type: diffview.LineDeleted, Content: "old line 1", OldLineNum: 1, NewLineNum: 0},
							{Type: diffview.LineAdded, Content: "new line 1", OldLineNum: 0, NewLineNum: 1},
							// Second pair
							{Type: diffview.LineDeleted, Content: "old line 2", OldLineNum: 2, NewLineNum: 0},
							{Type: diffview.LineAdded, Content: "new line 2", OldLineNum: 0, NewLineNum: 2},
						},
					},
				},
			},
		},
	}

	diffCallCount := 0
	wordDiffer := &mockWordDiffer{
		DiffFn: func(old, new string) (oldSegs, newSegs []diffview.Segment) {
			diffCallCount++
			// Return segments that mark the first word as unchanged, second as changed
			oldSegs = []diffview.Segment{
				{Text: "old ", Changed: false},
				{Text: "line " + old[len(old)-1:], Changed: true},
			}
			newSegs = []diffview.Segment{
				{Text: "new ", Changed: false},
				{Text: "line " + new[len(new)-1:], Changed: true},
			}
			return oldSegs, newSegs
		},
	}

	theme := dv.TestTheme()
	m := bubbletea.NewModel(diff,
		bubbletea.WithTheme(theme),
		bubbletea.WithRenderer(trueColorRenderer()),
		bubbletea.WithWordDiffer(wordDiffer),
	)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Wait for render - both pairs should be present
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasFirstDelete := bytes.Contains(out, []byte("-old"))
		hasFirstAdd := bytes.Contains(out, []byte("+new"))
		return hasFirstDelete && hasFirstAdd
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))

	// Word differ should be called twice, once for each pair
	assert.Equal(t, 2, diffCallCount, "WordDiffer should be called once per pair")
}

func TestModel_WordDiffHighlighting_ConsecutiveDeletesAndAdds(t *testing.T) {
	t.Parallel()

	// Test that consecutive deletes followed by consecutive adds are paired 1:1
	// This is common when changing multiple lines in a block
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
							// Two consecutive deletes
							{Type: diffview.LineDeleted, Content: `Foreground: "#1e1e2e",`, OldLineNum: 1, NewLineNum: 0},
							{Type: diffview.LineDeleted, Content: `Background: "#a6e3a1",`, OldLineNum: 2, NewLineNum: 0},
							// Two consecutive adds
							{Type: diffview.LineAdded, Content: `Foreground: "#cdd6f4",`, OldLineNum: 0, NewLineNum: 1},
							{Type: diffview.LineAdded, Content: `Background: "#3d5a3d",`, OldLineNum: 0, NewLineNum: 2},
						},
					},
				},
			},
		},
	}

	// Mock word differ that returns segments with shared structure
	pairsProcessed := make(map[string]bool)
	wordDiffer := &mockWordDiffer{
		DiffFn: func(old, new string) (oldSegs, newSegs []diffview.Segment) {
			pairsProcessed[old+"->"+new] = true
			// Simulate word diff where the color code is different but structure is shared
			// "Foreground: " is unchanged, the color code is changed
			if strings.HasPrefix(old, "Foreground") && strings.HasPrefix(new, "Foreground") {
				oldSegs = []diffview.Segment{
					{Text: `Foreground: "`, Changed: false},
					{Text: `#1e1e2e`, Changed: true},
					{Text: `",`, Changed: false},
				}
				newSegs = []diffview.Segment{
					{Text: `Foreground: "`, Changed: false},
					{Text: `#cdd6f4`, Changed: true},
					{Text: `",`, Changed: false},
				}
			} else if strings.HasPrefix(old, "Background") && strings.HasPrefix(new, "Background") {
				oldSegs = []diffview.Segment{
					{Text: `Background: "`, Changed: false},
					{Text: `#a6e3a1`, Changed: true},
					{Text: `",`, Changed: false},
				}
				newSegs = []diffview.Segment{
					{Text: `Background: "`, Changed: false},
					{Text: `#3d5a3d`, Changed: true},
					{Text: `",`, Changed: false},
				}
			}
			return oldSegs, newSegs
		},
	}

	theme := dv.TestTheme()
	m := bubbletea.NewModel(diff,
		bubbletea.WithTheme(theme),
		bubbletea.WithRenderer(trueColorRenderer()),
		bubbletea.WithWordDiffer(wordDiffer),
	)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasDeletedLine := bytes.Contains(out, []byte("-Foreground"))
		hasAddedLine := bytes.Contains(out, []byte("+Foreground"))
		// Check for gutter-intensity highlight backgrounds (word diff applied)
		hasHighlight := bytes.Contains(out, []byte("48;2;89;0;0")) || // DeletedHighlight
			bytes.Contains(out, []byte("48;2;0;89;0")) // AddedHighlight
		return hasDeletedLine && hasAddedLine && hasHighlight
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))

	// Verify correct pairing: 1st delete with 1st add, 2nd delete with 2nd add
	assert.True(t, pairsProcessed[`Foreground: "#1e1e2e",->`+`Foreground: "#cdd6f4",`],
		"1st delete should pair with 1st add")
	assert.True(t, pairsProcessed[`Background: "#a6e3a1",->`+`Background: "#3d5a3d",`],
		"2nd delete should pair with 2nd add")
}

func TestModel_WordDiffHighlighting_SkipsWhenLinesTooDifferent(t *testing.T) {
	t.Parallel()

	// When lines are too different (< 30% shared content), word-level diff should be skipped
	// to avoid highlighting everything as "changed" which is just noise
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
							{Type: diffview.LineDeleted, Content: "completely different old line", OldLineNum: 1, NewLineNum: 0},
							{Type: diffview.LineAdded, Content: "totally new content here", OldLineNum: 0, NewLineNum: 1},
						},
					},
				},
			},
		},
	}

	// Mock word differ that returns everything as changed (simulating very different lines)
	wordDifferCalled := false
	wordDiffer := &mockWordDiffer{
		DiffFn: func(old, new string) (oldSegs, newSegs []diffview.Segment) {
			wordDifferCalled = true
			// Return segments where everything is changed (no shared content)
			oldSegs = []diffview.Segment{
				{Text: old, Changed: true}, // 100% changed
			}
			newSegs = []diffview.Segment{
				{Text: new, Changed: true}, // 100% changed
			}
			return oldSegs, newSegs
		},
	}

	theme := dv.TestTheme()
	m := bubbletea.NewModel(diff,
		bubbletea.WithTheme(theme),
		bubbletea.WithRenderer(trueColorRenderer()),
		bubbletea.WithWordDiffer(wordDiffer),
	)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Wait for render
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasDeletedLine := bytes.Contains(out, []byte("-completely"))
		hasAddedLine := bytes.Contains(out, []byte("+totally"))
		// Should have dimmed backgrounds (no word-level highlighting applied)
		// because lines are too different
		hasDimmedBackground := bytes.Contains(out, []byte("48;2;0;38;0")) || // Added dimmed
			bytes.Contains(out, []byte("48;2;38;0;0")) // Deleted dimmed
		return hasDeletedLine && hasAddedLine && hasDimmedBackground
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))

	// Word differ should have been called (to compute segments)
	assert.True(t, wordDifferCalled, "WordDiffer should be called to compute segments")
}

func TestModel_WordDiffHighlighting_NoWordDiffer(t *testing.T) {
	t.Parallel()

	// When no WordDiffer is provided, rendering should work (graceful degradation)
	// Lines render with uniform line-level styling, no word-level segments
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

	// No WordDiffer provided - should render without crashing
	theme := dv.TestTheme()
	m := bubbletea.NewModel(diff,
		bubbletea.WithTheme(theme),
		bubbletea.WithRenderer(trueColorRenderer()),
		// Intentionally NOT setting WithWordDiffer
	)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Verify lines render correctly without WordDiffer
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasDeletedLine := bytes.Contains(out, []byte("-hello"))
		hasAddedLine := bytes.Contains(out, []byte("+hello"))
		// Lines should have dimmed background (15% blend), not gutter-intensity (35%)
		// since without word diff, entire line is uniformly styled
		hasDimmedAddedBackground := bytes.Contains(out, []byte("48;2;0;38;0"))
		hasDimmedDeletedBackground := bytes.Contains(out, []byte("48;2;38;0;0"))
		return hasDeletedLine && hasAddedLine && hasDimmedAddedBackground && hasDimmedDeletedBackground
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_ShowsEmptyFileCreation(t *testing.T) {
	t.Parallel()

	// Create a diff with an empty file creation (no hunks, but Operation=FileAdded)
	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				NewPath:   "empty.txt",
				Operation: diffview.FileAdded,
				// No hunks - this is an empty file
			},
		},
	}

	m := bubbletea.NewModel(diff)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Empty file should appear with filename and "(empty)" indicator
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasFilename := bytes.Contains(out, []byte("empty.txt"))
		hasEmptyIndicator := bytes.Contains(out, []byte("(empty)"))
		return hasFilename && hasEmptyIndicator
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestModel_ShowsEmptyFileDeletion(t *testing.T) {
	t.Parallel()

	// Create a diff with an empty file deletion (no hunks, but Operation=FileDeleted)
	diff := &diffview.Diff{
		Files: []diffview.FileDiff{
			{
				OldPath:   "deleted.txt",
				Operation: diffview.FileDeleted,
				// No hunks - this was an empty file that got deleted
			},
		},
	}

	m := bubbletea.NewModel(diff)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 24),
	)

	// Empty deleted file should appear with filename and "(empty)" indicator
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		hasFilename := bytes.Contains(out, []byte("deleted.txt"))
		hasEmptyIndicator := bytes.Contains(out, []byte("(empty)"))
		return hasFilename && hasEmptyIndicator
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}
