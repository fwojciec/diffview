package worddiff_test

import (
	"testing"

	"github.com/fwojciec/diffview"
	"github.com/fwojciec/diffview/worddiff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiffer_Diff_SingleWordChange(t *testing.T) {
	t.Parallel()

	d := worddiff.NewDiffer()

	oldSegs, newSegs := d.Diff("hello world", "hello universe")

	require.Len(t, oldSegs, 2)
	assert.Equal(t, diffview.Segment{Text: "hello ", Changed: false}, oldSegs[0])
	assert.Equal(t, diffview.Segment{Text: "world", Changed: true}, oldSegs[1])

	require.Len(t, newSegs, 2)
	assert.Equal(t, diffview.Segment{Text: "hello ", Changed: false}, newSegs[0])
	assert.Equal(t, diffview.Segment{Text: "universe", Changed: true}, newSegs[1])
}

func TestDiffer_Diff_IdenticalStrings(t *testing.T) {
	t.Parallel()

	d := worddiff.NewDiffer()

	oldSegs, newSegs := d.Diff("hello world", "hello world")

	// Identical strings should return single unchanged segment each
	require.Len(t, oldSegs, 1)
	assert.Equal(t, diffview.Segment{Text: "hello world", Changed: false}, oldSegs[0])

	require.Len(t, newSegs, 1)
	assert.Equal(t, diffview.Segment{Text: "hello world", Changed: false}, newSegs[0])
}

func TestDiffer_Diff_CompletelyDifferent(t *testing.T) {
	t.Parallel()

	d := worddiff.NewDiffer()

	oldSegs, newSegs := d.Diff("abc", "xyz")

	// Completely different strings should return single changed segment each
	require.Len(t, oldSegs, 1)
	assert.Equal(t, diffview.Segment{Text: "abc", Changed: true}, oldSegs[0])

	require.Len(t, newSegs, 1)
	assert.Equal(t, diffview.Segment{Text: "xyz", Changed: true}, newSegs[0])
}

func TestDiffer_Diff_MultipleChanges(t *testing.T) {
	t.Parallel()

	d := worddiff.NewDiffer()

	oldSegs, newSegs := d.Diff("function calculate(x, y) {", "function calculate(x, y, z) {")

	// The only difference is ", z" inserted before ")".
	// When text is inserted (not replaced), the old string has no changed segments,
	// and the new string highlights only the inserted portion.

	require.Len(t, oldSegs, 1, "old string has nothing changed (text was added)")
	assert.Equal(t, diffview.Segment{Text: "function calculate(x, y) {", Changed: false}, oldSegs[0])

	require.Len(t, newSegs, 3)
	assert.Equal(t, diffview.Segment{Text: "function calculate(x, y", Changed: false}, newSegs[0])
	assert.Equal(t, diffview.Segment{Text: ", z", Changed: true}, newSegs[1])
	assert.Equal(t, diffview.Segment{Text: ") {", Changed: false}, newSegs[2])
}

func TestDiffer_Diff_EmptyStrings(t *testing.T) {
	t.Parallel()

	d := worddiff.NewDiffer()

	t.Run("both empty", func(t *testing.T) {
		t.Parallel()

		oldSegs, newSegs := d.Diff("", "")

		assert.Empty(t, oldSegs)
		assert.Empty(t, newSegs)
	})

	t.Run("old empty", func(t *testing.T) {
		t.Parallel()

		oldSegs, newSegs := d.Diff("", "new text")

		assert.Empty(t, oldSegs)
		require.Len(t, newSegs, 1)
		assert.Equal(t, diffview.Segment{Text: "new text", Changed: true}, newSegs[0])
	})

	t.Run("new empty", func(t *testing.T) {
		t.Parallel()

		oldSegs, newSegs := d.Diff("old text", "")

		require.Len(t, oldSegs, 1)
		assert.Equal(t, diffview.Segment{Text: "old text", Changed: true}, oldSegs[0])
		assert.Empty(t, newSegs)
	})
}

func TestDiffer_Diff_ChangedAtBeginning(t *testing.T) {
	t.Parallel()

	d := worddiff.NewDiffer()

	oldSegs, newSegs := d.Diff("old prefix unchanged", "new prefix unchanged")

	require.Len(t, oldSegs, 2)
	assert.Equal(t, diffview.Segment{Text: "old", Changed: true}, oldSegs[0])
	assert.Equal(t, diffview.Segment{Text: " prefix unchanged", Changed: false}, oldSegs[1])

	require.Len(t, newSegs, 2)
	assert.Equal(t, diffview.Segment{Text: "new", Changed: true}, newSegs[0])
	assert.Equal(t, diffview.Segment{Text: " prefix unchanged", Changed: false}, newSegs[1])
}

func TestDiffer_Diff_UnicodeCharacters(t *testing.T) {
	t.Parallel()

	d := worddiff.NewDiffer()

	t.Run("emoji change", func(t *testing.T) {
		t.Parallel()

		oldSegs, newSegs := d.Diff("hello 👋 world", "hello 🌍 world")

		require.Len(t, oldSegs, 3)
		assert.Equal(t, diffview.Segment{Text: "hello ", Changed: false}, oldSegs[0])
		assert.Equal(t, diffview.Segment{Text: "👋", Changed: true}, oldSegs[1])
		assert.Equal(t, diffview.Segment{Text: " world", Changed: false}, oldSegs[2])

		require.Len(t, newSegs, 3)
		assert.Equal(t, diffview.Segment{Text: "hello ", Changed: false}, newSegs[0])
		assert.Equal(t, diffview.Segment{Text: "🌍", Changed: true}, newSegs[1])
		assert.Equal(t, diffview.Segment{Text: " world", Changed: false}, newSegs[2])
	})

	t.Run("CJK characters", func(t *testing.T) {
		t.Parallel()

		oldSegs, newSegs := d.Diff("hello 世界", "hello 宇宙")

		require.Len(t, oldSegs, 2)
		assert.Equal(t, diffview.Segment{Text: "hello ", Changed: false}, oldSegs[0])
		assert.Equal(t, diffview.Segment{Text: "世界", Changed: true}, oldSegs[1])

		require.Len(t, newSegs, 2)
		assert.Equal(t, diffview.Segment{Text: "hello ", Changed: false}, newSegs[0])
		assert.Equal(t, diffview.Segment{Text: "宇宙", Changed: true}, newSegs[1])
	})
}
