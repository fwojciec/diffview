package diffview

// Token represents a syntax-highlighted segment of code.
type Token struct {
	Text  string // The text content of this token
	Style Style  // Visual style to apply (colors, bold, etc.)
}

// Style represents the visual styling for a token.
type Style struct {
	Foreground string // Hex color code (e.g., "#ff0000") or empty for default
	Bold       bool   // Whether the text should be bold
}

// Tokenizer extracts syntax tokens from source code.
type Tokenizer interface {
	// Tokenize splits source code into syntax-highlighted tokens for the given language.
	// Returns nil if the language is not supported.
	Tokenize(language, source string) []Token
}

// LanguageDetector determines the programming language from a file path.
type LanguageDetector interface {
	// DetectFromPath returns the language name for the given path,
	// or an empty string if the language cannot be determined.
	// Accepts paths with or without "a/" or "b/" prefixes (common in diffs).
	DetectFromPath(path string) string
}
