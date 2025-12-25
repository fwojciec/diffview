package chroma_test

import (
	"testing"

	chromalib "github.com/alecthomas/chroma/v2"
	"github.com/fwojciec/diffview"
	"github.com/fwojciec/diffview/chroma"
	"github.com/fwojciec/diffview/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testStyleFunc returns a style function using the test palette.
func testStyleFunc() func(chromalib.TokenType) diffview.Style {
	return chroma.StyleFromPalette(lipgloss.TestTheme().Palette())
}

func TestTokenizer_Tokenize(t *testing.T) {
	t.Parallel()

	t.Run("tokenizes Go code", func(t *testing.T) {
		t.Parallel()

		tokenizer, err := chroma.NewTokenizer(testStyleFunc())
		require.NoError(t, err)
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

		tokenizer, err := chroma.NewTokenizer(testStyleFunc())
		require.NoError(t, err)
		tokens := tokenizer.Tokenize("nonexistent-language-xyz", "some code")

		assert.Nil(t, tokens)
	})

	t.Run("handles empty source", func(t *testing.T) {
		t.Parallel()

		tokenizer, err := chroma.NewTokenizer(testStyleFunc())
		require.NoError(t, err)
		tokens := tokenizer.Tokenize("go", "")

		assert.Empty(t, tokens)
	})

	t.Run("styles function names", func(t *testing.T) {
		t.Parallel()

		tokenizer, err := chroma.NewTokenizer(testStyleFunc())
		require.NoError(t, err)
		// Code with a function definition
		tokens := tokenizer.Tokenize("go", `func foo() {}`)

		require.NotEmpty(t, tokens)

		var fooStyle diffview.Style
		for _, tok := range tokens {
			if tok.Text == "foo" {
				fooStyle = tok.Style
				break
			}
		}

		assert.NotEmpty(t, fooStyle.Foreground, "function name should have color")
	})

	t.Run("uses colors from provided palette", func(t *testing.T) {
		t.Parallel()

		// Use test palette which has known colors
		palette := lipgloss.TestTheme().Palette()
		tokenizer, err := chroma.NewTokenizer(chroma.StyleFromPalette(palette))
		require.NoError(t, err)
		tokens := tokenizer.Tokenize("go", `package main`)

		require.NotEmpty(t, tokens)

		// Find the "package" keyword and verify it uses the palette's keyword color
		for _, tok := range tokens {
			if tok.Text == "package" {
				assert.Equal(t, string(palette.Keyword), tok.Style.Foreground,
					"keyword should use palette's keyword color")
				assert.True(t, tok.Style.Bold, "keyword should be bold")
				return
			}
		}
		t.Fatal("did not find 'package' keyword in tokens")
	})

	t.Run("returns error for nil styleFunc", func(t *testing.T) {
		t.Parallel()

		_, err := chroma.NewTokenizer(nil)
		assert.Error(t, err)
	})
}
