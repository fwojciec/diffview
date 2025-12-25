package diffview

// Color is a hex string in "#RRGGBB" format (e.g., "#ff0000" for red).
// Empty string indicates no color (use terminal default).
type Color string

// Palette defines semantic colors for a theme.
// All colors are hex strings in "#RRGGBB" format.
type Palette struct {
	// Base colors
	Background Color // Primary background
	Foreground Color // Primary foreground/text

	// Diff colors
	Added    Color // Added lines and text
	Deleted  Color // Deleted lines and text
	Modified Color // Modified content
	Context  Color // Unchanged context lines

	// Syntax highlighting colors
	Keyword     Color // Language keywords (if, for, func, etc.)
	String      Color // String literals
	Number      Color // Numeric literals
	Comment     Color // Comments
	Operator    Color // Operators (+, -, =, etc.)
	Function    Color // Function names
	Type        Color // Type names
	Constant    Color // Constants and boolean literals
	Punctuation Color // Brackets, semicolons, etc.

	// UI colors
	UIBackground Color // Secondary background (panels, sidebars)
	UIForeground Color // Secondary foreground (dimmed text)
	UIAccent     Color // Accent color (highlights, focus)
}

// ColorPair represents a foreground and background color combination.
// Colors should be hex strings in "#RRGGBB" format (e.g., "#ff0000" for red).
// Empty strings are valid and indicate no color override (use terminal default).
type ColorPair struct {
	Foreground string
	Background string
}

// Styles contains color pairs for all visual elements in a diff.
type Styles struct {
	Added            ColorPair // Style for added lines (+)
	Deleted          ColorPair // Style for deleted lines (-)
	Context          ColorPair // Style for context lines (unchanged)
	HunkHeader       ColorPair // Style for hunk headers (@@ ... @@)
	FileHeader       ColorPair // Style for file headers (--- a/... +++ b/...)
	FileSeparator    ColorPair // Style for separator lines between files
	LineNumber       ColorPair // Style for line numbers in the gutter
	AddedHighlight   ColorPair // Style for changed text within added lines (word-level diff)
	DeletedHighlight ColorPair // Style for changed text within deleted lines (word-level diff)
}

// Theme provides styles for rendering diffs.
// Different implementations can provide light/dark variants.
type Theme interface {
	Styles() Styles
	Palette() Palette
}
