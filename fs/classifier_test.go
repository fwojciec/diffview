package fs_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/fwojciec/diffview"
	"github.com/fwojciec/diffview/fs"
	"github.com/fwojciec/diffview/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifier_CacheMiss_DelegatesToInner(t *testing.T) {
	t.Parallel()

	cacheDir := t.TempDir()
	innerCalled := false
	expected := &diffview.StoryClassification{
		ChangeType: "feature",
		Summary:    "Test summary",
	}

	inner := &mock.StoryClassifier{
		ClassifyFn: func(ctx context.Context, input diffview.ClassificationInput) (*diffview.StoryClassification, error) {
			innerCalled = true
			return expected, nil
		},
	}

	classifier := fs.NewClassifier(inner, cacheDir)

	input := diffview.ClassificationInput{
		Diff: diffview.Diff{
			Files: []diffview.FileDiff{{NewPath: "test.go"}},
		},
	}

	result, err := classifier.Classify(context.Background(), input)

	require.NoError(t, err)
	assert.True(t, innerCalled, "inner classifier should be called on cache miss")
	assert.Equal(t, expected, result)
}

func TestClassifier_CacheHit_ReturnsWithoutCallingInner(t *testing.T) {
	t.Parallel()

	cacheDir := t.TempDir()
	callCount := 0
	expected := &diffview.StoryClassification{
		ChangeType: "bugfix",
		Summary:    "Fix the bug",
	}

	inner := &mock.StoryClassifier{
		ClassifyFn: func(ctx context.Context, input diffview.ClassificationInput) (*diffview.StoryClassification, error) {
			callCount++
			return expected, nil
		},
	}

	classifier := fs.NewClassifier(inner, cacheDir)

	input := diffview.ClassificationInput{
		Diff: diffview.Diff{
			Files: []diffview.FileDiff{{NewPath: "bugfix.go"}},
		},
	}

	// First call - should call inner and cache
	result1, err := classifier.Classify(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, 1, callCount, "first call should invoke inner")
	assert.Equal(t, expected, result1)

	// Second call with same input - should return cached, not call inner
	result2, err := classifier.Classify(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, 1, callCount, "second call should NOT invoke inner (cache hit)")
	assert.Equal(t, expected, result2)
}

func TestClassifier_DifferentInput_CallsInnerAgain(t *testing.T) {
	t.Parallel()

	cacheDir := t.TempDir()
	callCount := 0

	inner := &mock.StoryClassifier{
		ClassifyFn: func(ctx context.Context, input diffview.ClassificationInput) (*diffview.StoryClassification, error) {
			callCount++
			return &diffview.StoryClassification{
				ChangeType: "feature",
				Summary:    "Call " + string(rune('0'+callCount)),
			}, nil
		},
	}

	classifier := fs.NewClassifier(inner, cacheDir)

	input1 := diffview.ClassificationInput{
		Diff: diffview.Diff{
			Files: []diffview.FileDiff{{NewPath: "file1.go"}},
		},
	}
	input2 := diffview.ClassificationInput{
		Diff: diffview.Diff{
			Files: []diffview.FileDiff{{NewPath: "file2.go"}},
		},
	}

	// First input
	_, err := classifier.Classify(context.Background(), input1)
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)

	// Different input - should call inner again
	_, err = classifier.Classify(context.Background(), input2)
	require.NoError(t, err)
	assert.Equal(t, 2, callCount, "different input should trigger new inner call")

	// First input again - should be cached
	_, err = classifier.Classify(context.Background(), input1)
	require.NoError(t, err)
	assert.Equal(t, 2, callCount, "first input should still be cached")
}

func TestDefaultCacheDir_UsesXDGIfSet(t *testing.T) {
	// Can't use t.Parallel with t.Setenv
	t.Setenv("XDG_CACHE_HOME", "/custom/cache")

	dir := fs.DefaultCacheDir()

	assert.Equal(t, "/custom/cache/diffstory", dir)
}

func TestDefaultCacheDir_FallsBackToHomeCache(t *testing.T) {
	// Can't use t.Parallel with t.Setenv
	t.Setenv("XDG_CACHE_HOME", "")

	dir := fs.DefaultCacheDir()

	home, _ := os.UserHomeDir()
	assert.Equal(t, filepath.Join(home, ".cache", "diffstory"), dir)
}

func TestClassifier_CorruptedCache_TreatedAsMiss(t *testing.T) {
	t.Parallel()

	cacheDir := t.TempDir()
	callCount := 0
	expected := &diffview.StoryClassification{
		ChangeType: "refactor",
		Summary:    "Clean up",
	}

	inner := &mock.StoryClassifier{
		ClassifyFn: func(ctx context.Context, input diffview.ClassificationInput) (*diffview.StoryClassification, error) {
			callCount++
			return expected, nil
		},
	}

	classifier := fs.NewClassifier(inner, cacheDir)

	input := diffview.ClassificationInput{
		Diff: diffview.Diff{
			Files: []diffview.FileDiff{{NewPath: "refactor.go"}},
		},
	}

	// First call - populates cache
	_, err := classifier.Classify(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)

	// Corrupt the cache file
	files, err := os.ReadDir(cacheDir)
	require.NoError(t, err)
	require.Len(t, files, 1)
	cachePath := filepath.Join(cacheDir, files[0].Name())
	err = os.WriteFile(cachePath, []byte("not valid json"), 0644)
	require.NoError(t, err)

	// Next call should treat corrupted file as miss
	result, err := classifier.Classify(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, 2, callCount, "corrupted cache should trigger new inner call")
	assert.Equal(t, expected, result)
}
