// Package lipgloss provides theme implementations using the Lipgloss styling library.
package lipgloss

import "github.com/fwojciec/diffview"

// Compile-time interface verification.
var _ diffview.Theme = (*Theme)(nil)

// Theme implements diffview.Theme with Lipgloss-compatible colors.
type Theme struct {
	styles  diffview.Styles
	palette diffview.Palette
}

// Styles returns the color styles for this theme.
func (t *Theme) Styles() diffview.Styles {
	return t.styles
}

// Palette returns the semantic color palette for this theme.
func (t *Theme) Palette() diffview.Palette {
	return t.palette
}

// NewTheme creates a Theme from a Palette, deriving all styles from the palette colors.
func NewTheme(p diffview.Palette) *Theme {
	return &Theme{
		palette: p,
		styles:  stylesFromPalette(p),
	}
}

// stylesFromPalette derives Styles from a Palette.
func stylesFromPalette(p diffview.Palette) diffview.Styles {
	return diffview.Styles{
		Added: diffview.ColorPair{
			Foreground: string(p.Added),
		},
		Deleted: diffview.ColorPair{
			Foreground: string(p.Deleted),
		},
		Context: diffview.ColorPair{
			Foreground: string(p.Context),
		},
		HunkHeader: diffview.ColorPair{
			Foreground: string(p.UIAccent),
		},
		FileHeader: diffview.ColorPair{
			Foreground: string(p.Modified),
			Background: string(p.UIBackground),
		},
		FileSeparator: diffview.ColorPair{
			Foreground: string(p.UIForeground),
		},
		LineNumber: diffview.ColorPair{
			Foreground: string(p.Context),
		},
		AddedHighlight: diffview.ColorPair{
			Foreground: string(p.Background),
			Background: string(p.Added),
		},
		DeletedHighlight: diffview.ColorPair{
			Foreground: string(p.Background),
			Background: string(p.Deleted),
		},
	}
}

// DefaultTheme returns the default theme (Catppuccin Mocha, dark background optimized).
func DefaultTheme() *Theme {
	return NewTheme(mochaPalette())
}

// TestTheme returns a theme with predictable, pure colors for testing.
// Uses simple hex colors that are easy to assert in tests.
func TestTheme() *Theme {
	return NewTheme(testPalette())
}

// testPalette returns a palette with predictable, pure colors for testing.
func testPalette() diffview.Palette {
	return diffview.Palette{
		// Base colors
		Background: "#000000",
		Foreground: "#ffffff",

		// Diff colors
		Added:    "#00ff00",
		Deleted:  "#ff0000",
		Modified: "#ffff00",
		Context:  "#888888",

		// Syntax highlighting colors
		Keyword:     "#ff00ff",
		String:      "#00ff00",
		Number:      "#ff8800",
		Comment:     "#888888",
		Operator:    "#00ffff",
		Function:    "#0000ff",
		Type:        "#ffff00",
		Constant:    "#ff8800",
		Punctuation: "#aaaaaa",

		// UI colors
		UIBackground: "#333333",
		UIForeground: "#cccccc",
		UIAccent:     "#0000ff",
	}
}

// mochaPalette returns the Catppuccin Mocha color palette.
func mochaPalette() diffview.Palette {
	return diffview.Palette{
		// Base colors
		Background: "#1e1e2e",
		Foreground: "#cdd6f4",

		// Diff colors
		Added:    "#a6e3a1",
		Deleted:  "#f38ba8",
		Modified: "#f9e2af",
		Context:  "#6c7086",

		// Syntax highlighting colors
		Keyword:     "#cba6f7",
		String:      "#a6e3a1",
		Number:      "#fab387",
		Comment:     "#6c7086",
		Operator:    "#89dceb",
		Function:    "#89b4fa",
		Type:        "#f9e2af",
		Constant:    "#fab387",
		Punctuation: "#9399b2",

		// UI colors
		UIBackground: "#313244",
		UIForeground: "#a6adc8",
		UIAccent:     "#89b4fa",
	}
}
