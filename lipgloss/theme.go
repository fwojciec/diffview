// Package lipgloss provides theme implementations using the Lipgloss styling library.
package lipgloss

import (
	"fmt"

	"github.com/fwojciec/diffview"
)

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
			Foreground: string(p.Foreground),
			Background: blendWithBackground(p.Added, p.Background, 0.15),
		},
		Deleted: diffview.ColorPair{
			Foreground: string(p.Foreground),
			Background: blendWithBackground(p.Deleted, p.Background, 0.15),
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
		AddedGutter: diffview.ColorPair{
			Foreground: string(p.Foreground), // Same as code line foreground
			Background: blendWithBackground(p.Added, p.Background, 0.35),
		},
		DeletedGutter: diffview.ColorPair{
			Foreground: string(p.Foreground), // Same as code line foreground
			Background: blendWithBackground(p.Deleted, p.Background, 0.35),
		},
		AddedHighlight: diffview.ColorPair{
			Foreground: string(p.Foreground),                             // Same as code line foreground (neutral)
			Background: blendWithBackground(p.Added, p.Background, 0.35), // Same as gutter
		},
		DeletedHighlight: diffview.ColorPair{
			Foreground: string(p.Foreground),                               // Same as code line foreground (neutral)
			Background: blendWithBackground(p.Deleted, p.Background, 0.35), // Same as gutter
		},
	}
}

// DefaultTheme returns the default theme (GitHub-inspired dark theme).
func DefaultTheme() *Theme {
	return NewTheme(githubDarkPalette())
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

// githubDarkPalette returns a GitHub-inspired dark theme color palette.
// Based on GitHub's Primer design system dark mode colors.
func githubDarkPalette() diffview.Palette {
	return diffview.Palette{
		// Base colors - GitHub dark mode canvas
		Background: "#0d1117",
		Foreground: "#e6edf3",

		// Diff colors - GitHub success/danger semantic colors
		Added:    "#3fb950", // GitHub green for additions
		Deleted:  "#f85149", // GitHub red for deletions
		Modified: "#d29922", // GitHub warning/modified yellow
		Context:  "#8b949e", // GitHub muted foreground

		// Syntax highlighting colors - GitHub dark mode syntax
		Keyword:     "#ff7b72", // Red for keywords
		String:      "#a5d6ff", // Light blue for strings
		Number:      "#79c0ff", // Blue for numbers
		Comment:     "#8b949e", // Muted for comments
		Operator:    "#ff7b72", // Red for operators
		Function:    "#d2a8ff", // Purple for functions
		Type:        "#ffa657", // Orange for types
		Constant:    "#79c0ff", // Blue for constants
		Punctuation: "#8b949e", // Muted for punctuation

		// UI colors - GitHub dark mode surfaces
		UIBackground: "#161b22", // Elevated surface
		UIForeground: "#8b949e", // Muted text
		UIAccent:     "#58a6ff", // GitHub blue accent
	}
}

// blendWithBackground creates a subtle background color by blending
// the accent color with the background color at the given ratio.
// ratio of 0.15 means 15% accent, 85% background.
func blendWithBackground(accent, background diffview.Color, ratio float64) string {
	ar, ag, ab := parseHex(string(accent))
	br, bg, bb := parseHex(string(background))

	r := int(float64(ar)*ratio + float64(br)*(1-ratio))
	g := int(float64(ag)*ratio + float64(bg)*(1-ratio))
	b := int(float64(ab)*ratio + float64(bb)*(1-ratio))

	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// parseHex parses a hex color string like "#rrggbb" into RGB components.
func parseHex(hex string) (r, g, b int) {
	if len(hex) != 7 || hex[0] != '#' {
		return 0, 0, 0
	}
	n, err := fmt.Sscanf(hex[1:], "%02x%02x%02x", &r, &g, &b)
	if err != nil || n != 3 {
		return 0, 0, 0
	}
	return r, g, b
}
