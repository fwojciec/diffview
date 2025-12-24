package diffview_test

import (
	"testing"

	"github.com/fwojciec/diffview"
	"github.com/stretchr/testify/assert"
)

func TestColorPair(t *testing.T) {
	t.Parallel()

	t.Run("stores foreground and background colors", func(t *testing.T) {
		t.Parallel()

		cp := diffview.ColorPair{
			Foreground: "#00ff00",
			Background: "#000000",
		}

		assert.Equal(t, "#00ff00", cp.Foreground)
		assert.Equal(t, "#000000", cp.Background)
	})
}

func TestStyles(t *testing.T) {
	t.Parallel()

	t.Run("contains styles for all diff line types", func(t *testing.T) {
		t.Parallel()

		styles := diffview.Styles{
			Added:   diffview.ColorPair{Foreground: "#00ff00"},
			Deleted: diffview.ColorPair{Foreground: "#ff0000"},
			Context: diffview.ColorPair{Foreground: "#888888"},
		}

		assert.Equal(t, "#00ff00", styles.Added.Foreground)
		assert.Equal(t, "#ff0000", styles.Deleted.Foreground)
		assert.Equal(t, "#888888", styles.Context.Foreground)
	})

	t.Run("contains styles for hunk headers", func(t *testing.T) {
		t.Parallel()

		styles := diffview.Styles{
			HunkHeader: diffview.ColorPair{Foreground: "#00ffff"},
		}

		assert.Equal(t, "#00ffff", styles.HunkHeader.Foreground)
	})

	t.Run("contains styles for file headers", func(t *testing.T) {
		t.Parallel()

		styles := diffview.Styles{
			FileHeader: diffview.ColorPair{Foreground: "#ffffff", Background: "#333333"},
		}

		assert.Equal(t, "#ffffff", styles.FileHeader.Foreground)
		assert.Equal(t, "#333333", styles.FileHeader.Background)
	})

	t.Run("contains highlight styles for word-level changes", func(t *testing.T) {
		t.Parallel()

		styles := diffview.Styles{
			AddedHighlight:   diffview.ColorPair{Foreground: "#00ff00", Background: "#003300"},
			DeletedHighlight: diffview.ColorPair{Foreground: "#ff0000", Background: "#330000"},
		}

		assert.Equal(t, "#00ff00", styles.AddedHighlight.Foreground)
		assert.Equal(t, "#003300", styles.AddedHighlight.Background)
		assert.Equal(t, "#ff0000", styles.DeletedHighlight.Foreground)
		assert.Equal(t, "#330000", styles.DeletedHighlight.Background)
	})
}

func TestTheme(t *testing.T) {
	t.Parallel()

	t.Run("returns styles", func(t *testing.T) {
		t.Parallel()

		theme := &mockTheme{
			styles: diffview.Styles{
				Added: diffview.ColorPair{Foreground: "#00ff00"},
			},
		}

		result := theme.Styles()
		assert.Equal(t, "#00ff00", result.Added.Foreground)
	})
}

// mockTheme implements diffview.Theme for testing.
type mockTheme struct {
	styles diffview.Styles
}

func (m *mockTheme) Styles() diffview.Styles {
	return m.styles
}

// Verify mockTheme implements Theme interface
var _ diffview.Theme = (*mockTheme)(nil)
