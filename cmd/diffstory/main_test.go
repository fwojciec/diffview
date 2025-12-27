package main_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fwojciec/diffview"
	main "github.com/fwojciec/diffview/cmd/diffstory"
	"github.com/fwojciec/diffview/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApp_Run_OutputsValidJSON(t *testing.T) {
	t.Parallel()

	diffInput := `diff --git a/hello.go b/hello.go
new file mode 100644
index 0000000..e69de29
--- /dev/null
+++ b/hello.go
@@ -0,0 +1,3 @@
+package main
+
+func hello() {}
`

	var stdout bytes.Buffer
	app := &main.App{
		Input:  strings.NewReader(diffInput),
		Output: &stdout,
		Generator: &mock.StoryGenerator{
			GenerateFn: func(_ context.Context, _ []diffview.AnnotatedHunk) (*diffview.DiffAnalysis, error) {
				return &diffview.DiffAnalysis{
					Version: 1,
					Analyses: []diffview.Analysis{
						{
							Type:    "story",
							Payload: []byte(`{"changeType":"feature","summary":"Add hello function","parts":[]}`),
						},
					},
				}, nil
			},
		},
	}

	err := app.Run(context.Background())
	require.NoError(t, err)

	// Output should be valid JSON containing the analysis
	output := stdout.String()
	assert.Contains(t, output, `"Version"`)
	assert.Contains(t, output, `"Analyses"`)
}

func TestApp_Run_ReadsFromFilePath(t *testing.T) {
	t.Parallel()

	diffContent := `diff --git a/hello.go b/hello.go
new file mode 100644
index 0000000..e69de29
--- /dev/null
+++ b/hello.go
@@ -0,0 +1,3 @@
+package main
+
+func hello() {}
`
	// Create a temp file with the diff
	tmpDir := t.TempDir()
	diffPath := filepath.Join(tmpDir, "test.patch")
	err := os.WriteFile(diffPath, []byte(diffContent), 0o644)
	require.NoError(t, err)

	var stdout bytes.Buffer
	app := &main.App{
		FilePath: diffPath,
		Output:   &stdout,
		Generator: &mock.StoryGenerator{
			GenerateFn: func(_ context.Context, _ []diffview.AnnotatedHunk) (*diffview.DiffAnalysis, error) {
				return &diffview.DiffAnalysis{Version: 1}, nil
			},
		},
	}

	err = app.Run(context.Background())
	require.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, `"Version"`)
}

func TestApp_Run_IncludesFilePathInHunkID(t *testing.T) {
	t.Parallel()

	diffInput := `diff --git a/src/auth.go b/src/auth.go
index 0000000..e69de29
--- a/src/auth.go
+++ b/src/auth.go
@@ -1,3 +1,4 @@
 package auth

+func login() {}
 func logout() {}
`

	var capturedHunks []diffview.AnnotatedHunk
	var stdout bytes.Buffer
	app := &main.App{
		Input:  strings.NewReader(diffInput),
		Output: &stdout,
		Generator: &mock.StoryGenerator{
			GenerateFn: func(_ context.Context, hunks []diffview.AnnotatedHunk) (*diffview.DiffAnalysis, error) {
				capturedHunks = hunks
				return &diffview.DiffAnalysis{Version: 1}, nil
			},
		},
	}

	err := app.Run(context.Background())
	require.NoError(t, err)

	require.Len(t, capturedHunks, 1)
	// The hunk ID should contain the file path for context
	assert.Contains(t, capturedHunks[0].ID, "src/auth.go")
}

func TestApp_Run_GeneratorError(t *testing.T) {
	t.Parallel()

	diffInput := `diff --git a/hello.go b/hello.go
new file mode 100644
--- /dev/null
+++ b/hello.go
@@ -0,0 +1 @@
+package main
`

	var stdout bytes.Buffer
	app := &main.App{
		Input:  strings.NewReader(diffInput),
		Output: &stdout,
		Generator: &mock.StoryGenerator{
			GenerateFn: func(_ context.Context, _ []diffview.AnnotatedHunk) (*diffview.DiffAnalysis, error) {
				return nil, errors.New("API error")
			},
		},
	}

	err := app.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API error")
}

func TestApp_Run_FileNotFound(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	app := &main.App{
		FilePath: "/nonexistent/path/to/diff.patch",
		Output:   &stdout,
		Generator: &mock.StoryGenerator{
			GenerateFn: func(_ context.Context, _ []diffview.AnnotatedHunk) (*diffview.DiffAnalysis, error) {
				return &diffview.DiffAnalysis{Version: 1}, nil
			},
		},
	}

	err := app.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no such file")
}

func TestApp_Run_EmptyDiff(t *testing.T) {
	t.Parallel()

	// Empty input - no diff content at all
	diffInput := ""

	var stdout bytes.Buffer
	app := &main.App{
		Input:  strings.NewReader(diffInput),
		Output: &stdout,
		Generator: &mock.StoryGenerator{
			GenerateFn: func(_ context.Context, _ []diffview.AnnotatedHunk) (*diffview.DiffAnalysis, error) {
				t.Error("Generator should not be called for empty diff")
				return &diffview.DiffAnalysis{Version: 1}, nil
			},
		},
	}

	err := app.Run(context.Background())
	require.Error(t, err)
	assert.Equal(t, main.ErrNoChanges, err)
}

func TestCollector_Run_WritesJSONL(t *testing.T) {
	t.Parallel()

	diffOutput := `diff --git a/hello.go b/hello.go
new file mode 100644
index 0000000..e69de29
--- /dev/null
+++ b/hello.go
@@ -0,0 +1,3 @@
+package main
+
+func hello() {}
`

	var stdout bytes.Buffer
	collector := &main.Collector{
		Output:   &stdout,
		RepoName: "testrepo",
		Git: &mock.GitRunner{
			// No merge commits - triggers fallback to commit-level
			MergeCommitsFn: func(_ context.Context, _ string, _ int) ([]string, error) {
				return nil, nil
			},
			LogFn: func(_ context.Context, _ string, _ int) ([]string, error) {
				return []string{"abc1234"}, nil
			},
			ShowFn: func(_ context.Context, _ string, hash string) (string, error) {
				if hash == "abc1234" {
					return diffOutput, nil
				}
				return "", errors.New("unknown hash")
			},
			MessageFn: func(_ context.Context, _ string, hash string) (string, error) {
				return "Add hello function", nil
			},
		},
	}

	err := collector.Run(context.Background())
	require.NoError(t, err)

	// Output should be JSONL with one line per commit
	output := stdout.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	require.Len(t, lines, 1)

	// Line should contain commit hash and diff files (new lowercase JSON keys)
	assert.Contains(t, lines[0], `"hash":"abc1234"`)
	assert.Contains(t, lines[0], `"repo":"testrepo"`)
	assert.Contains(t, lines[0], `"message":"Add hello function"`)
	assert.Contains(t, lines[0], `"Files"`)
}

func TestCollector_Run_MultipleCommits(t *testing.T) {
	t.Parallel()

	diff1 := `diff --git a/a.go b/a.go
new file mode 100644
--- /dev/null
+++ b/a.go
@@ -0,0 +1 @@
+package a
`
	diff2 := `diff --git a/b.go b/b.go
new file mode 100644
--- /dev/null
+++ b/b.go
@@ -0,0 +1 @@
+package b
`

	var stdout bytes.Buffer
	collector := &main.Collector{
		Output:   &stdout,
		RepoName: "testrepo",
		Git: &mock.GitRunner{
			// No merge commits - triggers fallback to commit-level
			MergeCommitsFn: func(_ context.Context, _ string, _ int) ([]string, error) {
				return nil, nil
			},
			LogFn: func(_ context.Context, _ string, _ int) ([]string, error) {
				return []string{"commit1", "commit2"}, nil
			},
			ShowFn: func(_ context.Context, _ string, hash string) (string, error) {
				switch hash {
				case "commit1":
					return diff1, nil
				case "commit2":
					return diff2, nil
				}
				return "", errors.New("unknown hash")
			},
			MessageFn: func(_ context.Context, _ string, hash string) (string, error) {
				return "Commit message for " + hash, nil
			},
		},
	}

	err := collector.Run(context.Background())
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	require.Len(t, lines, 2)
	assert.Contains(t, lines[0], `"hash":"commit1"`)
	assert.Contains(t, lines[1], `"hash":"commit2"`)
}

func TestCollector_Run_IncludesFilePaths(t *testing.T) {
	t.Parallel()

	diffOutput := `diff --git a/src/auth/login.go b/src/auth/login.go
new file mode 100644
--- /dev/null
+++ b/src/auth/login.go
@@ -0,0 +1,3 @@
+package auth
+
+func Login() {}
`

	var stdout bytes.Buffer
	collector := &main.Collector{
		Output:   &stdout,
		RepoName: "testrepo",
		Git: &mock.GitRunner{
			// No merge commits - triggers fallback to commit-level
			MergeCommitsFn: func(_ context.Context, _ string, _ int) ([]string, error) {
				return nil, nil
			},
			LogFn: func(_ context.Context, _ string, _ int) ([]string, error) {
				return []string{"abc"}, nil
			},
			ShowFn: func(_ context.Context, _ string, _ string) (string, error) {
				return diffOutput, nil
			},
			MessageFn: func(_ context.Context, _ string, _ string) (string, error) {
				return "Add login", nil
			},
		},
	}

	err := collector.Run(context.Background())
	require.NoError(t, err)

	output := stdout.String()
	// Output should include file path in diff structure
	assert.Contains(t, output, `"NewPath":"src/auth/login.go"`)
}

func TestCollector_Run_GitLogError(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	collector := &main.Collector{
		Output:   &stdout,
		RepoName: "testrepo",
		Git: &mock.GitRunner{
			// No merge commits - triggers fallback to commit-level
			MergeCommitsFn: func(_ context.Context, _ string, _ int) ([]string, error) {
				return nil, nil
			},
			LogFn: func(_ context.Context, _ string, _ int) ([]string, error) {
				return nil, errors.New("not a git repository")
			},
			ShowFn: func(_ context.Context, _ string, _ string) (string, error) {
				return "", nil
			},
			MessageFn: func(_ context.Context, _ string, _ string) (string, error) {
				return "", nil
			},
		},
	}

	err := collector.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a git repository")
}

func TestCollector_Run_GitShowError(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	collector := &main.Collector{
		Output:   &stdout,
		RepoName: "testrepo",
		Git: &mock.GitRunner{
			// No merge commits - triggers fallback to commit-level
			MergeCommitsFn: func(_ context.Context, _ string, _ int) ([]string, error) {
				return nil, nil
			},
			LogFn: func(_ context.Context, _ string, _ int) ([]string, error) {
				return []string{"abc123"}, nil
			},
			ShowFn: func(_ context.Context, _ string, _ string) (string, error) {
				return "", errors.New("commit not found")
			},
			MessageFn: func(_ context.Context, _ string, _ string) (string, error) {
				return "", nil
			},
		},
	}

	err := collector.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "commit not found")
}

func TestCollector_Run_SkipsCommitsWithoutFiles(t *testing.T) {
	t.Parallel()

	// Some commits have no diff (e.g., empty commits)
	emptyDiff := ""
	realDiff := `diff --git a/a.go b/a.go
new file mode 100644
--- /dev/null
+++ b/a.go
@@ -0,0 +1 @@
+package a
`

	var stdout bytes.Buffer
	collector := &main.Collector{
		Output:   &stdout,
		RepoName: "testrepo",
		Git: &mock.GitRunner{
			// No merge commits - triggers fallback to commit-level
			MergeCommitsFn: func(_ context.Context, _ string, _ int) ([]string, error) {
				return nil, nil
			},
			LogFn: func(_ context.Context, _ string, _ int) ([]string, error) {
				return []string{"empty-commit", "real-commit"}, nil
			},
			ShowFn: func(_ context.Context, _ string, hash string) (string, error) {
				if hash == "empty-commit" {
					return emptyDiff, nil
				}
				return realDiff, nil
			},
			MessageFn: func(_ context.Context, _ string, hash string) (string, error) {
				return "Message for " + hash, nil
			},
		},
	}

	err := collector.Run(context.Background())
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	// Should only have 1 line (real commit), empty commit skipped
	require.Len(t, lines, 1)
	assert.Contains(t, lines[0], `"hash":"real-commit"`)
}

func TestCollector_Run_GitMessageError(t *testing.T) {
	t.Parallel()

	diffOutput := `diff --git a/a.go b/a.go
new file mode 100644
--- /dev/null
+++ b/a.go
@@ -0,0 +1 @@
+package a
`

	var stdout bytes.Buffer
	collector := &main.Collector{
		Output:   &stdout,
		RepoName: "testrepo",
		Git: &mock.GitRunner{
			// No merge commits - triggers fallback to commit-level
			MergeCommitsFn: func(_ context.Context, _ string, _ int) ([]string, error) {
				return nil, nil
			},
			LogFn: func(_ context.Context, _ string, _ int) ([]string, error) {
				return []string{"abc123"}, nil
			},
			ShowFn: func(_ context.Context, _ string, _ string) (string, error) {
				return diffOutput, nil
			},
			MessageFn: func(_ context.Context, _ string, _ string) (string, error) {
				return "", errors.New("failed to get commit message")
			},
		},
	}

	err := collector.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get commit message")
}

func TestClassifyRunner_Run_ClassifiesAllCases(t *testing.T) {
	t.Parallel()

	// Create test cases (as if read from JSONL)
	testCases := []diffview.EvalCase{
		{
			Input: diffview.ClassificationInput{
				Repo: "testrepo", Commits: []diffview.CommitBrief{{Hash: "abc123", Message: "Fix bug"}},
				Diff: diffview.Diff{Files: []diffview.FileDiff{{NewPath: "a.go"}}},
			},
			Story: nil,
		},
		{
			Input: diffview.ClassificationInput{
				Repo: "testrepo", Commits: []diffview.CommitBrief{{Hash: "def456", Message: "Add feature"}},
				Diff: diffview.Diff{Files: []diffview.FileDiff{{NewPath: "b.go"}}},
			},
			Story: nil,
		},
	}

	var classifyCalls int
	var stdout bytes.Buffer
	classifier := &main.ClassifyRunner{
		Output: &stdout,
		Cases:  testCases,
		Classifier: &mock.StoryClassifier{
			ClassifyFn: func(_ context.Context, input diffview.ClassificationInput) (*diffview.StoryClassification, error) {
				classifyCalls++
				return &diffview.StoryClassification{
					ChangeType: "bugfix",
					Summary:    "Fixed a bug in " + input.FirstCommitHash(),
				}, nil
			},
		},
	}

	err := classifier.Run(context.Background())
	require.NoError(t, err)

	// Should have called classify for each case
	assert.Equal(t, 2, classifyCalls)

	// Output should be JSONL with classified stories
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	require.Len(t, lines, 2)
	assert.Contains(t, lines[0], `"hash":"abc123"`)
	assert.Contains(t, lines[0], `"change_type":"bugfix"`)
	assert.Contains(t, lines[1], `"hash":"def456"`)
	assert.Contains(t, lines[1], `"change_type":"bugfix"`)
}

func TestClassifyRunner_Run_ClassifierError(t *testing.T) {
	t.Parallel()

	testCases := []diffview.EvalCase{
		{
			Input: diffview.ClassificationInput{
				Commits: []diffview.CommitBrief{{Hash: "abc123"}},
				Diff:    diffview.Diff{Files: []diffview.FileDiff{{NewPath: "a.go"}}},
			},
		},
	}

	var stdout bytes.Buffer
	classifier := &main.ClassifyRunner{
		Output: &stdout,
		Cases:  testCases,
		Classifier: &mock.StoryClassifier{
			ClassifyFn: func(_ context.Context, _ diffview.ClassificationInput) (*diffview.StoryClassification, error) {
				return nil, errors.New("API rate limit exceeded")
			},
		},
	}

	err := classifier.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API rate limit exceeded")
}

func TestClassifyRunner_Run_PreservesExistingStories(t *testing.T) {
	t.Parallel()

	existingStory := &diffview.StoryClassification{
		ChangeType: "feature",
		Summary:    "Already classified",
	}
	testCases := []diffview.EvalCase{
		{
			Input: diffview.ClassificationInput{
				Commits: []diffview.CommitBrief{{Hash: "abc123"}},
				Diff:    diffview.Diff{Files: []diffview.FileDiff{{NewPath: "a.go"}}},
			},
			Story: existingStory, // Already has a story
		},
		{
			Input: diffview.ClassificationInput{
				Commits: []diffview.CommitBrief{{Hash: "def456"}},
				Diff:    diffview.Diff{Files: []diffview.FileDiff{{NewPath: "b.go"}}},
			},
			Story: nil, // Needs classification
		},
	}

	var classifyCalls int
	var stdout bytes.Buffer
	classifier := &main.ClassifyRunner{
		Output: &stdout,
		Cases:  testCases,
		Classifier: &mock.StoryClassifier{
			ClassifyFn: func(_ context.Context, _ diffview.ClassificationInput) (*diffview.StoryClassification, error) {
				classifyCalls++
				return &diffview.StoryClassification{
					ChangeType: "bugfix",
					Summary:    "Newly classified",
				}, nil
			},
		},
	}

	err := classifier.Run(context.Background())
	require.NoError(t, err)

	// Should only call classify for the case without a story
	assert.Equal(t, 1, classifyCalls)

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	require.Len(t, lines, 2)
	// First case should preserve original classification
	assert.Contains(t, lines[0], `"summary":"Already classified"`)
	// Second case should have new classification
	assert.Contains(t, lines[1], `"summary":"Newly classified"`)
}

func TestParseBranchFromMergeMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		message string
		want    string
	}{
		{
			name:    "standard GitHub merge",
			message: "Merge pull request #42 from user/feature-branch",
			want:    "feature-branch",
		},
		{
			name:    "multi-line message",
			message: "Merge pull request #42 from user/feature-branch\n\nThis PR adds a new feature.",
			want:    "feature-branch",
		},
		{
			name:    "nested branch path",
			message: "Merge pull request #42 from user/bugfix/auth/login",
			want:    "bugfix/auth/login",
		},
		{
			name:    "non-GitHub merge format",
			message: "Merge branch 'feature' into main",
			want:    "",
		},
		{
			name:    "empty message",
			message: "",
			want:    "",
		},
		{
			name:    "no from clause",
			message: "Merge pull request #42",
			want:    "",
		},
		{
			name:    "no slash in user/branch",
			message: "Merge pull request #42 from just-branch-name",
			want:    "just-branch-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := main.ParseBranchFromMergeMessage(tt.message)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCollector_Run_FallsBackToCommitLevelWithNoMergeCommits(t *testing.T) {
	t.Parallel()

	diffOutput := `diff --git a/fix.go b/fix.go
new file mode 100644
--- /dev/null
+++ b/fix.go
@@ -0,0 +1,3 @@
+package main
+
+func fix() {}
`

	var stdout bytes.Buffer
	collector := &main.Collector{
		Output:   &stdout,
		RepoName: "testrepo",
		Git: &mock.GitRunner{
			// No merge commits - triggers fallback
			MergeCommitsFn: func(_ context.Context, _ string, _ int) ([]string, error) {
				return nil, nil // Empty slice means no merge commits
			},
			LogFn: func(_ context.Context, _ string, _ int) ([]string, error) {
				return []string{"abc123"}, nil
			},
			ShowFn: func(_ context.Context, _ string, _ string) (string, error) {
				return diffOutput, nil
			},
			MessageFn: func(_ context.Context, _ string, _ string) (string, error) {
				return "Fix bug", nil
			},
			// PR-level methods should not be called
			CommitsInRangeFn: func(_ context.Context, _ string, _, _ string) ([]diffview.CommitBrief, error) {
				t.Error("CommitsInRange should not be called in fallback mode")
				return nil, nil
			},
			DiffRangeFn: func(_ context.Context, _ string, _, _ string) (string, error) {
				t.Error("DiffRange should not be called in fallback mode")
				return "", nil
			},
		},
	}

	err := collector.Run(context.Background())
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	require.Len(t, lines, 1)

	output := lines[0]
	// Should have single commit in the commits array
	assert.Contains(t, output, `"hash":"abc123"`)
	assert.Contains(t, output, `"message":"Fix bug"`)
	// Branch should be empty in fallback mode
	assert.Contains(t, output, `"branch":""`)
}

func TestCollector_Run_ExtractsPRLevelFromMergeCommits(t *testing.T) {
	t.Parallel()

	// PR diff showing combined changes from the feature branch
	prDiff := `diff --git a/feature.go b/feature.go
new file mode 100644
--- /dev/null
+++ b/feature.go
@@ -0,0 +1,5 @@
+package main
+
+func newFeature() {
+	// implementation
+}
`

	var stdout bytes.Buffer
	collector := &main.Collector{
		Output:   &stdout,
		RepoName: "testrepo",
		Git: &mock.GitRunner{
			// PR-level methods
			MergeCommitsFn: func(_ context.Context, _ string, _ int) ([]string, error) {
				// Return one merge commit
				return []string{"merge123"}, nil
			},
			CommitsInRangeFn: func(_ context.Context, _ string, base, head string) ([]diffview.CommitBrief, error) {
				// Commits in the PR (base^1..base^2 where base is merge commit)
				if base == "merge123^1" && head == "merge123^2" {
					return []diffview.CommitBrief{
						{Hash: "feat1", Message: "Add new feature"},
						{Hash: "feat2", Message: "Fix tests"},
					}, nil
				}
				return nil, errors.New("unexpected range")
			},
			DiffRangeFn: func(_ context.Context, _ string, base, head string) (string, error) {
				if base == "merge123^1" && head == "merge123^2" {
					return prDiff, nil
				}
				return "", errors.New("unexpected range")
			},
			MessageFn: func(_ context.Context, _ string, hash string) (string, error) {
				if hash == "merge123" {
					return "Merge pull request #42 from user/feature-branch", nil
				}
				return "", errors.New("unknown hash")
			},
			// Deprecated methods should not be called
			LogFn: func(_ context.Context, _ string, _ int) ([]string, error) {
				t.Error("Log should not be called when merge commits exist")
				return nil, nil
			},
			ShowFn: func(_ context.Context, _ string, _ string) (string, error) {
				t.Error("Show should not be called for PR-level extraction")
				return "", nil
			},
		},
	}

	err := collector.Run(context.Background())
	require.NoError(t, err)

	// Should output one PR-level case
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	require.Len(t, lines, 1)

	output := lines[0]
	// Should have branch name extracted from merge message
	assert.Contains(t, output, `"branch":"feature-branch"`)
	// Should have all commits from the PR
	assert.Contains(t, output, `"hash":"feat1"`)
	assert.Contains(t, output, `"hash":"feat2"`)
	assert.Contains(t, output, `"message":"Add new feature"`)
	// Should have the combined diff
	assert.Contains(t, output, `"NewPath":"feature.go"`)
}
