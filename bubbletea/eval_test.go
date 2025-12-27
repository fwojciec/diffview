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

func TestEvalModel_Init(t *testing.T) {
	t.Parallel()

	cases := []diffview.EvalCase{
		{
			Commit: "abc123",
			Story: diffview.StoryAnalysis{
				ChangeType: "refactor",
				Summary:    "Refactored foo",
			},
		},
	}

	m := bubbletea.NewEvalModel(cases)
	cmd := m.Init()

	assert.Nil(t, cmd, "Init should return nil command")
}

func TestEvalModel_ViewBeforeReady(t *testing.T) {
	t.Parallel()

	m := bubbletea.NewEvalModel(nil)
	view := m.View()

	assert.Contains(t, view, "Loading", "View should show loading state before WindowSizeMsg")
}

func TestEvalModel_ViewAfterReady(t *testing.T) {
	t.Parallel()

	cases := []diffview.EvalCase{
		{
			Commit: "abc123",
			Story: diffview.StoryAnalysis{
				ChangeType: "refactor",
				Summary:    "Refactored foo for better performance",
			},
		},
	}

	m := bubbletea.NewEvalModel(cases)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 40),
	)

	// Wait for content to appear
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Refactored foo"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestEvalModel_QuitOnQ(t *testing.T) {
	t.Parallel()

	m := bubbletea.NewEvalModel(nil)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 40),
	)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestEvalModel_NavigationWithJK(t *testing.T) {
	t.Parallel()

	cases := []diffview.EvalCase{
		{
			Commit: "first123",
			Story: diffview.StoryAnalysis{
				Summary: "First case summary",
			},
		},
		{
			Commit: "second456",
			Story: diffview.StoryAnalysis{
				Summary: "Second case summary",
			},
		},
		{
			Commit: "third789",
			Story: diffview.StoryAnalysis{
				Summary: "Third case summary",
			},
		},
	}

	m := bubbletea.NewEvalModel(cases)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 40),
	)

	// Wait for first case to appear
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("First case"))
	})

	// Navigate to next case with 'j'
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	// Wait for second case to appear
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Second case"))
	})

	// Navigate back with 'k'
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})

	// Wait for first case to appear again
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("First case"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestEvalModel_NavigationBetweenCases(t *testing.T) {
	t.Parallel()

	// Tests navigation between cases: forward with j, backward with k.
	cases := []diffview.EvalCase{
		{Commit: "first", Story: diffview.StoryAnalysis{Summary: "First summary"}},
		{Commit: "second", Story: diffview.StoryAnalysis{Summary: "Second summary"}},
	}

	m := bubbletea.NewEvalModel(cases)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 40),
	)

	// Wait for first case
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("First summary"))
	})

	// Navigate forward then back
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Second summary"))
	})

	// Navigate back to first
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("First summary"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestEvalModel_PanelSwitching(t *testing.T) {
	t.Parallel()

	cases := []diffview.EvalCase{
		{
			Commit: "abc123",
			Hunks: []diffview.AnnotatedHunk{
				{
					ID: "h0",
					Hunk: diffview.Hunk{
						Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "diff content here"},
						},
					},
				},
			},
			Story: diffview.StoryAnalysis{
				Summary: "Story content here",
			},
		},
	}

	m := bubbletea.NewEvalModel(cases)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 40),
	)

	// Wait for initial content
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("DIFF"))
	})

	// Switch to story panel with 's'
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	// Should show STORY as active
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("STORY [active]"))
	})

	// Switch back to diff panel with 'd'
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("DIFF [active]"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestEvalModel_PassJudgment(t *testing.T) {
	t.Parallel()

	cases := []diffview.EvalCase{
		{
			Commit: "abc123",
			Story:  diffview.StoryAnalysis{Summary: "Test story"},
		},
	}

	m := bubbletea.NewEvalModel(cases)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 40),
	)

	// Wait for case to appear
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Test story"))
	})

	// Press 'p' to mark as pass
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})

	// Should show pass marker filled
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("● Pass"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestEvalModel_FailJudgment(t *testing.T) {
	t.Parallel()

	cases := []diffview.EvalCase{
		{
			Commit: "abc123",
			Story:  diffview.StoryAnalysis{Summary: "Test story"},
		},
	}

	m := bubbletea.NewEvalModel(cases)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 40),
	)

	// Wait for case to appear
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Test story"))
	})

	// Press 'f' to mark as fail
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})

	// Should show fail marker filled
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("● Fail"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestEvalModel_JudgmentUpdatesProgress(t *testing.T) {
	t.Parallel()

	cases := []diffview.EvalCase{
		{Commit: "abc", Story: diffview.StoryAnalysis{Summary: "Case A"}},
		{Commit: "def", Story: diffview.StoryAnalysis{Summary: "Case B"}},
	}

	m := bubbletea.NewEvalModel(cases)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 40),
	)

	// Wait for first case
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("0/2 reviewed"))
	})

	// Judge first case
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("1/2 reviewed"))
	})

	// Navigate to second and judge
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Case B"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("2/2 reviewed"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}
