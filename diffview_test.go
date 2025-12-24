package diffview_test

import (
	"testing"

	"github.com/fwojciec/diffview"
	"github.com/stretchr/testify/assert"
)

func TestFileDiff_Stats(t *testing.T) {
	t.Parallel()

	t.Run("counts added and deleted lines", func(t *testing.T) {
		t.Parallel()

		file := diffview.FileDiff{
			Hunks: []diffview.Hunk{
				{
					Lines: []diffview.Line{
						{Type: diffview.LineContext},
						{Type: diffview.LineDeleted},
						{Type: diffview.LineAdded},
						{Type: diffview.LineAdded},
						{Type: diffview.LineContext},
					},
				},
			},
		}

		added, deleted := file.Stats()

		assert.Equal(t, 2, added)
		assert.Equal(t, 1, deleted)
	})

	t.Run("counts across multiple hunks", func(t *testing.T) {
		t.Parallel()

		file := diffview.FileDiff{
			Hunks: []diffview.Hunk{
				{
					Lines: []diffview.Line{
						{Type: diffview.LineDeleted},
						{Type: diffview.LineAdded},
					},
				},
				{
					Lines: []diffview.Line{
						{Type: diffview.LineDeleted},
						{Type: diffview.LineDeleted},
						{Type: diffview.LineAdded},
					},
				},
			},
		}

		added, deleted := file.Stats()

		assert.Equal(t, 2, added)
		assert.Equal(t, 3, deleted)
	})

	t.Run("returns zero for empty hunks", func(t *testing.T) {
		t.Parallel()

		file := diffview.FileDiff{}

		added, deleted := file.Stats()

		assert.Equal(t, 0, added)
		assert.Equal(t, 0, deleted)
	})

	t.Run("returns zero for context-only hunks", func(t *testing.T) {
		t.Parallel()

		file := diffview.FileDiff{
			Hunks: []diffview.Hunk{
				{
					Lines: []diffview.Line{
						{Type: diffview.LineContext},
						{Type: diffview.LineContext},
					},
				},
			},
		}

		added, deleted := file.Stats()

		assert.Equal(t, 0, added)
		assert.Equal(t, 0, deleted)
	})
}
