package difflib_test

import (
	"testing"

	"github.com/fwojciec/diffview/difflib"
	"github.com/stretchr/testify/assert"
)

func TestTokenize(t *testing.T) {
	t.Parallel()

	d := difflib.NewDiffer()

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := d.Tokenize(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}
