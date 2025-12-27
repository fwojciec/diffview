package diffview_test

import (
	"strings"
	"testing"

	"github.com/fwojciec/diffview"
	"github.com/stretchr/testify/assert"
)

func TestDefaultFormatter_Format(t *testing.T) {
	t.Parallel()

	input := diffview.ClassificationInput{
		Commit: diffview.CommitInfo{
			Hash:    "abc123",
			Repo:    "testrepo",
			Message: "Fix authentication token expiry\n\nTokens were not being refreshed properly.",
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

	// Check commit message section
	assert.Contains(t, result, "<commit_message>")
	assert.Contains(t, result, "Fix authentication token expiry")
	assert.Contains(t, result, "</commit_message>")

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
		Commit: diffview.CommitInfo{
			Hash:    "def456",
			Repo:    "testrepo",
			Message: "Add new feature",
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
		Commit: diffview.CommitInfo{
			Hash:    "ghi789",
			Repo:    "testrepo",
			Message: "Remove old file",
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
		Commit: diffview.CommitInfo{Message: "test"},
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
