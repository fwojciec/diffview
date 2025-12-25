package chroma_test

import (
	"testing"

	"github.com/fwojciec/diffview"
	"github.com/fwojciec/diffview/chroma"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenizer_Tokenize(t *testing.T) {
	t.Parallel()

	t.Run("tokenizes Go code", func(t *testing.T) {
		t.Parallel()

		tokenizer := chroma.NewTokenizer()
		tokens := tokenizer.Tokenize("go", `package main`)

		require.NotEmpty(t, tokens, "expected tokens for valid Go code")

		// Reconstruct the source from tokens
		var reconstructed string
		for _, tok := range tokens {
			reconstructed += tok.Text
		}
		assert.Equal(t, "package main", reconstructed)

		// Check that keyword "package" gets a style
		var foundPackageKeyword bool
		for _, tok := range tokens {
			if tok.Text == "package" {
				foundPackageKeyword = true
				assert.NotEmpty(t, tok.Style.Foreground, "keyword should have foreground color")
			}
		}
		assert.True(t, foundPackageKeyword, "should find 'package' keyword token")
	})

	t.Run("returns nil for unsupported language", func(t *testing.T) {
		t.Parallel()

		tokenizer := chroma.NewTokenizer()
		tokens := tokenizer.Tokenize("nonexistent-language-xyz", "some code")

		assert.Nil(t, tokens)
	})

	t.Run("handles empty source", func(t *testing.T) {
		t.Parallel()

		tokenizer := chroma.NewTokenizer()
		tokens := tokenizer.Tokenize("go", "")

		assert.Empty(t, tokens)
	})

	t.Run("differentiates function names from builtin names", func(t *testing.T) {
		t.Parallel()

		tokenizer := chroma.NewTokenizer()
		// Code with both a function definition and a builtin call
		tokens := tokenizer.Tokenize("go", `func foo() { println() }`)

		require.NotEmpty(t, tokens)

		var fooStyle, printlnStyle diffview.Style
		for _, tok := range tokens {
			switch tok.Text {
			case "foo":
				fooStyle = tok.Style
			case "println":
				printlnStyle = tok.Style
			}
		}

		// Function name and builtin name should have different colors
		assert.NotEmpty(t, fooStyle.Foreground, "function name should have color")
		assert.NotEmpty(t, printlnStyle.Foreground, "builtin name should have color")
		assert.NotEqual(t, fooStyle.Foreground, printlnStyle.Foreground,
			"function and builtin should have different colors")
	})
}
