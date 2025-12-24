package diffview

// ColorPair represents a foreground and background color combination.
// Colors should be hex strings in "#RRGGBB" format (e.g., "#ff0000" for red).
// Empty strings are valid and indicate no color override (use terminal default).
type ColorPair struct {
	Foreground string
	Background string
}

// Styles contains color pairs for all visual elements in a diff.
type Styles struct {
	Added      ColorPair // Style for added lines (+)
	Deleted    ColorPair // Style for deleted lines (-)
	Context    ColorPair // Style for context lines (unchanged)
	HunkHeader ColorPair // Style for hunk headers (@@ ... @@)
	FileHeader ColorPair // Style for file headers (--- a/... +++ b/...)
}

// Theme provides styles for rendering diffs.
// Different implementations can provide light/dark variants.
type Theme interface {
	Styles() Styles
}
