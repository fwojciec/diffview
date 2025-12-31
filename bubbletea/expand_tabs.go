package bubbletea

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ExpandTabs converts tab characters to the appropriate number of spaces
// based on standard 8-column tab stops. The startCol parameter indicates
// the column position where the string begins, which affects how the first
// tab is expanded.
func ExpandTabs(s string, startCol int) string {
	if !strings.Contains(s, "\t") {
		return s
	}

	var sb strings.Builder
	col := startCol
	for _, r := range s {
		if r == '\t' {
			nextStop := ((col / tabWidth) + 1) * tabWidth
			spaces := nextStop - col
			sb.WriteString(strings.Repeat(" ", spaces))
			col = nextStop
		} else {
			sb.WriteRune(r)
			col += lipgloss.Width(string(r))
		}
	}
	return sb.String()
}
