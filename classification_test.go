package diffview_test

import (
	"encoding/json"
	"testing"

	diffview "github.com/fwojciec/diffstory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommitBrief_JSONOmitsEmptyDiff(t *testing.T) {
	t.Parallel()

	t.Run("omits diff when nil", func(t *testing.T) {
		t.Parallel()

		brief := diffview.CommitBrief{
			Hash:    "abc123",
			Message: "test commit",
			Diff:    nil,
		}

		data, err := json.Marshal(brief)
		require.NoError(t, err)

		// Should not contain "diff" key at all
		assert.NotContains(t, string(data), "diff")
	})

	t.Run("includes diff when present", func(t *testing.T) {
		t.Parallel()

		brief := diffview.CommitBrief{
			Hash:    "abc123",
			Message: "test commit",
			Diff:    &diffview.Diff{Files: []diffview.FileDiff{{NewPath: "test.go"}}},
		}

		data, err := json.Marshal(brief)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"diff"`)
	})
}

func TestStoryClassification_JSONOmitsEmptyEvolution(t *testing.T) {
	t.Parallel()

	t.Run("omits evolution when empty", func(t *testing.T) {
		t.Parallel()

		classification := diffview.StoryClassification{
			ChangeType: "feature",
			Narrative:  "core-periphery",
			Summary:    "Test summary",
			Sections:   []diffview.Section{},
			Evolution:  "",
		}

		data, err := json.Marshal(classification)
		require.NoError(t, err)

		assert.NotContains(t, string(data), "evolution")
	})

	t.Run("includes evolution when present", func(t *testing.T) {
		t.Parallel()

		classification := diffview.StoryClassification{
			ChangeType: "feature",
			Narrative:  "core-periphery",
			Summary:    "Test summary",
			Sections:   []diffview.Section{},
			Evolution:  "Changes evolved from initial implementation to refined version",
		}

		data, err := json.Marshal(classification)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"evolution"`)
		assert.Contains(t, string(data), "Changes evolved")
	})
}
