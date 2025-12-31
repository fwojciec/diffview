// Package chroma provides syntax highlighting using the chroma library.
package chroma

import (
	"errors"
	"strings"

	chromalib "github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/fwojciec/diffstory"
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
func NewTokenizer(styleFunc StyleFunc) (*Tokenizer, error) {
	if styleFunc == nil {
		return nil, errors.New("chroma: styleFunc cannot be nil")
	}
	return &Tokenizer{styleFunc: styleFunc}, nil
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

// TokenizeLines tokenizes source code with full context, then splits tokens by line.
// This correctly handles multi-line constructs like /* */ comments and JSDoc.
// Returns nil if the language is not supported or an error occurs.
// Returns an empty slice for empty source.
func (t *Tokenizer) TokenizeLines(language, source string) [][]diffview.Token {
	if source == "" {
		return [][]diffview.Token{}
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

	// Collect all tokens first (with full context)
	var allTokens []diffview.Token
	for token := iterator(); token != chromalib.EOF; token = iterator() {
		style := t.styleFunc(token.Type)
		allTokens = append(allTokens, diffview.Token{
			Text:  token.Value,
			Style: style,
		})
	}

	// Split tokens by newlines
	return splitTokensByLine(allTokens)
}

// splitTokensByLine splits a flat list of tokens into per-line token slices.
// Handles tokens that span multiple lines by splitting them at newline boundaries.
func splitTokensByLine(tokens []diffview.Token) [][]diffview.Token {
	if len(tokens) == 0 {
		return [][]diffview.Token{}
	}

	var result [][]diffview.Token
	var currentLine []diffview.Token

	for _, tok := range tokens {
		// Token without newlines goes directly to current line
		if !strings.Contains(tok.Text, "\n") {
			currentLine = append(currentLine, tok)
			continue
		}

		// Split the token at newline boundaries
		parts := strings.Split(tok.Text, "\n")
		for i, part := range parts {
			if part != "" {
				currentLine = append(currentLine, diffview.Token{
					Text:  part,
					Style: tok.Style,
				})
			}
			// If this isn't the last part, we hit a newline - finalize the line
			if i < len(parts)-1 {
				result = append(result, currentLine)
				currentLine = nil
			}
		}
	}

	// Don't forget the last line if it has content
	if len(currentLine) > 0 {
		result = append(result, currentLine)
	}

	return result
}
