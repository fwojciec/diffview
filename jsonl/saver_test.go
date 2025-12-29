package jsonl_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fwojciec/diffview"
	"github.com/fwojciec/diffview/jsonl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaver_Save(t *testing.T) {
	t.Parallel()

	t.Run("appends case to new file", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "output.jsonl")

		saver := jsonl.NewSaver()
		evalCase := diffview.EvalCase{
			Input: diffview.ClassificationInput{
				Repo:   "test/repo",
				Branch: "main",
				Commits: []diffview.CommitBrief{
					{Hash: "abc123", Message: "Test commit"},
				},
			},
			Story: &diffview.StoryClassification{
				ChangeType: "feature",
				Summary:    "Added feature",
			},
		}

		err := saver.Save(path, evalCase)

		require.NoError(t, err)

		// Verify file contains the case
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Contains(t, string(content), `"repo":"test/repo"`)
		assert.Contains(t, string(content), `"hash":"abc123"`)
		assert.Contains(t, string(content), `"change_type":"feature"`)
	})

	t.Run("appends to existing file", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "existing.jsonl")

		// Create file with existing content
		existing := `{"input":{"repo":"old/repo","commits":[]},"story":null}` + "\n"
		require.NoError(t, os.WriteFile(path, []byte(existing), 0o644))

		saver := jsonl.NewSaver()
		evalCase := diffview.EvalCase{
			Input: diffview.ClassificationInput{
				Repo: "new/repo",
			},
		}

		err := saver.Save(path, evalCase)

		require.NoError(t, err)

		// Verify both lines exist
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		lines := splitLines(string(content))
		assert.Len(t, lines, 2)
		assert.Contains(t, lines[0], `"repo":"old/repo"`)
		assert.Contains(t, lines[1], `"repo":"new/repo"`)
	})

	t.Run("creates parent directories", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "nested", "deep", "output.jsonl")

		saver := jsonl.NewSaver()
		evalCase := diffview.EvalCase{
			Input: diffview.ClassificationInput{Repo: "test"},
		}

		err := saver.Save(path, evalCase)

		require.NoError(t, err)
		assert.FileExists(t, path)
	})
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			if i > start {
				lines = append(lines, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
