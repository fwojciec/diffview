package bubbletea_test

import (
	"bytes"
	"sync"
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
			Input: diffview.ClassificationInput{
				Repo:    "test-repo",
				Branch:  "test-branch",
				Commits: []diffview.CommitBrief{{Hash: "abc123"}},
			},
			Story: &diffview.StoryClassification{
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
			Input: diffview.ClassificationInput{
				Repo:    "test-repo",
				Branch:  "test-branch",
				Commits: []diffview.CommitBrief{{Hash: "abc123"}},
			},
			Story: &diffview.StoryClassification{
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
			Input: diffview.ClassificationInput{
				Repo:    "repo",
				Branch:  "first",
				Commits: []diffview.CommitBrief{{Hash: "first123"}},
			},
			Story: &diffview.StoryClassification{
				Summary: "First case summary",
			},
		},
		{
			Input: diffview.ClassificationInput{
				Repo:    "repo",
				Branch:  "second",
				Commits: []diffview.CommitBrief{{Hash: "second456"}},
			},
			Story: &diffview.StoryClassification{
				Summary: "Second case summary",
			},
		},
		{
			Input: diffview.ClassificationInput{
				Repo:    "repo",
				Branch:  "third",
				Commits: []diffview.CommitBrief{{Hash: "third789"}},
			},
			Story: &diffview.StoryClassification{
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
		{Input: diffview.ClassificationInput{Repo: "repo", Branch: "first", Commits: []diffview.CommitBrief{{Hash: "first"}}}, Story: &diffview.StoryClassification{Summary: "First summary"}},
		{Input: diffview.ClassificationInput{Repo: "repo", Branch: "second", Commits: []diffview.CommitBrief{{Hash: "second"}}}, Story: &diffview.StoryClassification{Summary: "Second summary"}},
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
			Input: diffview.ClassificationInput{
				Repo:    "test-repo",
				Branch:  "test-branch",
				Commits: []diffview.CommitBrief{{Hash: "abc123"}},
				Diff: diffview.Diff{
					Files: []diffview.FileDiff{
						{
							NewPath: "test.go",
							Hunks: []diffview.Hunk{
								{
									Lines: []diffview.Line{
										{Type: diffview.LineContext, Content: "diff content here"},
									},
								},
							},
						},
					},
				},
			},
			Story: &diffview.StoryClassification{
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
			Input: diffview.ClassificationInput{Repo: "repo", Branch: "branch", Commits: []diffview.CommitBrief{{Hash: "abc123"}}},
			Story: &diffview.StoryClassification{Summary: "Test story"},
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
			Input: diffview.ClassificationInput{Repo: "repo", Branch: "branch", Commits: []diffview.CommitBrief{{Hash: "abc123"}}},
			Story: &diffview.StoryClassification{Summary: "Test story"},
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
		{Input: diffview.ClassificationInput{Repo: "repo", Branch: "branch-a", Commits: []diffview.CommitBrief{{Hash: "abc"}}}, Story: &diffview.StoryClassification{Summary: "Case A"}},
		{Input: diffview.ClassificationInput{Repo: "repo", Branch: "branch-b", Commits: []diffview.CommitBrief{{Hash: "def"}}}, Story: &diffview.StoryClassification{Summary: "Case B"}},
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

func TestEvalModel_StatusBarShowsJudgmentIndicators(t *testing.T) {
	t.Parallel()

	cases := []diffview.EvalCase{
		{Input: diffview.ClassificationInput{Repo: "repo", Branch: "case1", Commits: []diffview.CommitBrief{{Hash: "case1"}}}, Story: &diffview.StoryClassification{Summary: "Case 1"}},
		{Input: diffview.ClassificationInput{Repo: "repo", Branch: "case2", Commits: []diffview.CommitBrief{{Hash: "case2"}}}, Story: &diffview.StoryClassification{Summary: "Case 2"}},
		{Input: diffview.ClassificationInput{Repo: "repo", Branch: "case3", Commits: []diffview.CommitBrief{{Hash: "case3"}}}, Story: &diffview.StoryClassification{Summary: "Case 3"}},
	}

	m := bubbletea.NewEvalModel(cases)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(100, 40),
	)

	// Initially all unjudged - should show 3 ○ indicators
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("○ ○ ○"))
	})

	// Mark first as pass - should show ✓ ○ ○
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("✓ ○ ○"))
	})

	// Navigate to second and mark as fail - should show ✓ ✗ ○
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("✓ ✗ ○"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestEvalModel_StatusBarShowsCritiqueIndicator(t *testing.T) {
	t.Parallel()

	cases := []diffview.EvalCase{
		{Input: diffview.ClassificationInput{Repo: "repo", Branch: "case1", Commits: []diffview.CommitBrief{{Hash: "case1"}}}, Story: &diffview.StoryClassification{Summary: "Case 1"}},
		{Input: diffview.ClassificationInput{Repo: "repo", Branch: "case2", Commits: []diffview.CommitBrief{{Hash: "case2"}}}, Story: &diffview.StoryClassification{Summary: "Case 2"}},
	}

	// Pre-load with a critique-only judgment (has critique but no pass/fail yet)
	judgments := []diffview.Judgment{
		{CaseID: "repo/case1", Critique: "Some critique text"},
	}

	m := bubbletea.NewEvalModel(cases, bubbletea.WithExistingJudgments(judgments))
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(100, 40),
	)

	// First case has critique without pass/fail - should show ● ○
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("● ○"))
	})

	// Mark as pass - should now show ✓ ○
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("✓ ○"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestEvalModel_JumpToNextUnjudged(t *testing.T) {
	t.Parallel()

	cases := []diffview.EvalCase{
		{Input: diffview.ClassificationInput{Repo: "repo", Branch: "case1", Commits: []diffview.CommitBrief{{Hash: "case1"}}}, Story: &diffview.StoryClassification{Summary: "Case 1"}},
		{Input: diffview.ClassificationInput{Repo: "repo", Branch: "case2", Commits: []diffview.CommitBrief{{Hash: "case2"}}}, Story: &diffview.StoryClassification{Summary: "Case 2"}},
		{Input: diffview.ClassificationInput{Repo: "repo", Branch: "case3", Commits: []diffview.CommitBrief{{Hash: "case3"}}}, Story: &diffview.StoryClassification{Summary: "Case 3"}},
		{Input: diffview.ClassificationInput{Repo: "repo", Branch: "case4", Commits: []diffview.CommitBrief{{Hash: "case4"}}}, Story: &diffview.StoryClassification{Summary: "Case 4"}},
	}

	// Pre-load judgments for cases 1 and 3 (indices 0 and 2)
	judgments := []diffview.Judgment{
		{CaseID: "repo/case1", Judged: true, Pass: true},
		{CaseID: "repo/case3", Judged: true, Pass: false},
	}

	m := bubbletea.NewEvalModel(cases, bubbletea.WithExistingJudgments(judgments))
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(100, 40),
	)

	// Starts on case 1, which is judged
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Case 1"))
	})

	// Press 'u' to jump to next unjudged - should go to case 2
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Case 2"))
	})

	// Press 'u' again - should jump to case 4 (case 3 is judged)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Case 4"))
	})

	// Press 'u' again - should wrap to case 2 (first unjudged)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Case 2"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestEvalModel_JumpToPrevUnjudged(t *testing.T) {
	t.Parallel()

	cases := []diffview.EvalCase{
		{Input: diffview.ClassificationInput{Repo: "repo", Branch: "case1", Commits: []diffview.CommitBrief{{Hash: "case1"}}}, Story: &diffview.StoryClassification{Summary: "Case 1"}},
		{Input: diffview.ClassificationInput{Repo: "repo", Branch: "case2", Commits: []diffview.CommitBrief{{Hash: "case2"}}}, Story: &diffview.StoryClassification{Summary: "Case 2"}},
		{Input: diffview.ClassificationInput{Repo: "repo", Branch: "case3", Commits: []diffview.CommitBrief{{Hash: "case3"}}}, Story: &diffview.StoryClassification{Summary: "Case 3"}},
		{Input: diffview.ClassificationInput{Repo: "repo", Branch: "case4", Commits: []diffview.CommitBrief{{Hash: "case4"}}}, Story: &diffview.StoryClassification{Summary: "Case 4"}},
	}

	// Pre-load judgments for cases 1 and 3 (indices 0 and 2)
	judgments := []diffview.Judgment{
		{CaseID: "repo/case1", Judged: true, Pass: true},
		{CaseID: "repo/case3", Judged: true, Pass: false},
	}

	m := bubbletea.NewEvalModel(cases, bubbletea.WithExistingJudgments(judgments))
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(100, 40),
	)

	// Navigate to case 3 first
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Case 1"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Case 3"))
	})

	// Press 'U' to jump to previous unjudged - should go to case 2
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'U'}})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Case 2"))
	})

	// Press 'U' again - should wrap to case 4 (last unjudged)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'U'}})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Case 4"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestEvalModel_StoryPanelShowsFullCritique(t *testing.T) {
	t.Parallel()

	longCritique := "This is a very long critique that should be displayed in full in the story panel without any truncation. It contains multiple sentences and detailed feedback about the classification quality."

	cases := []diffview.EvalCase{
		{Input: diffview.ClassificationInput{Repo: "repo", Branch: "case1", Commits: []diffview.CommitBrief{{Hash: "case1"}}}, Story: &diffview.StoryClassification{Summary: "Test Story"}},
	}

	// Pre-load with a judgment that has a long critique
	judgments := []diffview.Judgment{
		{CaseID: "repo/case1", Judged: true, Pass: false, Critique: longCritique},
	}

	m := bubbletea.NewEvalModel(cases, bubbletea.WithExistingJudgments(judgments))
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(120, 40),
	)

	// Full critique should appear in output (in the story panel)
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("CRITIQUE:")) &&
			bytes.Contains(out, []byte("without any truncation"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestEvalModel_JudgmentBarWithCritiqueOnly(t *testing.T) {
	t.Parallel()

	cases := []diffview.EvalCase{
		{Input: diffview.ClassificationInput{Repo: "repo", Branch: "case1", Commits: []diffview.CommitBrief{{Hash: "case1"}}}, Story: &diffview.StoryClassification{Summary: "Case 1"}},
	}

	// Case has critique but Judged is false (not explicitly passed/failed)
	judgments := []diffview.Judgment{
		{CaseID: "repo/case1", Judged: false, Critique: "Some critique text"},
	}

	m := bubbletea.NewEvalModel(cases, bubbletea.WithExistingJudgments(judgments))
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(100, 40),
	)

	// Judgment bar should show both markers as empty (not filled)
	// since Judged is false
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		// Should show empty circles for pass/fail, but critique text
		return bytes.Contains(out, []byte("○ Pass  ○ Fail")) &&
			bytes.Contains(out, []byte("Critique: Some critique text"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestEvalModel_StyledDiffRendering(t *testing.T) {
	t.Parallel()

	// Create a case with actual diff content
	cases := []diffview.EvalCase{
		{
			Input: diffview.ClassificationInput{
				Repo:    "test-repo",
				Branch:  "test-branch",
				Commits: []diffview.CommitBrief{{Hash: "abc123"}},
				Diff: diffview.Diff{
					Files: []diffview.FileDiff{
						{
							NewPath:   "test.go",
							Operation: diffview.FileModified,
							Hunks: []diffview.Hunk{
								{
									OldStart: 1,
									OldCount: 2,
									NewStart: 1,
									NewCount: 2,
									Lines: []diffview.Line{
										{Type: diffview.LineDeleted, Content: "old line", OldLineNum: 1},
										{Type: diffview.LineAdded, Content: "new line", NewLineNum: 1},
									},
								},
							},
						},
					},
				},
			},
			Story: &diffview.StoryClassification{Summary: "Test story"},
		},
	}

	m := bubbletea.NewEvalModel(cases)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(100, 40),
	)

	// Should render with styled file header (box-drawing characters)
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("── test.go"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestEvalModel_CopyCaseToClipboard(t *testing.T) {
	t.Parallel()

	cases := []diffview.EvalCase{
		{
			Input: diffview.ClassificationInput{
				Repo:    "test-repo",
				Branch:  "feature-branch",
				Commits: []diffview.CommitBrief{{Hash: "abc123", Message: "Add new feature"}},
				Diff: diffview.Diff{
					Files: []diffview.FileDiff{
						{
							NewPath:   "main.go",
							Operation: diffview.FileModified,
							Hunks: []diffview.Hunk{
								{
									OldStart: 10,
									OldCount: 3,
									NewStart: 10,
									NewCount: 4,
									Lines: []diffview.Line{
										{Type: diffview.LineContext, Content: "context line"},
										{Type: diffview.LineDeleted, Content: "old code", OldLineNum: 11},
										{Type: diffview.LineAdded, Content: "new code", NewLineNum: 11},
										{Type: diffview.LineAdded, Content: "more new code", NewLineNum: 12},
									},
								},
							},
						},
					},
				},
			},
			Story: &diffview.StoryClassification{
				ChangeType: "feature",
				Narrative:  "core-periphery",
				Summary:    "Added a new feature to the codebase",
				Sections: []diffview.Section{
					{
						Role:        "core",
						Title:       "Main implementation",
						Explanation: "The core logic for the feature",
						Hunks:       []diffview.HunkRef{{File: "main.go", HunkIndex: 0}},
					},
				},
			},
		},
	}

	// Create a mock clipboard to capture the copied content
	mockClipboard := &mockClipboard{}
	m := bubbletea.NewEvalModel(cases, bubbletea.WithClipboard(mockClipboard))
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(100, 40),
	)

	// Wait for model to be ready (content appears in viewport)
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Added a new feature"))
	})

	// Press 'y' to copy case to clipboard
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	// Wait for the clipboard to be populated - the copy happens synchronously
	// but we need to give the event loop a chance to process
	teatest.WaitFor(t, tm.Output(), func(_ []byte) bool {
		return mockClipboard.Content() != ""
	})

	// Verify clipboard received the formatted content
	content := mockClipboard.Content()
	assert.NotEmpty(t, content, "clipboard should have received content")

	// Check that the content includes key elements
	assert.Contains(t, content, "# Diff Classification Review")
	assert.Contains(t, content, "## Input: Raw Diff")
	assert.Contains(t, content, "test-repo")
	assert.Contains(t, content, "feature-branch")
	assert.Contains(t, content, "old code")
	assert.Contains(t, content, "new code")
	assert.Contains(t, content, "## Output: Story Classification")
	assert.Contains(t, content, "Change Type: feature")
	assert.Contains(t, content, "Narrative: core-periphery")
	assert.Contains(t, content, "Added a new feature to the codebase")
	assert.Contains(t, content, "## Your Task")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

// mockClipboard captures content for testing.
type mockClipboard struct {
	mu      sync.Mutex
	content string
}

func (m *mockClipboard) Copy(content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.content = content
	return nil
}

func (m *mockClipboard) Content() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.content
}
