package lipgloss_test

import (
	"testing"

	"github.com/fwojciec/diffview"
	"github.com/fwojciec/diffview/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestNewTheme(t *testing.T) {
	t.Parallel()

	t.Run("derives added style from palette", func(t *testing.T) {
		t.Parallel()

		palette := diffview.Palette{
			Background: "#000000",
			Added:      "#00ff00",
		}

		theme := lipgloss.NewTheme(palette)
		styles := theme.Styles()

		assert.Equal(t, "#00ff00", styles.Added.Foreground)
	})

	t.Run("derives deleted style from palette", func(t *testing.T) {
		t.Parallel()

		palette := diffview.Palette{
			Background: "#000000",
			Deleted:    "#ff0000",
		}

		theme := lipgloss.NewTheme(palette)
		styles := theme.Styles()

		assert.Equal(t, "#ff0000", styles.Deleted.Foreground)
	})

	t.Run("derives context style from palette", func(t *testing.T) {
		t.Parallel()

		palette := diffview.Palette{
			Context: "#888888",
		}

		theme := lipgloss.NewTheme(palette)
		styles := theme.Styles()

		assert.Equal(t, "#888888", styles.Context.Foreground)
	})

	t.Run("derives hunk header style from palette", func(t *testing.T) {
		t.Parallel()

		palette := diffview.Palette{
			UIAccent: "#0000ff",
		}

		theme := lipgloss.NewTheme(palette)
		styles := theme.Styles()

		assert.Equal(t, "#0000ff", styles.HunkHeader.Foreground)
	})

	t.Run("derives file header style from palette", func(t *testing.T) {
		t.Parallel()

		palette := diffview.Palette{
			Modified:     "#ffff00",
			UIBackground: "#333333",
		}

		theme := lipgloss.NewTheme(palette)
		styles := theme.Styles()

		assert.Equal(t, "#ffff00", styles.FileHeader.Foreground)
		assert.Equal(t, "#333333", styles.FileHeader.Background)
	})

	t.Run("derives file separator style from palette", func(t *testing.T) {
		t.Parallel()

		palette := diffview.Palette{
			UIForeground: "#666666",
		}

		theme := lipgloss.NewTheme(palette)
		styles := theme.Styles()

		assert.Equal(t, "#666666", styles.FileSeparator.Foreground)
	})

	t.Run("derives line number style from palette", func(t *testing.T) {
		t.Parallel()

		palette := diffview.Palette{
			Context: "#777777",
		}

		theme := lipgloss.NewTheme(palette)
		styles := theme.Styles()

		assert.Equal(t, "#777777", styles.LineNumber.Foreground)
	})

	t.Run("stores palette for retrieval", func(t *testing.T) {
		t.Parallel()

		palette := diffview.Palette{
			Background: "#111111",
			Foreground: "#eeeeee",
			Added:      "#00ff00",
		}

		theme := lipgloss.NewTheme(palette)

		assert.Equal(t, palette, theme.Palette())
	})

	t.Run("derives added highlight with bright background", func(t *testing.T) {
		t.Parallel()

		palette := diffview.Palette{
			Background: "#1e1e2e",
			Added:      "#00ff00",
		}

		theme := lipgloss.NewTheme(palette)
		styles := theme.Styles()

		// Highlight uses added color as background with contrasting foreground
		assert.Equal(t, "#00ff00", styles.AddedHighlight.Background)
		assert.Equal(t, "#1e1e2e", styles.AddedHighlight.Foreground)
	})

	t.Run("derives deleted highlight with bright background", func(t *testing.T) {
		t.Parallel()

		palette := diffview.Palette{
			Background: "#1e1e2e",
			Deleted:    "#ff0000",
		}

		theme := lipgloss.NewTheme(palette)
		styles := theme.Styles()

		// Highlight uses deleted color as background with contrasting foreground
		assert.Equal(t, "#ff0000", styles.DeletedHighlight.Background)
		assert.Equal(t, "#1e1e2e", styles.DeletedHighlight.Foreground)
	})
}

func TestDefaultTheme(t *testing.T) {
	t.Parallel()

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

	t.Run("derives styles from its palette", func(t *testing.T) {
		t.Parallel()

		theme := lipgloss.DefaultTheme()
		styles := theme.Styles()
		palette := theme.Palette()

		// Styles should be derived from palette colors
		assert.Equal(t, string(palette.Added), styles.Added.Foreground)
		assert.Equal(t, string(palette.Deleted), styles.Deleted.Foreground)
		assert.Equal(t, string(palette.Context), styles.Context.Foreground)
	})

	t.Run("uses Catppuccin Mocha colors", func(t *testing.T) {
		t.Parallel()

		theme := lipgloss.DefaultTheme()
		palette := theme.Palette()

		// Catppuccin Mocha base colors
		assert.Equal(t, diffview.Color("#1e1e2e"), palette.Background)
		assert.Equal(t, diffview.Color("#cdd6f4"), palette.Foreground)
	})
}

func TestTestTheme(t *testing.T) {
	t.Parallel()

	t.Run("uses predictable pure colors for testing", func(t *testing.T) {
		t.Parallel()

		theme := lipgloss.TestTheme()
		palette := theme.Palette()

		// Pure colors for easy test assertions
		assert.Equal(t, diffview.Color("#000000"), palette.Background)
		assert.Equal(t, diffview.Color("#ffffff"), palette.Foreground)
		assert.Equal(t, diffview.Color("#00ff00"), palette.Added)
		assert.Equal(t, diffview.Color("#ff0000"), palette.Deleted)
	})

	t.Run("derives styles from its palette", func(t *testing.T) {
		t.Parallel()

		theme := lipgloss.TestTheme()
		styles := theme.Styles()
		palette := theme.Palette()

		// Styles should be derived from palette colors
		assert.Equal(t, string(palette.Added), styles.Added.Foreground)
		assert.Equal(t, string(palette.Deleted), styles.Deleted.Foreground)
	})
}
