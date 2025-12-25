package lipgloss_test

import (
	"testing"

	"github.com/fwojciec/diffview"
	"github.com/fwojciec/diffview/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestDefaultTheme(t *testing.T) {
	t.Parallel()

	t.Run("implements Theme interface", func(t *testing.T) {
		t.Parallel()

		var _ diffview.Theme = lipgloss.DefaultTheme()
	})

	t.Run("returns styles with added line coloring", func(t *testing.T) {
		t.Parallel()

		theme := lipgloss.DefaultTheme()
		styles := theme.Styles()

		assert.NotEmpty(t, styles.Added.Foreground)
	})

	t.Run("returns styles with deleted line coloring", func(t *testing.T) {
		t.Parallel()

		theme := lipgloss.DefaultTheme()
		styles := theme.Styles()

		assert.NotEmpty(t, styles.Deleted.Foreground)
	})

	t.Run("returns styles with context line coloring", func(t *testing.T) {
		t.Parallel()

		theme := lipgloss.DefaultTheme()
		styles := theme.Styles()

		assert.NotEmpty(t, styles.Context.Foreground)
	})

	t.Run("returns styles with hunk header coloring", func(t *testing.T) {
		t.Parallel()

		theme := lipgloss.DefaultTheme()
		styles := theme.Styles()

		assert.NotEmpty(t, styles.HunkHeader.Foreground)
	})

	t.Run("returns styles with file header coloring", func(t *testing.T) {
		t.Parallel()

		theme := lipgloss.DefaultTheme()
		styles := theme.Styles()

		assert.NotEmpty(t, styles.FileHeader.Foreground)
	})

	t.Run("returns styles with file separator coloring", func(t *testing.T) {
		t.Parallel()

		theme := lipgloss.DefaultTheme()
		styles := theme.Styles()

		assert.NotEmpty(t, styles.FileSeparator.Foreground)
	})

	t.Run("returns same styles as DarkTheme", func(t *testing.T) {
		t.Parallel()

		defaultStyles := lipgloss.DefaultTheme().Styles()
		darkStyles := lipgloss.DarkTheme().Styles()

		assert.Equal(t, darkStyles, defaultStyles)
	})
}

func TestDarkTheme(t *testing.T) {
	t.Parallel()

	t.Run("implements Theme interface", func(t *testing.T) {
		t.Parallel()

		var _ diffview.Theme = lipgloss.DarkTheme()
	})

	t.Run("returns styles optimized for dark backgrounds", func(t *testing.T) {
		t.Parallel()

		theme := lipgloss.DarkTheme()
		styles := theme.Styles()

		// Dark theme should have all required styles
		assert.NotEmpty(t, styles.Added.Foreground)
		assert.NotEmpty(t, styles.Deleted.Foreground)
		assert.NotEmpty(t, styles.Context.Foreground)
		assert.NotEmpty(t, styles.HunkHeader.Foreground)
		assert.NotEmpty(t, styles.FileHeader.Foreground)
		assert.NotEmpty(t, styles.FileSeparator.Foreground)
	})

	t.Run("dims context lines for better change visibility", func(t *testing.T) {
		t.Parallel()

		theme := lipgloss.DarkTheme()
		styles := theme.Styles()

		// Context should use muted gray (#6c7086) so changes pop more
		assert.Equal(t, "#6c7086", styles.Context.Foreground)
	})

	t.Run("returns highlight styles for word-level diff", func(t *testing.T) {
		t.Parallel()

		theme := lipgloss.DarkTheme()
		styles := theme.Styles()

		// Highlight styles should be brighter than base styles
		assert.NotEmpty(t, styles.AddedHighlight.Foreground)
		assert.NotEmpty(t, styles.DeletedHighlight.Foreground)
	})

	t.Run("returns palette with all semantic colors", func(t *testing.T) {
		t.Parallel()

		theme := lipgloss.DarkTheme()
		palette := theme.Palette()

		// Base colors
		assert.NotEmpty(t, palette.Background)
		assert.NotEmpty(t, palette.Foreground)

		// Diff colors
		assert.NotEmpty(t, palette.Added)
		assert.NotEmpty(t, palette.Deleted)
		assert.NotEmpty(t, palette.Modified)
		assert.NotEmpty(t, palette.Context)

		// Syntax colors
		assert.NotEmpty(t, palette.Keyword)
		assert.NotEmpty(t, palette.String)
		assert.NotEmpty(t, palette.Number)
		assert.NotEmpty(t, palette.Comment)
		assert.NotEmpty(t, palette.Operator)
		assert.NotEmpty(t, palette.Function)
		assert.NotEmpty(t, palette.Type)
		assert.NotEmpty(t, palette.Constant)
		assert.NotEmpty(t, palette.Punctuation)

		// UI colors
		assert.NotEmpty(t, palette.UIBackground)
		assert.NotEmpty(t, palette.UIForeground)
		assert.NotEmpty(t, palette.UIAccent)
	})
}

func TestLightTheme(t *testing.T) {
	t.Parallel()

	t.Run("implements Theme interface", func(t *testing.T) {
		t.Parallel()

		var _ diffview.Theme = lipgloss.LightTheme()
	})

	t.Run("returns styles optimized for light backgrounds", func(t *testing.T) {
		t.Parallel()

		theme := lipgloss.LightTheme()
		styles := theme.Styles()

		// Light theme should have all required styles
		assert.NotEmpty(t, styles.Added.Foreground)
		assert.NotEmpty(t, styles.Deleted.Foreground)
		assert.NotEmpty(t, styles.Context.Foreground)
		assert.NotEmpty(t, styles.HunkHeader.Foreground)
		assert.NotEmpty(t, styles.FileHeader.Foreground)
		assert.NotEmpty(t, styles.FileSeparator.Foreground)
	})

	t.Run("dims context lines for better change visibility", func(t *testing.T) {
		t.Parallel()

		theme := lipgloss.LightTheme()
		styles := theme.Styles()

		// Context should use muted gray (#9ca0b0) so changes pop more
		assert.Equal(t, "#9ca0b0", styles.Context.Foreground)
	})

	t.Run("returns highlight styles for word-level diff", func(t *testing.T) {
		t.Parallel()

		theme := lipgloss.LightTheme()
		styles := theme.Styles()

		// Highlight styles should have appropriate colors for light backgrounds
		assert.NotEmpty(t, styles.AddedHighlight.Foreground)
		assert.NotEmpty(t, styles.DeletedHighlight.Foreground)
	})

	t.Run("returns palette with all semantic colors", func(t *testing.T) {
		t.Parallel()

		theme := lipgloss.LightTheme()
		palette := theme.Palette()

		// Base colors
		assert.NotEmpty(t, palette.Background)
		assert.NotEmpty(t, palette.Foreground)

		// Diff colors
		assert.NotEmpty(t, palette.Added)
		assert.NotEmpty(t, palette.Deleted)
		assert.NotEmpty(t, palette.Modified)
		assert.NotEmpty(t, palette.Context)

		// Syntax colors
		assert.NotEmpty(t, palette.Keyword)
		assert.NotEmpty(t, palette.String)
		assert.NotEmpty(t, palette.Number)
		assert.NotEmpty(t, palette.Comment)
		assert.NotEmpty(t, palette.Operator)
		assert.NotEmpty(t, palette.Function)
		assert.NotEmpty(t, palette.Type)
		assert.NotEmpty(t, palette.Constant)
		assert.NotEmpty(t, palette.Punctuation)

		// UI colors
		assert.NotEmpty(t, palette.UIBackground)
		assert.NotEmpty(t, palette.UIForeground)
		assert.NotEmpty(t, palette.UIAccent)
	})
}
