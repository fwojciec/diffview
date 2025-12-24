// Package lipgloss provides theme implementations using the Lipgloss styling library.
package lipgloss

import "github.com/fwojciec/diffview"

// Compile-time interface verification.
var _ diffview.Theme = (*Theme)(nil)

// Theme implements diffview.Theme with Lipgloss-compatible colors.
type Theme struct {
	styles diffview.Styles
}

// Styles returns the color styles for this theme.
func (t *Theme) Styles() diffview.Styles {
	return t.styles
}

// DefaultTheme returns the default theme (dark background optimized).
func DefaultTheme() *Theme {
	return DarkTheme()
}

// DarkTheme returns a theme optimized for dark terminal backgrounds.
func DarkTheme() *Theme {
	return &Theme{
		styles: diffview.Styles{
			Added: diffview.ColorPair{
				Foreground: "#a6e3a1", // Green
				Background: "#2d3f2d", // Subtle green background
			},
			Deleted: diffview.ColorPair{
				Foreground: "#f38ba8", // Red
				Background: "#3f2d2d", // Subtle red background
			},
			Context: diffview.ColorPair{
				Foreground: "#cdd6f4", // Light gray
			},
			HunkHeader: diffview.ColorPair{
				Foreground: "#89b4fa", // Blue
			},
			FileHeader: diffview.ColorPair{
				Foreground: "#f9e2af", // Yellow
				Background: "#313244", // Dark surface
			},
			LineNumber: diffview.ColorPair{
				Foreground: "#6c7086", // Muted gray
			},
		},
	}
}

// LightTheme returns a theme optimized for light terminal backgrounds.
func LightTheme() *Theme {
	return &Theme{
		styles: diffview.Styles{
			Added: diffview.ColorPair{
				Foreground: "#40a02b", // Green
				Background: "#d4f4d4", // Subtle green background
			},
			Deleted: diffview.ColorPair{
				Foreground: "#d20f39", // Red
				Background: "#f4d4d4", // Subtle red background
			},
			Context: diffview.ColorPair{
				Foreground: "#4c4f69", // Dark gray
			},
			HunkHeader: diffview.ColorPair{
				Foreground: "#1e66f5", // Blue
			},
			FileHeader: diffview.ColorPair{
				Foreground: "#df8e1d", // Yellow
				Background: "#e6e9ef", // Light surface
			},
			LineNumber: diffview.ColorPair{
				Foreground: "#9ca0b0", // Muted gray for light theme
			},
		},
	}
}
