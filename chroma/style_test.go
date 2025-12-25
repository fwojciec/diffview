package chroma_test

import (
	"testing"

	chromalib "github.com/alecthomas/chroma/v2"
	"github.com/fwojciec/diffview"
	"github.com/fwojciec/diffview/chroma"
	"github.com/stretchr/testify/assert"
)

func TestStyleFromPalette(t *testing.T) {
	t.Parallel()

	palette := diffview.Palette{
		Background:  "#000000",
		Foreground:  "#ffffff",
		Keyword:     "#ff00ff",
		String:      "#00ff00",
		Number:      "#ff8800",
		Comment:     "#888888",
		Operator:    "#00ffff",
		Function:    "#0000ff",
		Type:        "#ffff00",
		Constant:    "#ff8800",
		Punctuation: "#aaaaaa",
	}

	styleFunc := chroma.StyleFromPalette(palette)

	t.Run("keywords are bold with palette color", func(t *testing.T) {
		t.Parallel()
		style := styleFunc(chromalib.Keyword)
		assert.Equal(t, "#ff00ff", style.Foreground)
		assert.True(t, style.Bold)
	})

	t.Run("strings use palette color", func(t *testing.T) {
		t.Parallel()
		style := styleFunc(chromalib.String)
		assert.Equal(t, "#00ff00", style.Foreground)
	})

	t.Run("numbers use palette color", func(t *testing.T) {
		t.Parallel()
		style := styleFunc(chromalib.Number)
		assert.Equal(t, "#ff8800", style.Foreground)
	})

	t.Run("comments use palette color", func(t *testing.T) {
		t.Parallel()
		style := styleFunc(chromalib.Comment)
		assert.Equal(t, "#888888", style.Foreground)
	})

	t.Run("operators use palette color", func(t *testing.T) {
		t.Parallel()
		style := styleFunc(chromalib.Operator)
		assert.Equal(t, "#00ffff", style.Foreground)
	})

	t.Run("function names use palette color", func(t *testing.T) {
		t.Parallel()
		style := styleFunc(chromalib.NameFunction)
		assert.Equal(t, "#0000ff", style.Foreground)
	})

	t.Run("type keywords use palette color and are bold", func(t *testing.T) {
		t.Parallel()
		style := styleFunc(chromalib.KeywordType)
		assert.Equal(t, "#ffff00", style.Foreground)
		assert.True(t, style.Bold)
	})

	t.Run("constants use palette color", func(t *testing.T) {
		t.Parallel()
		style := styleFunc(chromalib.NameConstant)
		assert.Equal(t, "#ff8800", style.Foreground)
	})

	t.Run("punctuation uses palette color", func(t *testing.T) {
		t.Parallel()
		style := styleFunc(chromalib.Punctuation)
		assert.Equal(t, "#aaaaaa", style.Foreground)
	})

	t.Run("unknown token types return empty style", func(t *testing.T) {
		t.Parallel()
		style := styleFunc(chromalib.Error)
		assert.Empty(t, style.Foreground)
		assert.False(t, style.Bold)
	})
}
