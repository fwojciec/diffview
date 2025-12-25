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

// DefaultTheme returns the default theme (dark background optimized).
func DefaultTheme() *Theme {
	return DarkTheme()
}

// DarkTheme returns a theme optimized for dark terminal backgrounds.
// Background colors are very dark to allow syntax highlighting colors to remain readable.
func DarkTheme() *Theme {
	return &Theme{
		styles: diffview.Styles{
			Added: diffview.ColorPair{
				Foreground: "#a6e3a1", // Green
				Background: "#004000", // Very dark green - syntax colors stay readable
			},
			Deleted: diffview.ColorPair{
				Foreground: "#f38ba8", // Red
				Background: "#3f0001", // Very dark red - syntax colors stay readable
			},
			Context: diffview.ColorPair{
				Foreground: "#6c7086", // Muted gray (dimmed for change visibility)
			},
			HunkHeader: diffview.ColorPair{
				Foreground: "#89b4fa", // Blue
			},
			FileHeader: diffview.ColorPair{
				Foreground: "#f9e2af", // Yellow
				Background: "#313244", // Dark surface
			},
			FileSeparator: diffview.ColorPair{
				Foreground: "#45475a", // Muted gray (subtle)
			},
			LineNumber: diffview.ColorPair{
				Foreground: "#6c7086", // Muted gray
			},
			AddedHighlight: diffview.ColorPair{
				Foreground: "#1e1e2e", // Dark text on bright background
				Background: "#a6e3a1", // Bright green background
			},
			DeletedHighlight: diffview.ColorPair{
				Foreground: "#1e1e2e", // Dark text on bright background
				Background: "#f38ba8", // Bright red background
			},
		},
		palette: diffview.Palette{
			// Base colors (Catppuccin Mocha)
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
				Foreground: "#9ca0b0", // Muted gray (dimmed for change visibility)
			},
			HunkHeader: diffview.ColorPair{
				Foreground: "#1e66f5", // Blue
			},
			FileHeader: diffview.ColorPair{
				Foreground: "#df8e1d", // Yellow
				Background: "#e6e9ef", // Light surface
			},
			FileSeparator: diffview.ColorPair{
				Foreground: "#bcc0cc", // Muted gray (subtle for light)
			},
			LineNumber: diffview.ColorPair{
				Foreground: "#9ca0b0", // Muted gray for light theme
			},
			AddedHighlight: diffview.ColorPair{
				Foreground: "#ffffff", // White text on dark background
				Background: "#40a02b", // Bright green background
			},
			DeletedHighlight: diffview.ColorPair{
				Foreground: "#ffffff", // White text on dark background
				Background: "#d20f39", // Bright red background
			},
		},
		palette: diffview.Palette{
			// Base colors (Catppuccin Latte)
			Background: "#eff1f5",
			Foreground: "#4c4f69",

			// Diff colors
			Added:    "#40a02b",
			Deleted:  "#d20f39",
			Modified: "#df8e1d",
			Context:  "#9ca0b0",

			// Syntax highlighting colors
			Keyword:     "#8839ef",
			String:      "#40a02b",
			Number:      "#fe640b",
			Comment:     "#9ca0b0",
			Operator:    "#04a5e5",
			Function:    "#1e66f5",
			Type:        "#df8e1d",
			Constant:    "#fe640b",
			Punctuation: "#6c6f85",

			// UI colors
			UIBackground: "#e6e9ef",
			UIForeground: "#6c6f85",
			UIAccent:     "#1e66f5",
		},
	}
}
