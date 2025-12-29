package main_test

import (
	"context"
	"errors"
	"testing"

	"github.com/fwojciec/diffstory"
	main "github.com/fwojciec/diffstory/cmd/diffstory"
	"github.com/fwojciec/diffstory/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApp_Run_GetsDiffFromGit(t *testing.T) {
	t.Parallel()

	diffFromGit := `diff --git a/feature.go b/feature.go
new file mode 100644
--- /dev/null
+++ b/feature.go
@@ -0,0 +1,3 @@
+package main
+
+func newFeature() {}
`

	app := &main.App{
		GitRunner: &mock.GitRunner{
			DiffRangeFn: func(_ context.Context, repoPath, base, head string) (string, error) {
				assert.Equal(t, "/repo", repoPath)
				assert.Equal(t, "main", base)
				assert.Equal(t, "HEAD", head)
				return diffFromGit, nil
			},
		},
		RepoPath:   "/repo",
		BaseBranch: "main",
		Classifier: &mock.StoryClassifier{
			ClassifyFn: func(_ context.Context, input diffview.ClassificationInput) (*diffview.StoryClassification, error) {
				require.Len(t, input.Diff.Files, 1)
				assert.Equal(t, "feature.go", input.Diff.Files[0].NewPath)
				return &diffview.StoryClassification{ChangeType: "feature"}, nil
			},
		},
	}

	diff, classification, err := app.Run(context.Background())
	require.NoError(t, err)
	require.NotNil(t, diff)
	require.NotNil(t, classification)
	assert.Len(t, diff.Files, 1)
	assert.Equal(t, "feature.go", diff.Files[0].NewPath)
}

func TestApp_Run_GitError(t *testing.T) {
	t.Parallel()

	app := &main.App{
		GitRunner: &mock.GitRunner{
			DiffRangeFn: func(_ context.Context, _, _, _ string) (string, error) {
				return "", errors.New("git diff failed: not a git repository")
			},
		},
		RepoPath:   "/not-a-repo",
		BaseBranch: "main",
		Classifier: &mock.StoryClassifier{
			ClassifyFn: func(_ context.Context, _ diffview.ClassificationInput) (*diffview.StoryClassification, error) {
				t.Error("Classifier should not be called when git fails")
				return nil, nil
			},
		},
	}

	_, _, err := app.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "git diff failed")
}

func TestApp_Run_EmptyDiff(t *testing.T) {
	t.Parallel()

	// When on main branch or no changes, git diff returns empty
	app := &main.App{
		GitRunner: &mock.GitRunner{
			DiffRangeFn: func(_ context.Context, _, _, _ string) (string, error) {
				return "", nil
			},
		},
		RepoPath:   "/repo",
		BaseBranch: "main",
		Classifier: &mock.StoryClassifier{
			ClassifyFn: func(_ context.Context, _ diffview.ClassificationInput) (*diffview.StoryClassification, error) {
				t.Error("Classifier should not be called for empty diff")
				return nil, nil
			},
		},
	}

	_, _, err := app.Run(context.Background())
	require.Error(t, err)
	assert.Equal(t, main.ErrNoChanges, err)
}

func TestApp_Run_ClassifierError(t *testing.T) {
	t.Parallel()

	diffFromGit := `diff --git a/hello.go b/hello.go
new file mode 100644
--- /dev/null
+++ b/hello.go
@@ -0,0 +1 @@
+package main
`

	app := &main.App{
		GitRunner: &mock.GitRunner{
			DiffRangeFn: func(_ context.Context, _, _, _ string) (string, error) {
				return diffFromGit, nil
			},
		},
		RepoPath:   "/repo",
		BaseBranch: "main",
		Classifier: &mock.StoryClassifier{
			ClassifyFn: func(_ context.Context, _ diffview.ClassificationInput) (*diffview.StoryClassification, error) {
				return nil, errors.New("API error")
			},
		},
	}

	_, _, err := app.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API error")
}

func TestApp_Run_PassesDiffToClassifier(t *testing.T) {
	t.Parallel()

	diffFromGit := `diff --git a/src/auth.go b/src/auth.go
index 0000000..e69de29
--- a/src/auth.go
+++ b/src/auth.go
@@ -1,3 +1,4 @@
 package auth

+func login() {}
 func logout() {}
`

	var capturedInput diffview.ClassificationInput
	app := &main.App{
		GitRunner: &mock.GitRunner{
			DiffRangeFn: func(_ context.Context, _, _, _ string) (string, error) {
				return diffFromGit, nil
			},
		},
		RepoPath:   "/repo",
		BaseBranch: "main",
		Classifier: &mock.StoryClassifier{
			ClassifyFn: func(_ context.Context, input diffview.ClassificationInput) (*diffview.StoryClassification, error) {
				capturedInput = input
				return &diffview.StoryClassification{ChangeType: "feature"}, nil
			},
		},
	}

	_, _, err := app.Run(context.Background())
	require.NoError(t, err)

	// Verify the diff was passed to the classifier
	require.Len(t, capturedInput.Diff.Files, 1)
	assert.Equal(t, "src/auth.go", capturedInput.Diff.Files[0].NewPath)
	require.Len(t, capturedInput.Diff.Files[0].Hunks, 1)
}

func TestApp_Run_UsesRawRangeWhenProvided(t *testing.T) {
	t.Parallel()

	diffFromGit := `diff --git a/feature.go b/feature.go
new file mode 100644
--- /dev/null
+++ b/feature.go
@@ -0,0 +1,3 @@
+package main
+
+func newFeature() {}
`

	var capturedRangeSpec string
	app := &main.App{
		GitRunner: &mock.GitRunner{
			DiffFn: func(_ context.Context, repoPath, rangeSpec string) (string, error) {
				capturedRangeSpec = rangeSpec
				assert.Equal(t, "/repo", repoPath)
				return diffFromGit, nil
			},
			// DiffRangeFn should NOT be called when Range is set
			DiffRangeFn: func(_ context.Context, _, _, _ string) (string, error) {
				t.Error("DiffRangeFn should not be called when Range is set")
				return "", nil
			},
		},
		RepoPath: "/repo",
		Range:    "main...feature-branch", // Raw range specification
		Classifier: &mock.StoryClassifier{
			ClassifyFn: func(_ context.Context, _ diffview.ClassificationInput) (*diffview.StoryClassification, error) {
				return &diffview.StoryClassification{ChangeType: "feature"}, nil
			},
		},
	}

	diff, classification, err := app.Run(context.Background())
	require.NoError(t, err)
	require.NotNil(t, diff)
	require.NotNil(t, classification)

	// Verify the raw range was passed directly to Diff
	assert.Equal(t, "main...feature-branch", capturedRangeSpec)
}

func TestParseRange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantBase  string
		wantHead  string
		wantError bool
	}{
		{
			name:     "three-dot notation",
			input:    "main...feature",
			wantBase: "main",
			wantHead: "feature",
		},
		{
			name:     "two-dot notation",
			input:    "HEAD~3..HEAD",
			wantBase: "HEAD~3",
			wantHead: "HEAD",
		},
		{
			name:     "origin prefix",
			input:    "origin/main...feature-branch",
			wantBase: "origin/main",
			wantHead: "feature-branch",
		},
		{
			name:     "commit hashes",
			input:    "abc123..def456",
			wantBase: "abc123",
			wantHead: "def456",
		},
		{
			name:      "no separator",
			input:     "main",
			wantError: true,
		},
		{
			name:      "empty base",
			input:     "...feature",
			wantError: true,
		},
		{
			name:      "empty head",
			input:     "main...",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			base, head, err := main.ParseRange(tt.input)

			if tt.wantError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantBase, base)
			assert.Equal(t, tt.wantHead, head)
		})
	}
}
