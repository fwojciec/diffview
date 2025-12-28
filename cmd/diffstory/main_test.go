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
