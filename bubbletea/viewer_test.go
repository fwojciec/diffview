package bubbletea_test

import (
	"bytes"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/fwojciec/diffview"
	"github.com/fwojciec/diffview/bubbletea"
	"github.com/stretchr/testify/assert"
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
