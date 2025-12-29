package main_test

import (
	"errors"
	"testing"

	"github.com/fwojciec/diffview"
	main "github.com/fwojciec/diffview/cmd/diffstory"
	"github.com/fwojciec/diffview/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplayApp_Run_LoadsCase(t *testing.T) {
	t.Parallel()

	testCase := diffview.EvalCase{
		Input: diffview.ClassificationInput{
			Repo:   "test-repo",
			Branch: "feature-branch",
			Diff: diffview.Diff{
				Files: []diffview.FileDiff{
					{NewPath: "feature.go"},
				},
			},
		},
		Story: &diffview.StoryClassification{
			ChangeType: "feature",
			Summary:    "Added a new feature",
		},
	}

	app := &main.ReplayApp{
		Loader: &mock.EvalCaseLoader{
			LoadFn: func(path string) ([]diffview.EvalCase, error) {
				assert.Equal(t, "test.jsonl", path)
				return []diffview.EvalCase{testCase}, nil
			},
		},
		FilePath: "test.jsonl",
		Index:    0,
	}

	diff, story, err := app.Run()
	require.NoError(t, err)
	require.NotNil(t, diff)
	require.NotNil(t, story)
	assert.Len(t, diff.Files, 1)
	assert.Equal(t, "feature.go", diff.Files[0].NewPath)
	assert.Equal(t, "feature", story.ChangeType)
}

func TestReplayApp_Run_LoaderError(t *testing.T) {
	t.Parallel()

	app := &main.ReplayApp{
		Loader: &mock.EvalCaseLoader{
			LoadFn: func(path string) ([]diffview.EvalCase, error) {
				return nil, errors.New("file not found")
			},
		},
		FilePath: "missing.jsonl",
		Index:    0,
	}

	_, _, err := app.Run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file not found")
}

func TestReplayApp_Run_IndexOutOfBounds(t *testing.T) {
	t.Parallel()

	app := &main.ReplayApp{
		Loader: &mock.EvalCaseLoader{
			LoadFn: func(path string) ([]diffview.EvalCase, error) {
				return []diffview.EvalCase{
					{Input: diffview.ClassificationInput{Repo: "test"}},
				}, nil
			},
		},
		FilePath: "test.jsonl",
		Index:    5, // Out of bounds
	}

	_, _, err := app.Run()
	require.Error(t, err)
	assert.Equal(t, main.ErrIndexOutOfBounds, err)
}

func TestReplayApp_Run_NegativeIndex(t *testing.T) {
	t.Parallel()

	app := &main.ReplayApp{
		Loader: &mock.EvalCaseLoader{
			LoadFn: func(path string) ([]diffview.EvalCase, error) {
				return []diffview.EvalCase{
					{Input: diffview.ClassificationInput{Repo: "test"}},
				}, nil
			},
		},
		FilePath: "test.jsonl",
		Index:    -1, // Negative index
	}

	_, _, err := app.Run()
	require.Error(t, err)
	assert.Equal(t, main.ErrIndexOutOfBounds, err)
}

func TestReplayApp_Run_EmptyFile(t *testing.T) {
	t.Parallel()

	app := &main.ReplayApp{
		Loader: &mock.EvalCaseLoader{
			LoadFn: func(path string) ([]diffview.EvalCase, error) {
				return []diffview.EvalCase{}, nil // Empty file
			},
		},
		FilePath: "empty.jsonl",
		Index:    0,
	}

	_, _, err := app.Run()
	require.Error(t, err)
	assert.Equal(t, main.ErrIndexOutOfBounds, err)
}

func TestReplayApp_Run_CaseWithNilStory(t *testing.T) {
	t.Parallel()

	// Some cases might not have a story classification yet
	testCase := diffview.EvalCase{
		Input: diffview.ClassificationInput{
			Repo: "test-repo",
			Diff: diffview.Diff{
				Files: []diffview.FileDiff{
					{NewPath: "feature.go"},
				},
			},
		},
		Story: nil, // No classification
	}

	app := &main.ReplayApp{
		Loader: &mock.EvalCaseLoader{
			LoadFn: func(path string) ([]diffview.EvalCase, error) {
				return []diffview.EvalCase{testCase}, nil
			},
		},
		FilePath: "test.jsonl",
		Index:    0,
	}

	diff, story, err := app.Run()
	require.NoError(t, err)
	require.NotNil(t, diff)
	assert.Nil(t, story) // Story can be nil
	assert.Len(t, diff.Files, 1)
}
