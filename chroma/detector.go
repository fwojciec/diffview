package chroma

import (
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/fwojciec/diffview"
)

// Compile-time interface verification.
var _ diffview.LanguageDetector = (*Detector)(nil)

// Detector detects programming languages from file paths using chroma.
type Detector struct{}

// NewDetector creates a new chroma-based language detector.
func NewDetector() *Detector {
	return &Detector{}
}

// DetectFromPath returns the language name for the given path,
// or an empty string if the language cannot be determined.
// Strips "a/" or "b/" prefixes common in diff output.
func (d *Detector) DetectFromPath(path string) string {
	// Strip common diff prefixes
	path = strings.TrimPrefix(path, "a/")
	path = strings.TrimPrefix(path, "b/")

	// Get just the filename for extension matching
	filename := filepath.Base(path)

	lexer := lexers.Match(filename)
	if lexer == nil {
		return ""
	}

	return lexer.Config().Name
}
