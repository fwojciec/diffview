package bubbletea_test

import (
	"testing"

	"github.com/fwojciec/diffstory/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestExpandTabs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		startCol int
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			startCol: 0,
			expected: "",
		},
		{
			name:     "no tabs",
			input:    "hello world",
			startCol: 0,
			expected: "hello world",
		},
		{
			name:     "single tab at start expands to 8 spaces",
			input:    "\t",
			startCol: 0,
			expected: "        ", // 8 spaces
		},
		{
			name:     "tab after one char expands to 7 spaces",
			input:    "a\t",
			startCol: 0,
			expected: "a       ", // a + 7 spaces = 8 cols
		},
		{
			name:     "tab after seven chars expands to 1 space",
			input:    "1234567\t",
			startCol: 0,
			expected: "1234567 ", // 7 chars + 1 space = 8 cols
		},
		{
			name:     "tab after eight chars expands to 8 spaces",
			input:    "12345678\t",
			startCol: 0,
			expected: "12345678        ", // 8 chars + 8 spaces = 16 cols
		},
		{
			name:     "multiple tabs",
			input:    "\t\t",
			startCol: 0,
			expected: "                ", // 16 spaces
		},
		{
			name:     "mixed content with tabs",
			input:    "abc\tdef",
			startCol: 0,
			expected: "abc     def", // abc + 5 spaces + def = 11 cols
		},
		{
			name:     "typescript style indentation",
			input:    "\t\tconst x = 1;",
			startCol: 0,
			expected: "                const x = 1;", // 16 spaces + code
		},
		{
			name:     "startCol affects first tab expansion",
			input:    "\t",
			startCol: 3,
			expected: "     ", // from col 3 to col 8 = 5 spaces
		},
		{
			name:     "startCol at tab boundary",
			input:    "\t",
			startCol: 8,
			expected: "        ", // from col 8 to col 16 = 8 spaces
		},
		{
			name:     "startCol with text and tab",
			input:    "x\t",
			startCol: 3,
			expected: "x    ", // x at col 3→4, tab expands col 4→8 = 4 spaces
		},
		{
			name:     "unicode character before tab",
			input:    "日\t", // CJK character (width 2) + tab
			startCol: 0,
			expected: "日      ", // col 0→2, tab expands col 2→8 = 6 spaces
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := bubbletea.ExpandTabs(tt.input, tt.startCol)
			assert.Equal(t, tt.expected, result)
		})
	}
}
