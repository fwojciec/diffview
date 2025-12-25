// Package chroma provides syntax highlighting using the chroma library.
package chroma

import (
	chromalib "github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/fwojciec/diffview"
)

// Compile-time interface verification.
var _ diffview.Tokenizer = (*Tokenizer)(nil)

// StyleFunc maps chroma token types to diffview styles.
type StyleFunc func(chromalib.TokenType) diffview.Style

// Tokenizer extracts syntax tokens using chroma.
type Tokenizer struct {
	styleFunc StyleFunc
}

// NewTokenizer creates a new chroma-based tokenizer with the given style function.
// Use StyleFromPalette to create a style function from a diffview.Palette.
func NewTokenizer(styleFunc StyleFunc) *Tokenizer {
	if styleFunc == nil {
		panic("chroma: styleFunc cannot be nil")
	}
	return &Tokenizer{styleFunc: styleFunc}
}

// Tokenize splits source code into syntax-highlighted tokens for the given language.
// Returns nil if the language is not supported or an error occurs.
// Returns an empty slice for empty source (valid input, no tokens).
func (t *Tokenizer) Tokenize(language, source string) []diffview.Token {
	if source == "" {
		return []diffview.Token{}
	}

	lexer := lexers.Get(language)
	if lexer == nil {
		return nil
	}

	// Coalesce for better performance with consecutive tokens of the same type
	lexer = chromalib.Coalesce(lexer)

	iterator, err := lexer.Tokenise(nil, source)
	if err != nil {
		return nil
	}

	var tokens []diffview.Token
	for token := iterator(); token != chromalib.EOF; token = iterator() {
		style := t.styleFunc(token.Type)
		tokens = append(tokens, diffview.Token{
			Text:  token.Value,
			Style: style,
		})
	}

	return tokens
}
