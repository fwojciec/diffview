package jsonl_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fwojciec/diffview"
	"github.com/fwojciec/diffview/jsonl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_Load(t *testing.T) {
	t.Parallel()

	t.Run("loads valid judgments file", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "judgments.jsonl")
		content := `{"commit":"abc123","index":0,"pass":true,"critique":"","judged_at":"2025-01-15T10:30:00Z"}
{"commit":"def456","index":1,"pass":false,"critique":"Missing context","judged_at":"2025-01-15T10:31:00Z"}`
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

		store := jsonl.NewStore()
		judgments, err := store.Load(path)

		require.NoError(t, err)
		assert.Len(t, judgments, 2)
		assert.Equal(t, "abc123", judgments[0].Commit)
		assert.True(t, judgments[0].Pass)
		assert.Equal(t, "def456", judgments[1].Commit)
		assert.False(t, judgments[1].Pass)
		assert.Equal(t, "Missing context", judgments[1].Critique)
	})

	t.Run("returns empty slice for non-existent file", func(t *testing.T) {
		t.Parallel()

		store := jsonl.NewStore()
		judgments, err := store.Load("/nonexistent/path.jsonl")

		require.NoError(t, err)
		assert.Empty(t, judgments)
	})

	t.Run("returns error for malformed JSON", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "bad.jsonl")
		content := `{"commit":"abc123","index":0}
not valid json`
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

		store := jsonl.NewStore()
		_, err := store.Load(path)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "line 2")
	})
}

func TestStore_Save(t *testing.T) {
	t.Parallel()

	t.Run("saves judgments to file", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "judgments.jsonl")

		judgments := []diffview.Judgment{
			{
				Commit:   "abc123",
				Index:    0,
				Pass:     true,
				Critique: "",
				JudgedAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
			},
			{
				Commit:   "def456",
				Index:    1,
				Pass:     false,
				Critique: "Wrong analysis",
				JudgedAt: time.Date(2025, 1, 15, 10, 31, 0, 0, time.UTC),
			},
		}

		store := jsonl.NewStore()
		err := store.Save(path, judgments)

		require.NoError(t, err)

		// Verify by reading back
		loaded, err := store.Load(path)
		require.NoError(t, err)
		assert.Len(t, loaded, 2)
		assert.Equal(t, "abc123", loaded[0].Commit)
		assert.True(t, loaded[0].Pass)
		assert.Equal(t, "def456", loaded[1].Commit)
		assert.False(t, loaded[1].Pass)
		assert.Equal(t, "Wrong analysis", loaded[1].Critique)
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "judgments.jsonl")
		require.NoError(t, os.WriteFile(path, []byte("old content"), 0o644))

		judgments := []diffview.Judgment{
			{Commit: "new123", Index: 0, Pass: true, JudgedAt: time.Now()},
		}

		store := jsonl.NewStore()
		err := store.Save(path, judgments)

		require.NoError(t, err)

		loaded, err := store.Load(path)
		require.NoError(t, err)
		assert.Len(t, loaded, 1)
		assert.Equal(t, "new123", loaded[0].Commit)
	})

	t.Run("creates parent directories", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "subdir", "nested", "judgments.jsonl")

		judgments := []diffview.Judgment{
			{Commit: "abc123", Index: 0, Pass: true, JudgedAt: time.Now()},
		}

		store := jsonl.NewStore()
		err := store.Save(path, judgments)

		require.NoError(t, err)

		loaded, err := store.Load(path)
		require.NoError(t, err)
		assert.Len(t, loaded, 1)
	})

	t.Run("handles empty judgments slice", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "empty.jsonl")

		store := jsonl.NewStore()
		err := store.Save(path, []diffview.Judgment{})

		require.NoError(t, err)

		loaded, err := store.Load(path)
		require.NoError(t, err)
		assert.Empty(t, loaded)
	})
}
