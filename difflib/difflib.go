package difflib

import "regexp"

// Differ tokenizes strings and computes word-level diffs.
type Differ struct {
	tokenPattern *regexp.Regexp
}

// NewDiffer creates a new Differ instance.
func NewDiffer() *Differ {
	return &Differ{
		tokenPattern: regexp.MustCompile(
			`[a-zA-Z_][a-zA-Z0-9_]*|` + // identifiers
				`[0-9]+\.?[0-9]*|` + // numbers
				`"[^"]*"|'[^']*'|` + // string literals
				`[+\-*/=<>!&|^%:]+|` + // operators (including :)
				`[(){}\[\];,.]|` + // punctuation
				`\s+|` + // whitespace
				`.`, // catch-all for any remaining character
		),
	}
}

// Tokenize splits a string into tokens.
func (d *Differ) Tokenize(s string) []string {
	return d.tokenPattern.FindAllString(s, -1)
}
