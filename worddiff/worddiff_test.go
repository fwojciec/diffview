package worddiff_test

import (
	"testing"

	"github.com/fwojciec/diffview"
	"github.com/fwojciec/diffview/worddiff"
	"github.com/stretchr/testify/assert"
)

func TestTokenize(t *testing.T) {
	t.Parallel()

	d := worddiff.NewDiffer()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		// Identifiers
		{
			name:     "simple identifier",
			input:    "func",
			expected: []string{"func"},
		},
		{
			name:     "camelCase identifier",
			input:    "myVariable",
			expected: []string{"myVariable"},
		},
		{
			name:     "underscore identifier",
			input:    "_privateVar",
			expected: []string{"_privateVar"},
		},
		{
			name:     "identifier with numbers",
			input:    "var123",
			expected: []string{"var123"},
		},

		// Numbers
		{
			name:     "integer",
			input:    "123",
			expected: []string{"123"},
		},
		{
			name:     "float",
			input:    "3.14",
			expected: []string{"3.14"},
		},
		{
			name:     "trailing dot not part of number",
			input:    "123.",
			expected: []string{"123", "."},
		},
		{
			name:     "leading dot not part of number",
			input:    ".5",
			expected: []string{".", "5"},
		},

		// String literals
		{
			name:     "double quoted string",
			input:    `"hello"`,
			expected: []string{`"hello"`},
		},
		{
			name:     "single quoted string",
			input:    "'x'",
			expected: []string{"'x'"},
		},
		{
			name:     "escaped quote in double quoted string",
			input:    `"say \"hello\""`,
			expected: []string{`"say \"hello\""`},
		},
		{
			name:     "escaped quote in single quoted string",
			input:    `'it\'s'`,
			expected: []string{`'it\'s'`},
		},
		{
			name:     "escaped backslash in string",
			input:    `"path\\to\\file"`,
			expected: []string{`"path\\to\\file"`},
		},

		// Operators
		{
			name:     "plus operator",
			input:    "+",
			expected: []string{"+"},
		},
		{
			name:     "assignment operator",
			input:    ":=",
			expected: []string{":="},
		},
		{
			name:     "equality operator",
			input:    "==",
			expected: []string{"=="},
		},

		// Punctuation
		{
			name:     "parentheses",
			input:    "()",
			expected: []string{"(", ")"},
		},
		{
			name:     "braces",
			input:    "{}",
			expected: []string{"{", "}"},
		},
		{
			name:     "brackets",
			input:    "[]",
			expected: []string{"[", "]"},
		},
		{
			name:     "semicolon",
			input:    ";",
			expected: []string{";"},
		},
		{
			name:     "comma",
			input:    ",",
			expected: []string{","},
		},

		// Whitespace preserved
		{
			name:     "space preserved",
			input:    "a b",
			expected: []string{"a", " ", "b"},
		},
		{
			name:     "multiple spaces preserved",
			input:    "a  b",
			expected: []string{"a", "  ", "b"},
		},
		{
			name:     "tab preserved",
			input:    "a\tb",
			expected: []string{"a", "\t", "b"},
		},

		// Combined expressions
		{
			name:     "simple expression",
			input:    "x + y",
			expected: []string{"x", " ", "+", " ", "y"},
		},
		{
			name:     "function call",
			input:    "foo(1, 2)",
			expected: []string{"foo", "(", "1", ",", " ", "2", ")"},
		},
		{
			name:     "assignment",
			input:    "x := 42",
			expected: []string{"x", " ", ":=", " ", "42"},
		},

		// Edge cases
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "special characters preserved",
			input:    "@#$?",
			expected: []string{"@", "#", "$", "?"},
		},
		{
			name:     "decorator syntax",
			input:    "@decorator",
			expected: []string{"@", "decorator"},
		},

		// UTF-8 multi-byte characters
		{
			name:     "emoji single character",
			input:    "👋",
			expected: []string{"👋"},
		},
		{
			name:     "emoji in context",
			input:    "hello 👋 world",
			expected: []string{"hello", " ", "👋", " ", "world"},
		},
		{
			name:     "multiple emojis",
			input:    "👋🌍🎉",
			expected: []string{"👋", "🌍", "🎉"},
		},
		{
			name:     "chinese characters",
			input:    "你好",
			expected: []string{"你", "好"},
		},
		{
			name:     "mixed unicode and ascii",
			input:    "café",
			expected: []string{"caf", "é"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := d.Tokenize(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestDiff(t *testing.T) {
	t.Parallel()

	d := worddiff.NewDiffer()

	t.Run("no partial identifier highlighting", func(t *testing.T) {
		t.Parallel()

		// Core improvement: myVariable vs myValue should show entire tokens as changed,
		// not just the differing characters (Va vs lue)
		oldSegs, newSegs := d.Diff("myVariable", "myValue")

		// Both should be single changed segments - no partial highlighting
		assert.Equal(t, []diffview.Segment{{Text: "myVariable", Changed: true}}, oldSegs)
		assert.Equal(t, []diffview.Segment{{Text: "myValue", Changed: true}}, newSegs)
	})

	t.Run("identical strings fast path", func(t *testing.T) {
		t.Parallel()

		oldSegs, newSegs := d.Diff("same text", "same text")

		expected := []diffview.Segment{{Text: "same text", Changed: false}}
		assert.Equal(t, expected, oldSegs)
		assert.Equal(t, expected, newSegs)
	})

	t.Run("both empty strings", func(t *testing.T) {
		t.Parallel()

		oldSegs, newSegs := d.Diff("", "")

		assert.Empty(t, oldSegs)
		assert.Empty(t, newSegs)
	})

	t.Run("old empty new has text", func(t *testing.T) {
		t.Parallel()

		oldSegs, newSegs := d.Diff("", "new text")

		assert.Empty(t, oldSegs)
		assert.Equal(t, []diffview.Segment{{Text: "new text", Changed: true}}, newSegs)
	})

	t.Run("old has text new empty", func(t *testing.T) {
		t.Parallel()

		oldSegs, newSegs := d.Diff("old text", "")

		assert.Equal(t, []diffview.Segment{{Text: "old text", Changed: true}}, oldSegs)
		assert.Empty(t, newSegs)
	})

	t.Run("high similarity keeps word diff", func(t *testing.T) {
		t.Parallel()

		// return x + 1 vs return x + 2 - high similarity, show word diff
		oldSegs, newSegs := d.Diff("return x + 1", "return x + 2")

		// Should show common parts unchanged, only the number changed
		assert.Equal(t, []diffview.Segment{
			{Text: "return x + ", Changed: false},
			{Text: "1", Changed: true},
		}, oldSegs)
		assert.Equal(t, []diffview.Segment{
			{Text: "return x + ", Changed: false},
			{Text: "2", Changed: true},
		}, newSegs)
	})

	t.Run("low similarity returns full replacement", func(t *testing.T) {
		t.Parallel()

		// Completely different lines with no structural similarity
		// "hello world" vs "12345 abcde" - no common tokens at all
		oldSegs, newSegs := d.Diff("hello world", "12345 abcde")

		// Low similarity - everything should be marked changed
		assert.Equal(t, []diffview.Segment{{Text: "hello world", Changed: true}}, oldSegs)
		assert.Equal(t, []diffview.Segment{{Text: "12345 abcde", Changed: true}}, newSegs)
	})

	t.Run("operators as tokens", func(t *testing.T) {
		t.Parallel()

		oldSegs, newSegs := d.Diff("x + y", "x - y")

		// + and - are different operators, x and y and spaces are same
		assert.Equal(t, []diffview.Segment{
			{Text: "x ", Changed: false},
			{Text: "+", Changed: true},
			{Text: " y", Changed: false},
		}, oldSegs)
		assert.Equal(t, []diffview.Segment{
			{Text: "x ", Changed: false},
			{Text: "-", Changed: true},
			{Text: " y", Changed: false},
		}, newSegs)
	})

	t.Run("unicode support", func(t *testing.T) {
		t.Parallel()

		oldSegs, newSegs := d.Diff("hello 👋", "hello 🌍")

		assert.Equal(t, []diffview.Segment{
			{Text: "hello ", Changed: false},
			{Text: "👋", Changed: true},
		}, oldSegs)
		assert.Equal(t, []diffview.Segment{
			{Text: "hello ", Changed: false},
			{Text: "🌍", Changed: true},
		}, newSegs)
	})

	t.Run("another identifier pair getUserName vs getUserEmail", func(t *testing.T) {
		t.Parallel()

		oldSegs, newSegs := d.Diff("getUserName", "getUserEmail")

		// Entire identifiers should be different, not partial
		assert.Equal(t, []diffview.Segment{{Text: "getUserName", Changed: true}}, oldSegs)
		assert.Equal(t, []diffview.Segment{{Text: "getUserEmail", Changed: true}}, newSegs)
	})
}

func BenchmarkDiffer_Diff(b *testing.B) {
	d := worddiff.NewDiffer()

	b.Run("identical", func(b *testing.B) {
		// Fast path: identical strings should skip diffing
		line := "func (s *Server) handleRequest(ctx context.Context, req *Request) (*Response, error)"
		for b.Loop() {
			d.Diff(line, line)
		}
	})

	b.Run("short_similar", func(b *testing.B) {
		// Common case: small change in otherwise similar lines
		oldLine := "return x + 1"
		newLine := "return x + 2"
		for b.Loop() {
			d.Diff(oldLine, newLine)
		}
	})

	b.Run("short_different", func(b *testing.B) {
		// Low similarity: completely different content
		oldLine := "hello world"
		newLine := "12345 abcde"
		for b.Loop() {
			d.Diff(oldLine, newLine)
		}
	})

	b.Run("long_line", func(b *testing.B) {
		// Realistic long code line with minor change
		oldLine := `	result, err := s.repository.FindUserByEmailAndOrganization(ctx, email, orgID, options)`
		newLine := `	result, err := s.repository.FindUserByEmailAndOrganization(ctx, email, orgID, opts)`
		for b.Loop() {
			d.Diff(oldLine, newLine)
		}
	})
}
