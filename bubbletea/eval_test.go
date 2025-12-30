package bubbletea_test

import (
	"bytes"
	"sync"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/fwojciec/diffstory"
	"github.com/fwojciec/diffstory/bubbletea"
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

func TestEvalModel_CaseNavigationWithN(t *testing.T) {
	t.Parallel()

	// Per design doc: n/N navigates between cases (replacing ]/[)
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

	// Navigate to next case with 'n'
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	// Wait for second case to appear
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Second case"))
	})

	// Navigate back with 'N'
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})

	// Wait for first case to appear again
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("First case"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestEvalModel_NavigationBetweenCases(t *testing.T) {
	t.Parallel()

	// Tests navigation between cases: forward with n, backward with N.
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
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Second summary"))
	})

	// Navigate back to first
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("First summary"))
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
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

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
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
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

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

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

	// Wait for model to be ready (section info appears in viewport)
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Main implementation"))
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

func TestEvalModel_JKScrollsDiffViewport(t *testing.T) {
	t.Parallel()

	// Create a case with content that spans multiple lines
	cases := []diffview.EvalCase{
		{
			Input: diffview.ClassificationInput{
				Repo:    "repo",
				Branch:  "branch",
				Commits: []diffview.CommitBrief{{Hash: "abc"}},
				Diff: diffview.Diff{
					Files: []diffview.FileDiff{
						{
							NewPath: "test.go",
							Hunks: []diffview.Hunk{
								{
									Lines: []diffview.Line{
										{Type: diffview.LineContext, Content: "line 1"},
										{Type: diffview.LineContext, Content: "line 2"},
										{Type: diffview.LineContext, Content: "line 3"},
										{Type: diffview.LineContext, Content: "line 4"},
										{Type: diffview.LineContext, Content: "line 5"},
									},
								},
							},
						},
					},
				},
			},
			Story: &diffview.StoryClassification{Summary: "Test story"},
		},
		{
			Input: diffview.ClassificationInput{
				Repo:    "repo",
				Branch:  "branch2",
				Commits: []diffview.CommitBrief{{Hash: "def"}},
			},
			Story: &diffview.StoryClassification{Summary: "Second story"},
		},
	}

	m := bubbletea.NewEvalModel(cases)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 40),
	)

	// Wait for first case to appear
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Test story"))
	})

	// Press j - should scroll, NOT navigate to next case
	// This sends the key but doesn't change visible content
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	// Navigate to next case with n
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	// Now we should see second case
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Second story"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestEvalModel_HelpOverlayShowsOnQuestionMark(t *testing.T) {
	t.Parallel()

	cases := []diffview.EvalCase{
		{
			Input: diffview.ClassificationInput{Repo: "repo", Branch: "branch", Commits: []diffview.CommitBrief{{Hash: "abc"}}},
			Story: &diffview.StoryClassification{Summary: "Test story"},
		},
	}

	m := bubbletea.NewEvalModel(cases)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(80, 40),
	)

	// Wait for normal view to appear
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Test story"))
	})

	// Press '?' to show help overlay
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})

	// Help overlay should show keybindings
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("HELP")) &&
			bytes.Contains(out, []byte("pass")) &&
			bytes.Contains(out, []byte("fail")) &&
			bytes.Contains(out, []byte("quit"))
	})

	// Press '?' again to dismiss
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})

	// Should return to normal view
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("Test story")) &&
			!bytes.Contains(out, []byte("HELP"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestEvalModel_StoryModeIsDefaultWhenCaseHasStory(t *testing.T) {
	t.Parallel()

	// Create a case with sections - story mode should be the default
	cases := []diffview.EvalCase{
		{
			Input: diffview.ClassificationInput{
				Repo:    "test-repo",
				Branch:  "test-branch",
				Commits: []diffview.CommitBrief{{Hash: "abc123"}},
				Diff: diffview.Diff{
					Files: []diffview.FileDiff{
						{
							NewPath: "main.go",
							Hunks: []diffview.Hunk{
								{
									Lines: []diffview.Line{
										{Type: diffview.LineAdded, Content: "added line"},
									},
								},
							},
						},
					},
				},
			},
			Story: &diffview.StoryClassification{
				ChangeType: "feature",
				Summary:    "Added new feature",
				Sections: []diffview.Section{
					{
						Role:        "core",
						Title:       "Main implementation",
						Explanation: "Core logic for the feature",
						Hunks:       []diffview.HunkRef{{File: "main.go", HunkIndex: 0}},
					},
				},
			},
		},
	}

	m := bubbletea.NewEvalModel(cases)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(100, 40),
	)

	// Story mode should show "story mode" in the status bar
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("story mode"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestEvalModel_ToggleModeWithM(t *testing.T) {
	t.Parallel()

	// Create a case with sections
	cases := []diffview.EvalCase{
		{
			Input: diffview.ClassificationInput{
				Repo:    "test-repo",
				Branch:  "test-branch",
				Commits: []diffview.CommitBrief{{Hash: "abc123"}},
				Diff: diffview.Diff{
					Files: []diffview.FileDiff{
						{
							NewPath: "main.go",
							Hunks: []diffview.Hunk{
								{
									Lines: []diffview.Line{
										{Type: diffview.LineAdded, Content: "added line"},
									},
								},
							},
						},
					},
				},
			},
			Story: &diffview.StoryClassification{
				ChangeType: "feature",
				Summary:    "Added new feature",
				Sections: []diffview.Section{
					{
						Role:        "core",
						Title:       "Main implementation",
						Explanation: "Core logic for the feature",
						Hunks:       []diffview.HunkRef{{File: "main.go", HunkIndex: 0}},
					},
				},
			},
		},
	}

	m := bubbletea.NewEvalModel(cases)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(100, 40),
	)

	// Starts in story mode
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("story mode"))
	})

	// Press 'm' to toggle to raw mode
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})

	// Should now show "raw mode"
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("raw mode"))
	})

	// Press 'm' again to toggle back to story mode
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})

	// Should show "story mode" again
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("story mode"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestEvalModel_StoryModeShowsOnlyCurrentSectionHunks(t *testing.T) {
	t.Parallel()

	// Create a case with multiple sections, each with different hunks
	cases := []diffview.EvalCase{
		{
			Input: diffview.ClassificationInput{
				Repo:    "test-repo",
				Branch:  "test-branch",
				Commits: []diffview.CommitBrief{{Hash: "abc123"}},
				Diff: diffview.Diff{
					Files: []diffview.FileDiff{
						{
							NewPath: "main.go",
							Hunks: []diffview.Hunk{
								{Lines: []diffview.Line{{Type: diffview.LineAdded, Content: "SECTION_ONE_CONTENT"}}},
								{Lines: []diffview.Line{{Type: diffview.LineAdded, Content: "SECTION_TWO_CONTENT"}}},
							},
						},
					},
				},
			},
			Story: &diffview.StoryClassification{
				ChangeType: "feature",
				Summary:    "Multi-section feature",
				Sections: []diffview.Section{
					{
						Role:        "core",
						Title:       "First section",
						Explanation: "Core logic",
						Hunks:       []diffview.HunkRef{{File: "main.go", HunkIndex: 0}},
					},
					{
						Role:        "support",
						Title:       "Second section",
						Explanation: "Support code",
						Hunks:       []diffview.HunkRef{{File: "main.go", HunkIndex: 1}},
					},
				},
			},
		},
	}

	m := bubbletea.NewEvalModel(cases)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(100, 40),
	)

	// In story mode, section 1 should show SECTION_ONE_CONTENT but NOT SECTION_TWO_CONTENT
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("SECTION_ONE_CONTENT")) &&
			!bytes.Contains(out, []byte("SECTION_TWO_CONTENT"))
	})

	// Navigate to section 2
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})

	// Now should show SECTION_TWO_CONTENT but NOT SECTION_ONE_CONTENT
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("SECTION_TWO_CONTENT")) &&
			!bytes.Contains(out, []byte("SECTION_ONE_CONTENT"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestEvalModel_StoryModeShowsSectionHeader(t *testing.T) {
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
							NewPath: "main.go",
							Hunks: []diffview.Hunk{
								{Lines: []diffview.Line{{Type: diffview.LineAdded, Content: "code"}}},
							},
						},
					},
				},
			},
			Story: &diffview.StoryClassification{
				ChangeType: "feature",
				Summary:    "Test feature",
				Sections: []diffview.Section{
					{
						Role:        "core",
						Title:       "Main Implementation",
						Explanation: "The primary logic for the feature",
						Hunks:       []diffview.HunkRef{{File: "main.go", HunkIndex: 0}},
					},
				},
			},
		},
	}

	m := bubbletea.NewEvalModel(cases)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(100, 40),
	)

	// In story mode, should show section header with role and title
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("[core]")) &&
			bytes.Contains(out, []byte("Main Implementation"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestEvalModel_CaseNavigationResetsStoryModeState(t *testing.T) {
	t.Parallel()

	// Create two cases with different sections
	cases := []diffview.EvalCase{
		{
			Input: diffview.ClassificationInput{
				Repo:    "test-repo",
				Branch:  "case1",
				Commits: []diffview.CommitBrief{{Hash: "case1"}},
				Diff: diffview.Diff{
					Files: []diffview.FileDiff{
						{
							NewPath: "main.go",
							Hunks: []diffview.Hunk{
								{Lines: []diffview.Line{{Type: diffview.LineAdded, Content: "case1h1"}}},
								{Lines: []diffview.Line{{Type: diffview.LineAdded, Content: "case1h2"}}},
							},
						},
					},
				},
			},
			Story: &diffview.StoryClassification{
				ChangeType: "feature",
				Summary:    "Case 1",
				Sections: []diffview.Section{
					{Role: "core", Title: "Case1-S1", Hunks: []diffview.HunkRef{{File: "main.go", HunkIndex: 0}}},
					{Role: "support", Title: "Case1-S2", Hunks: []diffview.HunkRef{{File: "main.go", HunkIndex: 1}}},
				},
			},
		},
		{
			Input: diffview.ClassificationInput{
				Repo:    "test-repo",
				Branch:  "case2",
				Commits: []diffview.CommitBrief{{Hash: "case2"}},
				Diff: diffview.Diff{
					Files: []diffview.FileDiff{
						{
							NewPath: "other.go",
							Hunks: []diffview.Hunk{
								{Lines: []diffview.Line{{Type: diffview.LineAdded, Content: "case2h1"}}},
							},
						},
					},
				},
			},
			Story: &diffview.StoryClassification{
				ChangeType: "refactor",
				Summary:    "Case 2",
				Sections: []diffview.Section{
					{Role: "noise", Title: "Case2-S1", Hunks: []diffview.HunkRef{{File: "other.go", HunkIndex: 0}}},
				},
			},
		},
	}

	m := bubbletea.NewEvalModel(cases)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(100, 40),
	)

	// At case 1, navigate to section 2
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("section 1/2"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("section 2/2"))
	})

	// Navigate to case 2 - should reset to section 1/1
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("section 1/1")) &&
			bytes.Contains(out, []byte("Case2-S1"))
	})

	// Navigate back to case 1 - should reset to section 1/2 (not stay at 2/2)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'N'}})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("section 1/2")) &&
			bytes.Contains(out, []byte("Case1-S1"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestEvalModel_SectionProgressIndicator(t *testing.T) {
	t.Parallel()

	// Create a case with 3 sections
	cases := []diffview.EvalCase{
		{
			Input: diffview.ClassificationInput{
				Repo:    "test-repo",
				Branch:  "test-branch",
				Commits: []diffview.CommitBrief{{Hash: "abc123"}},
				Diff: diffview.Diff{
					Files: []diffview.FileDiff{
						{
							NewPath: "main.go",
							Hunks: []diffview.Hunk{
								{Lines: []diffview.Line{{Type: diffview.LineAdded, Content: "h1"}}},
								{Lines: []diffview.Line{{Type: diffview.LineAdded, Content: "h2"}}},
								{Lines: []diffview.Line{{Type: diffview.LineAdded, Content: "h3"}}},
							},
						},
					},
				},
			},
			Story: &diffview.StoryClassification{
				ChangeType: "feature",
				Summary:    "Test",
				Sections: []diffview.Section{
					{Role: "core", Title: "S1", Hunks: []diffview.HunkRef{{File: "main.go", HunkIndex: 0}}},
					{Role: "support", Title: "S2", Hunks: []diffview.HunkRef{{File: "main.go", HunkIndex: 1}}},
					{Role: "noise", Title: "S3", Hunks: []diffview.HunkRef{{File: "main.go", HunkIndex: 2}}},
				},
			},
		},
	}

	m := bubbletea.NewEvalModel(cases)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(100, 40),
	)

	// At section 1, should show progress: current section 1, none reviewed
	// Expect: ● ○ ○ (current is filled, pending are empty)
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("● ○ ○"))
	})

	// Navigate to section 2 - section 1 becomes reviewed
	// Expect: ✓ ● ○ (reviewed is check, current is filled)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("✓ ● ○"))
	})

	// Navigate to section 3
	// Expect: ✓ ✓ ●
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("✓ ✓ ●"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestEvalModel_SectionNavigationWithBrackets(t *testing.T) {
	t.Parallel()

	// Per design doc: ]/[ navigates between sections (replacing s/S)
	cases := []diffview.EvalCase{
		{
			Input: diffview.ClassificationInput{
				Repo:    "test-repo",
				Branch:  "test-branch",
				Commits: []diffview.CommitBrief{{Hash: "abc123"}},
				Diff: diffview.Diff{
					Files: []diffview.FileDiff{
						{
							NewPath: "main.go",
							Hunks: []diffview.Hunk{
								{Lines: []diffview.Line{{Type: diffview.LineAdded, Content: "hunk 1"}}},
								{Lines: []diffview.Line{{Type: diffview.LineAdded, Content: "hunk 2"}}},
								{Lines: []diffview.Line{{Type: diffview.LineAdded, Content: "hunk 3"}}},
							},
						},
					},
				},
			},
			Story: &diffview.StoryClassification{
				ChangeType: "feature",
				Summary:    "Multi-section feature",
				Sections: []diffview.Section{
					{
						Role:        "core",
						Title:       "First section",
						Explanation: "Core logic",
						Hunks:       []diffview.HunkRef{{File: "main.go", HunkIndex: 0}},
					},
					{
						Role:        "support",
						Title:       "Second section",
						Explanation: "Support code",
						Hunks:       []diffview.HunkRef{{File: "main.go", HunkIndex: 1}},
					},
					{
						Role:        "noise",
						Title:       "Third section",
						Explanation: "Minor changes",
						Hunks:       []diffview.HunkRef{{File: "main.go", HunkIndex: 2}},
					},
				},
			},
		},
	}

	m := bubbletea.NewEvalModel(cases)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(100, 40),
	)

	// Starts in story mode at section 1
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("section 1/3"))
	})

	// Press ']' to go to next section
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})

	// Should show section 2/3
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("section 2/3"))
	})

	// Press ']' again to go to section 3
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("section 3/3"))
	})

	// Press '[' to go back to section 2
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("section 2/3"))
	})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}

func TestRenderDataView_FormatsClassificationAsTree(t *testing.T) {
	t.Parallel()

	// Data View should show the classification as a structured tree:
	// - change_type, narrative, summary at top
	// - sections with role, explanation, hunk list
	// - each hunk with file:hunk_index, category, collapsed state

	story := &diffview.StoryClassification{
		ChangeType: "bugfix",
		Narrative:  "cause-effect",
		Summary:    "Fix incorrect metadata lookups in filtered diff views",
		Sections: []diffview.Section{
			{
				Role:        "problem",
				Title:       "Identify the Bug",
				Explanation: "Shows the buggy code that caused the issue",
				Hunks: []diffview.HunkRef{
					{File: "bubbletea/eval.go", HunkIndex: 0, Category: "core", Collapsed: false},
				},
			},
			{
				Role:        "fix",
				Title:       "Index Mapping Logic",
				Explanation: "This is the core of the fix",
				Hunks: []diffview.HunkRef{
					{File: "bubbletea/render.go", HunkIndex: 0, Category: "core", Collapsed: true},
					{File: "bubbletea/render.go", HunkIndex: 1, Category: "core", Collapsed: false},
				},
			},
		},
	}

	result := bubbletea.RenderDataView(story, 80)

	// Should show classification metadata
	assert.Contains(t, result, "change_type: bugfix")
	assert.Contains(t, result, "narrative:   cause-effect")
	assert.Contains(t, result, "summary:")
	assert.Contains(t, result, "Fix incorrect metadata lookups")

	// Should show sections header
	assert.Contains(t, result, "sections")

	// Should show section details with role badge
	assert.Contains(t, result, "[problem]")
	assert.Contains(t, result, "Identify the Bug")
	assert.Contains(t, result, "Shows the buggy code")

	assert.Contains(t, result, "[fix]")
	assert.Contains(t, result, "Index Mapping Logic")
	assert.Contains(t, result, "This is the core of the fix")

	// Should show hunk references with file:hunk_index, category, and collapse state
	assert.Contains(t, result, "bubbletea/eval.go:H0")
	assert.Contains(t, result, "core")
	assert.Contains(t, result, "visible")

	assert.Contains(t, result, "bubbletea/render.go:H0")
	assert.Contains(t, result, "collapsed")

	assert.Contains(t, result, "bubbletea/render.go:H1")
}

func TestRenderDataView_HandlesNilStory(t *testing.T) {
	t.Parallel()

	result := bubbletea.RenderDataView(nil, 80)

	assert.Contains(t, result, "Not yet classified")
}

func TestRenderDataView_HandlesEmptySections(t *testing.T) {
	t.Parallel()

	story := &diffview.StoryClassification{
		ChangeType: "refactor",
		Narrative:  "before-after",
		Summary:    "Simplify the code structure",
		Sections:   []diffview.Section{},
	}

	result := bubbletea.RenderDataView(story, 80)

	assert.Contains(t, result, "change_type: refactor")
	assert.Contains(t, result, "sections")
	// No section content but header should still be present
}

func TestRenderDataView_SectionWithNoHunks(t *testing.T) {
	t.Parallel()

	story := &diffview.StoryClassification{
		ChangeType: "docs",
		Narrative:  "explanatory",
		Summary:    "Update documentation",
		Sections: []diffview.Section{
			{
				Role:        "docs",
				Title:       "Documentation Update",
				Explanation: "Clarifies the API usage",
				Hunks:       nil, // No hunks in this section
			},
		},
	}

	result := bubbletea.RenderDataView(story, 80)

	assert.Contains(t, result, "[docs] Documentation Update")
	assert.Contains(t, result, "Clarifies the API usage")
	// Should NOT contain "hunks:" label when section has no hunks
	assert.NotContains(t, result, "hunks:")
}

func TestEvalModel_FilteredDiffUsesOriginalHunkIndices(t *testing.T) {
	t.Parallel()

	// Bug: When a section references non-first hunks from a file, collapsed state,
	// categories, and collapse text are incorrectly looked up because render uses
	// filtered slice indices instead of original hunk indices.
	//
	// This test creates:
	// - A file with 2 hunks (indices 0 and 1)
	// - Section 1 references only hunk 1 (not hunk 0)
	// - Section 2 references hunk 0
	// - Hunk 0 has category "systematic" and collapse text "Hunk zero text"
	// - Hunk 1 has category "core" and collapse text "Hunk one text"
	//
	// The bug: When rendering section 1, the filtered diff only contains hunk 1,
	// but at position 0 in the filtered slice. The render code uses the position (0)
	// to lookup the category/collapse text, getting hunk 0's data instead of hunk 1's.

	cases := []diffview.EvalCase{
		{
			Input: diffview.ClassificationInput{
				Repo:    "test-repo",
				Branch:  "test-branch",
				Commits: []diffview.CommitBrief{{Hash: "abc123"}},
				Diff: diffview.Diff{
					Files: []diffview.FileDiff{
						{
							NewPath: "file.go",
							Hunks: []diffview.Hunk{
								{
									// Hunk 0: NOT in section 1
									OldStart: 1, OldCount: 2, NewStart: 1, NewCount: 2,
									Lines: []diffview.Line{
										{Type: diffview.LineContext, Content: "HUNK_ZERO_CONTENT"},
										{Type: diffview.LineContext, Content: "more hunk zero"},
									},
								},
								{
									// Hunk 1: IS in section 1
									OldStart: 100, OldCount: 2, NewStart: 100, NewCount: 2,
									Lines: []diffview.Line{
										{Type: diffview.LineContext, Content: "HUNK_ONE_CONTENT"},
										{Type: diffview.LineContext, Content: "more hunk one"},
									},
								},
							},
						},
					},
				},
			},
			Story: &diffview.StoryClassification{
				ChangeType: "feature",
				Summary:    "Test filtered indices",
				Sections: []diffview.Section{
					{
						// Section 1 references hunk 1 only
						Role:        "core",
						Title:       "Core Changes",
						Explanation: "Main changes",
						Hunks: []diffview.HunkRef{
							{
								File:         "file.go",
								HunkIndex:    1, // Original index is 1
								Category:     "core",
								Collapsed:    true,
								CollapseText: "Hunk one text", // Should show this
							},
						},
					},
					{
						// Section 2 references hunk 0
						Role:        "supporting",
						Title:       "Supporting",
						Explanation: "Support changes",
						Hunks: []diffview.HunkRef{
							{
								File:         "file.go",
								HunkIndex:    0,
								Category:     "systematic",
								Collapsed:    true,
								CollapseText: "Hunk zero text", // Bug would incorrectly show this in section 1
							},
						},
					},
				},
			},
		},
	}

	m := bubbletea.NewEvalModel(cases)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(100, 40),
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

func TestEvalModel_SplitResize(t *testing.T) {
	t.Parallel()

	// Per design doc: +/- resizes the story/diff split ratio
	// This is a unit test of the key handling - the visual test is handled
	// by the existing story mode tests.
	cases := []diffview.EvalCase{
		{
			Input: diffview.ClassificationInput{
				Repo:    "test-repo",
				Branch:  "test-branch",
				Commits: []diffview.CommitBrief{{Hash: "abc123"}},
				Diff: diffview.Diff{
					Files: []diffview.FileDiff{
						{
							NewPath: "main.go",
							Hunks: []diffview.Hunk{
								{Lines: []diffview.Line{{Type: diffview.LineAdded, Content: "code"}}},
							},
						},
					},
				},
			},
			Story: &diffview.StoryClassification{
				ChangeType: "feature",
				Summary:    "Test feature",
				Sections: []diffview.Section{
					{
						Role:        "core",
						Title:       "Main Implementation",
						Explanation: "The primary logic for the feature",
						Hunks:       []diffview.HunkRef{{File: "main.go", HunkIndex: 0}},
					},
				},
			},
		},
	}

	m := bubbletea.NewEvalModel(cases)
	tm := teatest.NewTestModel(t, m,
		teatest.WithInitialTermSize(100, 40),
	)

	// Wait for story mode to be active first
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte("story mode"))
	})

	// Press '+' to increase metadata pane (key just needs to not crash)
	// The split ratio change is internal state, we just verify UI still works
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}})

	// Wait a tiny bit for the key to be processed, then press '-'
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}})

	// Quit and verify app exits normally
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(0))
}
