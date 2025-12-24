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
	})
}
