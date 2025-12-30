package diffview_test

import (
	"strings"
	"testing"

	"github.com/fwojciec/diffstory"
	"github.com/stretchr/testify/assert"
)

func TestDefaultFormatter_Format(t *testing.T) {
	t.Parallel()

	input := diffview.ClassificationInput{
		Repo: "testrepo",
		Commits: []diffview.CommitBrief{
			{Hash: "abc123", Message: "Fix authentication token expiry\n\nTokens were not being refreshed properly."},
		},
		Diff: diffview.Diff{
			Files: []diffview.FileDiff{
				{
					NewPath:   "pkg/auth/login.go",
					Operation: diffview.FileModified,
					Hunks: []diffview.Hunk{
						{
							OldStart: 45,
							OldCount: 6,
							NewStart: 45,
							NewCount: 10,
							Lines: []diffview.Line{
								{Type: diffview.LineContext, Content: "func (a *Auth) ValidateToken(token string) error {\n"},
								{Type: diffview.LineAdded, Content: "    if a.isExpired(token) {\n"},
								{Type: diffview.LineAdded, Content: "        return ErrTokenExpired\n"},
								{Type: diffview.LineAdded, Content: "    }\n"},
								{Type: diffview.LineContext, Content: "    return a.validator.Validate(token)\n"},
							},
						},
					},
				},
			},
		},
	}

	formatter := &diffview.DefaultFormatter{}
	result := formatter.Format(input)

	// Check context section
	assert.Contains(t, result, "<context>")
	assert.Contains(t, result, "Repository: testrepo")
	assert.Contains(t, result, "- Commit 1 [abc123]: Fix authentication token expiry")
	assert.Contains(t, result, "</context>")

	// Check diff section
	assert.Contains(t, result, "<diff>")
	assert.Contains(t, result, "</diff>")

	// Check file header
	assert.Contains(t, result, "=== FILE: pkg/auth/login.go (modified) ===")

	// Check hunk header with ID
	assert.Contains(t, result, "--- HUNK H1 (@@ -45,6 +45,10 @@) ---")

	// Check line prefixes
	assert.Contains(t, result, " func (a *Auth) ValidateToken(token string) error {")
	assert.Contains(t, result, "+    if a.isExpired(token) {")
}

func TestDefaultFormatter_Format_MultipleFiles(t *testing.T) {
	t.Parallel()

	input := diffview.ClassificationInput{
		Repo: "testrepo",
		Commits: []diffview.CommitBrief{
			{Hash: "def456", Message: "Add new feature"},
		},
		Diff: diffview.Diff{
			Files: []diffview.FileDiff{
				{
					NewPath:   "a.go",
					Operation: diffview.FileModified,
					Hunks: []diffview.Hunk{
						{OldStart: 1, OldCount: 1, NewStart: 1, NewCount: 2, Lines: []diffview.Line{
							{Type: diffview.LineContext, Content: "package a\n"},
							{Type: diffview.LineAdded, Content: "// comment\n"},
						}},
					},
				},
				{
					NewPath:   "b.go",
					Operation: diffview.FileAdded,
					Hunks: []diffview.Hunk{
						{OldStart: 0, OldCount: 0, NewStart: 1, NewCount: 1, Lines: []diffview.Line{
							{Type: diffview.LineAdded, Content: "package b\n"},
						}},
					},
				},
			},
		},
	}

	formatter := &diffview.DefaultFormatter{}
	result := formatter.Format(input)

	// Hunk IDs should be sequential across files
	assert.Contains(t, result, "--- HUNK H1")
	assert.Contains(t, result, "--- HUNK H2")

	// File operations should be correct
	assert.Contains(t, result, "=== FILE: a.go (modified) ===")
	assert.Contains(t, result, "=== FILE: b.go (added) ===")
}

func TestDefaultFormatter_Format_DeletedFile(t *testing.T) {
	t.Parallel()

	input := diffview.ClassificationInput{
		Repo: "testrepo",
		Commits: []diffview.CommitBrief{
			{Hash: "ghi789", Message: "Remove old file"},
		},
		Diff: diffview.Diff{
			Files: []diffview.FileDiff{
				{
					OldPath:   "old.go",
					Operation: diffview.FileDeleted,
					Hunks: []diffview.Hunk{
						{OldStart: 1, OldCount: 1, NewStart: 0, NewCount: 0, Lines: []diffview.Line{
							{Type: diffview.LineDeleted, Content: "package old\n"},
						}},
					},
				},
			},
		},
	}

	formatter := &diffview.DefaultFormatter{}
	result := formatter.Format(input)

	assert.Contains(t, result, "=== FILE: old.go (deleted) ===")
	assert.Contains(t, result, "-package old")
}

func TestDefaultFormatter_Format_HunkIDsAreSequential(t *testing.T) {
	t.Parallel()

	input := diffview.ClassificationInput{
		Commits: []diffview.CommitBrief{
			{Message: "test"},
		},
		Diff: diffview.Diff{
			Files: []diffview.FileDiff{
				{
					NewPath:   "a.go",
					Operation: diffview.FileModified,
					Hunks: []diffview.Hunk{
						{Lines: []diffview.Line{{Type: diffview.LineAdded, Content: "a\n"}}},
						{Lines: []diffview.Line{{Type: diffview.LineAdded, Content: "b\n"}}},
					},
				},
				{
					NewPath:   "b.go",
					Operation: diffview.FileModified,
					Hunks: []diffview.Hunk{
						{Lines: []diffview.Line{{Type: diffview.LineAdded, Content: "c\n"}}},
					},
				},
			},
		},
	}

	formatter := &diffview.DefaultFormatter{}
	result := formatter.Format(input)

	// Count occurrences of HUNK markers
	h1Count := strings.Count(result, "HUNK H1")
	h2Count := strings.Count(result, "HUNK H2")
	h3Count := strings.Count(result, "HUNK H3")

	assert.Equal(t, 1, h1Count, "Should have exactly one H1")
	assert.Equal(t, 1, h2Count, "Should have exactly one H2")
	assert.Equal(t, 1, h3Count, "Should have exactly one H3")
}

func TestDefaultFormatter_Format_ContextSection(t *testing.T) {
	t.Parallel()

	input := diffview.ClassificationInput{
		Repo:   "diffview",
		Branch: "diffview-zv1",
		Commits: []diffview.CommitBrief{
			{Hash: "af44c89", Message: "Address PR feedback"},
			{Hash: "51fad8d", Message: "Fix blank lines"},
		},
		Diff: diffview.Diff{
			Files: []diffview.FileDiff{
				{
					NewPath:   "main.go",
					Operation: diffview.FileModified,
					Hunks: []diffview.Hunk{
						{Lines: []diffview.Line{{Type: diffview.LineAdded, Content: "// test\n"}}},
					},
				},
			},
		},
	}

	formatter := &diffview.DefaultFormatter{}
	result := formatter.Format(input)

	// Should have context section, not commit_message
	assert.NotContains(t, result, "<commit_message>")
	assert.Contains(t, result, "<context>")
	assert.Contains(t, result, "</context>")

	// Should have repo and branch
	assert.Contains(t, result, "Repository: diffview")
	assert.Contains(t, result, "Branch: diffview-zv1")

	// Should have commits section with all commits
	assert.Contains(t, result, "Commits:")
	assert.Contains(t, result, "- Commit 1 [af44c89]: Address PR feedback")
	assert.Contains(t, result, "- Commit 2 [51fad8d]: Fix blank lines")
}

func TestDefaultFormatter_Format_ContextSection_EmptyBranch(t *testing.T) {
	t.Parallel()

	input := diffview.ClassificationInput{
		Repo:   "diffview",
		Branch: "", // Empty - single commit fallback mode
		Commits: []diffview.CommitBrief{
			{Hash: "abc123", Message: "Single commit"},
		},
		Diff: diffview.Diff{
			Files: []diffview.FileDiff{
				{
					NewPath:   "main.go",
					Operation: diffview.FileModified,
					Hunks: []diffview.Hunk{
						{Lines: []diffview.Line{{Type: diffview.LineAdded, Content: "// test\n"}}},
					},
				},
			},
		},
	}

	formatter := &diffview.DefaultFormatter{}
	result := formatter.Format(input)

	// Should have context section
	assert.Contains(t, result, "<context>")
	assert.Contains(t, result, "</context>")

	// Should have repo but NOT branch line
	assert.Contains(t, result, "Repository: diffview")
	assert.NotContains(t, result, "Branch:")

	// Should still have commit
	assert.Contains(t, result, "- Commit 1 [abc123]: Single commit")
}

func TestDefaultFormatter_Format_ContextSection_EmptyCommits(t *testing.T) {
	t.Parallel()

	input := diffview.ClassificationInput{
		Repo:    "diffview",
		Branch:  "feature-branch",
		Commits: []diffview.CommitBrief{}, // Empty
		Diff: diffview.Diff{
			Files: []diffview.FileDiff{
				{
					NewPath:   "main.go",
					Operation: diffview.FileModified,
					Hunks: []diffview.Hunk{
						{Lines: []diffview.Line{{Type: diffview.LineAdded, Content: "// test\n"}}},
					},
				},
			},
		},
	}

	formatter := &diffview.DefaultFormatter{}
	result := formatter.Format(input)

	// Should have context section
	assert.Contains(t, result, "<context>")
	assert.Contains(t, result, "</context>")

	// Should have repo and branch
	assert.Contains(t, result, "Repository: diffview")
	assert.Contains(t, result, "Branch: feature-branch")

	// Should NOT have Commits section
	assert.NotContains(t, result, "Commits:")
}

func TestDefaultFormatter_Format_ContextSection_WithPRMetadata(t *testing.T) {
	t.Parallel()

	input := diffview.ClassificationInput{
		Repo:          "diffview",
		Branch:        "feature-branch",
		PRTitle:       "Add dark mode support",
		PRDescription: "This PR adds dark mode toggle to the settings page.\n\nCloses #123",
		Commits: []diffview.CommitBrief{
			{Hash: "abc123", Message: "Add dark mode toggle"},
		},
		Diff: diffview.Diff{
			Files: []diffview.FileDiff{
				{
					NewPath:   "main.go",
					Operation: diffview.FileModified,
					Hunks: []diffview.Hunk{
						{Lines: []diffview.Line{{Type: diffview.LineAdded, Content: "// test\n"}}},
					},
				},
			},
		},
	}

	formatter := &diffview.DefaultFormatter{}
	result := formatter.Format(input)

	// Should have context section with PR metadata
	assert.Contains(t, result, "<context>")
	assert.Contains(t, result, "PR Title: Add dark mode support")
	assert.Contains(t, result, "PR Description:")
	assert.Contains(t, result, "This PR adds dark mode toggle to the settings page.")
	assert.Contains(t, result, "Closes #123")
	assert.Contains(t, result, "</context>")
}

func TestDefaultFormatter_Format_ContextSection_PRTitleOnly(t *testing.T) {
	t.Parallel()

	input := diffview.ClassificationInput{
		Repo:    "diffview",
		Branch:  "feature-branch",
		PRTitle: "Quick fix for login",
		// No description
		Commits: []diffview.CommitBrief{
			{Hash: "abc123", Message: "Fix login"},
		},
		Diff: diffview.Diff{
			Files: []diffview.FileDiff{
				{
					NewPath:   "main.go",
					Operation: diffview.FileModified,
					Hunks: []diffview.Hunk{
						{Lines: []diffview.Line{{Type: diffview.LineAdded, Content: "// test\n"}}},
					},
				},
			},
		},
	}

	formatter := &diffview.DefaultFormatter{}
	result := formatter.Format(input)

	// Should have PR title but not description label
	assert.Contains(t, result, "PR Title: Quick fix for login")
	assert.NotContains(t, result, "PR Description:")
}

func TestDefaultFormatter_Format_WithPerCommitDiffs(t *testing.T) {
	t.Parallel()

	input := diffview.ClassificationInput{
		Repo: "testrepo",
		Commits: []diffview.CommitBrief{
			{
				Hash:    "abc123",
				Message: "Add feature",
				Diff: &diffview.Diff{
					Files: []diffview.FileDiff{
						{
							NewPath:   "feature.go",
							Operation: diffview.FileAdded,
							Hunks: []diffview.Hunk{
								{Lines: []diffview.Line{
									{Type: diffview.LineAdded, Content: "package main\n"},
									{Type: diffview.LineAdded, Content: "func Feature() {}\n"},
								}},
							},
						},
					},
				},
			},
			{
				Hash:    "def456",
				Message: "Add tests",
				Diff: &diffview.Diff{
					Files: []diffview.FileDiff{
						{
							NewPath:   "feature_test.go",
							Operation: diffview.FileAdded,
							Hunks: []diffview.Hunk{
								{Lines: []diffview.Line{
									{Type: diffview.LineAdded, Content: "package main\n"},
								}},
							},
						},
					},
				},
			},
		},
		Diff: diffview.Diff{
			Files: []diffview.FileDiff{
				{NewPath: "feature.go", Operation: diffview.FileAdded, Hunks: []diffview.Hunk{}},
			},
		},
	}

	formatter := &diffview.DefaultFormatter{}
	result := formatter.Format(input)

	// Should have commit-diffs section
	assert.Contains(t, result, "<commit-diffs>")
	assert.Contains(t, result, "=== COMMIT 1 [abc123]: Add feature ===")
	assert.Contains(t, result, "feature.go (added): +2/-0")
	assert.Contains(t, result, "=== COMMIT 2 [def456]: Add tests ===")
	assert.Contains(t, result, "feature_test.go (added): +1/-0")
	assert.Contains(t, result, "</commit-diffs>")
}

func TestDefaultFormatter_Format_NoPerCommitDiffs(t *testing.T) {
	t.Parallel()

	input := diffview.ClassificationInput{
		Repo: "testrepo",
		Commits: []diffview.CommitBrief{
			{Hash: "abc123", Message: "Some commit"}, // No Diff field
		},
		Diff: diffview.Diff{
			Files: []diffview.FileDiff{
				{NewPath: "main.go", Operation: diffview.FileModified, Hunks: []diffview.Hunk{
					{Lines: []diffview.Line{{Type: diffview.LineAdded, Content: "// test\n"}}},
				}},
			},
		},
	}

	formatter := &diffview.DefaultFormatter{}
	result := formatter.Format(input)

	// Should NOT have commit-diffs section
	assert.NotContains(t, result, "<commit-diffs>")
}

func TestDefaultFormatter_Format_PartialPerCommitDiffs(t *testing.T) {
	t.Parallel()

	input := diffview.ClassificationInput{
		Repo: "testrepo",
		Commits: []diffview.CommitBrief{
			{
				Hash:    "abc123",
				Message: "Add feature",
				Diff: &diffview.Diff{
					Files: []diffview.FileDiff{
						{
							NewPath:   "feature.go",
							Operation: diffview.FileAdded,
							Hunks: []diffview.Hunk{
								{Lines: []diffview.Line{
									{Type: diffview.LineAdded, Content: "package main\n"},
								}},
							},
						},
					},
				},
			},
			{Hash: "def456", Message: "Fixup commit"}, // No Diff field
			{
				Hash:    "ghi789",
				Message: "Add more",
				Diff: &diffview.Diff{
					Files: []diffview.FileDiff{
						{
							NewPath:   "more.go",
							Operation: diffview.FileAdded,
							Hunks: []diffview.Hunk{
								{Lines: []diffview.Line{
									{Type: diffview.LineAdded, Content: "package main\n"},
								}},
							},
						},
					},
				},
			},
		},
		Diff: diffview.Diff{
			Files: []diffview.FileDiff{
				{NewPath: "feature.go", Operation: diffview.FileAdded, Hunks: []diffview.Hunk{}},
			},
		},
	}

	formatter := &diffview.DefaultFormatter{}
	result := formatter.Format(input)

	// Should have commit-diffs section (some commits have diffs)
	assert.Contains(t, result, "<commit-diffs>")
	// Should include commits with diffs
	assert.Contains(t, result, "=== COMMIT 1 [abc123]: Add feature ===")
	assert.Contains(t, result, "=== COMMIT 3 [ghi789]: Add more ===")
	// Should NOT include commit without diff
	assert.NotContains(t, result, "COMMIT 2 [def456]")
	assert.Contains(t, result, "</commit-diffs>")
}
