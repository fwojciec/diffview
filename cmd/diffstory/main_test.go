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
		Output: &stdout,
		Git: &mock.GitRunner{
			LogFn: func(_ context.Context, _ string, _ int) ([]string, error) {
				return []string{"abc1234"}, nil
			},
			ShowFn: func(_ context.Context, _ string, hash string) (string, error) {
				if hash == "abc1234" {
					return diffOutput, nil
				}
				return "", errors.New("unknown hash")
			},
		},
	}

	err := collector.Run(context.Background())
	require.NoError(t, err)

	// Output should be JSONL with one line per commit
	output := stdout.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	require.Len(t, lines, 1)

	// Line should contain commit hash and hunks
	assert.Contains(t, lines[0], `"commit":"abc1234"`)
	assert.Contains(t, lines[0], `"hunks"`)
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
		Output: &stdout,
		Git: &mock.GitRunner{
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
		},
	}

	err := collector.Run(context.Background())
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	require.Len(t, lines, 2)
	assert.Contains(t, lines[0], `"commit":"commit1"`)
	assert.Contains(t, lines[1], `"commit":"commit2"`)
}

func TestCollector_Run_AnnotatesHunksWithFilePath(t *testing.T) {
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
		Output: &stdout,
		Git: &mock.GitRunner{
			LogFn: func(_ context.Context, _ string, _ int) ([]string, error) {
				return []string{"abc"}, nil
			},
			ShowFn: func(_ context.Context, _ string, _ string) (string, error) {
				return diffOutput, nil
			},
		},
	}

	err := collector.Run(context.Background())
	require.NoError(t, err)

	output := stdout.String()
	// Hunk ID should include file path
	assert.Contains(t, output, `"ID":"src/auth/login.go:h0"`)
}

func TestCollector_Run_GitLogError(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	collector := &main.Collector{
		Output: &stdout,
		Git: &mock.GitRunner{
			LogFn: func(_ context.Context, _ string, _ int) ([]string, error) {
				return nil, errors.New("not a git repository")
			},
			ShowFn: func(_ context.Context, _ string, _ string) (string, error) {
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
		Output: &stdout,
		Git: &mock.GitRunner{
			LogFn: func(_ context.Context, _ string, _ int) ([]string, error) {
				return []string{"abc123"}, nil
			},
			ShowFn: func(_ context.Context, _ string, _ string) (string, error) {
				return "", errors.New("commit not found")
			},
		},
	}

	err := collector.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "commit not found")
}

func TestCollector_Run_SkipsCommitsWithoutHunks(t *testing.T) {
	t.Parallel()

	// Merge commits often have no diff
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
		Output: &stdout,
		Git: &mock.GitRunner{
			LogFn: func(_ context.Context, _ string, _ int) ([]string, error) {
				return []string{"merge-commit", "real-commit"}, nil
			},
			ShowFn: func(_ context.Context, _ string, hash string) (string, error) {
				if hash == "merge-commit" {
					return emptyDiff, nil
				}
				return realDiff, nil
			},
		},
	}

	err := collector.Run(context.Background())
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	// Should only have 1 line (real commit), merge commit skipped
	require.Len(t, lines, 1)
	assert.Contains(t, lines[0], `"commit":"real-commit"`)
}
