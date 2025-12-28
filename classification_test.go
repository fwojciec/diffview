package diffview_test

import (
	"testing"

	"github.com/fwojciec/diffview"
	"github.com/stretchr/testify/assert"
)

func TestOrderSections(t *testing.T) {
	t.Parallel()

	t.Run("cause-effect orders problem before fix before test before supporting", func(t *testing.T) {
		t.Parallel()

		classification := &diffview.StoryClassification{
			Narrative: "cause-effect",
			Sections: []diffview.Section{
				{Role: "supporting", Title: "Update imports"},
				{Role: "test", Title: "Add regression test"},
				{Role: "fix", Title: "Fix the bug"},
				{Role: "problem", Title: "Identify the issue"},
			},
		}

		classification.OrderSections()

		roles := extractRoles(classification.Sections)
		assert.Equal(t, []string{"problem", "fix", "test", "supporting"}, roles)
	})

	t.Run("core-periphery orders core before supporting before cleanup", func(t *testing.T) {
		t.Parallel()

		classification := &diffview.StoryClassification{
			Narrative: "core-periphery",
			Sections: []diffview.Section{
				{Role: "cleanup", Title: "Remove old code"},
				{Role: "supporting", Title: "Update callers"},
				{Role: "core", Title: "Main change"},
			},
		}

		classification.OrderSections()

		roles := extractRoles(classification.Sections)
		assert.Equal(t, []string{"core", "supporting", "cleanup"}, roles)
	})

	t.Run("before-after orders cleanup before core before test before supporting", func(t *testing.T) {
		t.Parallel()

		// In before-after: cleanup = "before" (old pattern removal), core = "after" (new pattern)
		// supporting goes at end for incidental changes
		classification := &diffview.StoryClassification{
			Narrative: "before-after",
			Sections: []diffview.Section{
				{Role: "supporting", Title: "Update imports"},
				{Role: "test", Title: "Add tests"},
				{Role: "core", Title: "New pattern"},
				{Role: "cleanup", Title: "Remove old pattern"},
			},
		}

		classification.OrderSections()

		roles := extractRoles(classification.Sections)
		assert.Equal(t, []string{"cleanup", "core", "test", "supporting"}, roles)
	})

	t.Run("rule-instances orders pattern before core before supporting", func(t *testing.T) {
		t.Parallel()

		classification := &diffview.StoryClassification{
			Narrative: "rule-instances",
			Sections: []diffview.Section{
				{Role: "supporting", Title: "Helpers"},
				{Role: "core", Title: "Apply pattern"},
				{Role: "pattern", Title: "Define pattern"},
			},
		}

		classification.OrderSections()

		roles := extractRoles(classification.Sections)
		assert.Equal(t, []string{"pattern", "core", "supporting"}, roles)
	})

	t.Run("entry-implementation orders interface before core before test before supporting", func(t *testing.T) {
		t.Parallel()

		classification := &diffview.StoryClassification{
			Narrative: "entry-implementation",
			Sections: []diffview.Section{
				{Role: "supporting", Title: "Helpers"},
				{Role: "test", Title: "Tests"},
				{Role: "core", Title: "Implementation"},
				{Role: "interface", Title: "API entry"},
			},
		}

		classification.OrderSections()

		roles := extractRoles(classification.Sections)
		assert.Equal(t, []string{"interface", "core", "test", "supporting"}, roles)
	})

	t.Run("unknown roles are placed after known roles", func(t *testing.T) {
		t.Parallel()

		classification := &diffview.StoryClassification{
			Narrative: "cause-effect",
			Sections: []diffview.Section{
				{Role: "cleanup", Title: "Not in cause-effect order"},
				{Role: "problem", Title: "The problem"},
				{Role: "interface", Title: "Also not in cause-effect order"},
				{Role: "fix", Title: "The fix"},
			},
		}

		classification.OrderSections()

		roles := extractRoles(classification.Sections)
		// problem, fix come first (in order), then cleanup and interface (preserving relative order)
		assert.Equal(t, []string{"problem", "fix", "cleanup", "interface"}, roles)
	})

	t.Run("unknown narrative preserves original order", func(t *testing.T) {
		t.Parallel()

		classification := &diffview.StoryClassification{
			Narrative: "unknown-narrative",
			Sections: []diffview.Section{
				{Role: "test", Title: "First"},
				{Role: "core", Title: "Second"},
				{Role: "fix", Title: "Third"},
			},
		}

		classification.OrderSections()

		roles := extractRoles(classification.Sections)
		assert.Equal(t, []string{"test", "core", "fix"}, roles)
	})

	t.Run("nil classification is safe", func(t *testing.T) {
		t.Parallel()

		var classification *diffview.StoryClassification

		assert.NotPanics(t, func() {
			classification.OrderSections()
		})
	})

	t.Run("empty sections is safe", func(t *testing.T) {
		t.Parallel()

		classification := &diffview.StoryClassification{
			Narrative: "cause-effect",
			Sections:  []diffview.Section{},
		}

		assert.NotPanics(t, func() {
			classification.OrderSections()
		})
	})

	t.Run("duplicate roles preserve relative order", func(t *testing.T) {
		t.Parallel()

		classification := &diffview.StoryClassification{
			Narrative: "cause-effect",
			Sections: []diffview.Section{
				{Role: "fix", Title: "Fix A"},
				{Role: "problem", Title: "Problem"},
				{Role: "fix", Title: "Fix B"},
			},
		}

		classification.OrderSections()

		// problem comes first, then both fixes in original relative order
		assert.Equal(t, "problem", classification.Sections[0].Role)
		assert.Equal(t, "Fix A", classification.Sections[1].Title)
		assert.Equal(t, "Fix B", classification.Sections[2].Title)
	})
}

func extractRoles(sections []diffview.Section) []string {
	roles := make([]string, len(sections))
	for i, s := range sections {
		roles[i] = s.Role
	}
	return roles
}
