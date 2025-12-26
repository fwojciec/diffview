package worddiff

import (
	"strings"
	"unicode/utf8"

	"github.com/fwojciec/diffview"
)

// Differ tokenizes strings and computes word-level diffs.
type Differ struct{}

// NewDiffer creates a new Differ instance.
func NewDiffer() *Differ {
	return &Differ{}
}

// Tokenize splits a string into tokens using a hand-written scanner.
// Token types: identifiers, numbers, string literals, operators, punctuation, whitespace.
func (d *Differ) Tokenize(s string) []string {
	if len(s) == 0 {
		return nil
	}

	// Pre-allocate with estimated capacity (avoid reallocations)
	tokens := make([]string, 0, len(s)/3+1)
	i := 0

	for i < len(s) {
		start := i
		c := s[i]

		switch {
		case isIdentifierStart(c):
			// Identifier: [a-zA-Z_][a-zA-Z0-9_]*
			i++
			for i < len(s) && isIdentifierChar(s[i]) {
				i++
			}
			tokens = append(tokens, s[start:i])

		case isDigit(c):
			// Number: [0-9]+(\.[0-9]+)?
			i++
			for i < len(s) && isDigit(s[i]) {
				i++
			}
			// Check for decimal part
			if i < len(s) && s[i] == '.' && i+1 < len(s) && isDigit(s[i+1]) {
				i++ // consume '.'
				for i < len(s) && isDigit(s[i]) {
					i++
				}
			}
			tokens = append(tokens, s[start:i])

		case c == '"':
			// Double-quoted string literal (handles backslash escapes)
			i++
			for i < len(s) {
				if s[i] == '\\' && i+1 < len(s) {
					i += 2 // skip escaped character
					continue
				}
				if s[i] == '"' {
					i++ // consume closing quote
					break
				}
				i++
			}
			tokens = append(tokens, s[start:i])

		case c == '\'':
			// Single-quoted string literal (handles backslash escapes)
			i++
			for i < len(s) {
				if s[i] == '\\' && i+1 < len(s) {
					i += 2 // skip escaped character
					continue
				}
				if s[i] == '\'' {
					i++ // consume closing quote
					break
				}
				i++
			}
			tokens = append(tokens, s[start:i])

		case isOperatorChar(c):
			// Operator: [+\-*/=<>!&|^%:]+
			i++
			for i < len(s) && isOperatorChar(s[i]) {
				i++
			}
			tokens = append(tokens, s[start:i])

		case isPunctuation(c):
			// Single punctuation character
			i++
			tokens = append(tokens, s[start:i])

		case isWhitespace(c):
			// Whitespace run
			i++
			for i < len(s) && isWhitespace(s[i]) {
				i++
			}
			tokens = append(tokens, s[start:i])

		default:
			// Single character (catch-all for UTF-8 and other chars)
			_, size := utf8.DecodeRuneInString(s[i:])
			i += size
			tokens = append(tokens, s[start:i])
		}
	}

	return tokens
}

func isIdentifierStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isIdentifierChar(c byte) bool {
	return isIdentifierStart(c) || isDigit(c)
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isOperatorChar(c byte) bool {
	switch c {
	case '+', '-', '*', '/', '=', '<', '>', '!', '&', '|', '^', '%', ':':
		return true
	}
	return false
}

func isPunctuation(c byte) bool {
	switch c {
	case '(', ')', '{', '}', '[', ']', ';', ',', '.':
		return true
	}
	return false
}

func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// Compile-time interface verification.
var _ diffview.WordDiffer = (*Differ)(nil)

// similarityThreshold is the minimum ratio for word-level diffing.
// Below this threshold, lines are treated as complete replacements.
const similarityThreshold = 0.4

// Diff returns segments for both the old and new strings,
// marking which portions changed between them.
func (d *Differ) Diff(old, new string) (oldSegs, newSegs []diffview.Segment) {
	// Handle empty strings
	if old == "" && new == "" {
		return nil, nil
	}
	if old == "" {
		return nil, []diffview.Segment{{Text: new, Changed: true}}
	}
	if new == "" {
		return []diffview.Segment{{Text: old, Changed: true}}, nil
	}

	// Fast path for identical strings
	if old == new {
		seg := diffview.Segment{Text: old, Changed: false}
		return []diffview.Segment{seg}, []diffview.Segment{seg}
	}

	oldTokens := d.Tokenize(old)
	newTokens := d.Tokenize(new)

	// Quick similarity check: count common tokens
	if !hasSufficientSimilarity(oldTokens, newTokens) {
		return []diffview.Segment{{Text: old, Changed: true}},
			[]diffview.Segment{{Text: new, Changed: true}}
	}

	// Compute LCS and build pre-merged segments
	return lcsSegments(oldTokens, newTokens)
}

// hasSufficientSimilarity checks if tokens have enough overlap to warrant word-level diff.
// Uses a simple count of common tokens as an upper bound estimate.
func hasSufficientSimilarity(oldTokens, newTokens []string) bool {
	oldLen, newLen := len(oldTokens), len(newTokens)
	if oldLen == 0 || newLen == 0 {
		return false
	}

	// Count tokens in old sequence
	counts := make(map[string]int, oldLen)
	for _, t := range oldTokens {
		counts[t]++
	}

	// Count how many tokens from new exist in old
	common := 0
	for _, t := range newTokens {
		if counts[t] > 0 {
			counts[t]--
			common++
		}
	}

	// Ratio = 2.0 * common / (len(old) + len(new))
	// Check if ratio >= threshold
	total := oldLen + newLen
	return float64(2*common)/float64(total) >= similarityThreshold
}

// lcsSegments computes the LCS of two token sequences and returns merged diff segments.
// Uses O(n×m) dynamic programming with a flat array to minimize allocations.
// Returns pre-merged segments to avoid an extra allocation pass.
func lcsSegments(oldTokens, newTokens []string) (oldSegs, newSegs []diffview.Segment) {
	m, n := len(oldTokens), len(newTokens)

	// Allocate DP table as a flat slice (single allocation)
	// table[i*(n+1)+j] corresponds to table[i][j]
	table := make([]int, (m+1)*(n+1))
	stride := n + 1

	// Fill DP table
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if oldTokens[i-1] == newTokens[j-1] {
				table[i*stride+j] = table[(i-1)*stride+j-1] + 1
			} else if table[(i-1)*stride+j] > table[i*stride+j-1] {
				table[i*stride+j] = table[(i-1)*stride+j]
			} else {
				table[i*stride+j] = table[i*stride+j-1]
			}
		}
	}

	lcsLen := table[m*stride+n]
	if lcsLen == 0 {
		// No common subsequence
		return []diffview.Segment{{Text: joinTokens(oldTokens), Changed: true}},
			[]diffview.Segment{{Text: joinTokens(newTokens), Changed: true}}
	}

	// Backtrack to find matching positions
	type match struct{ oldIdx, newIdx int }
	matches := make([]match, 0, lcsLen)

	i, j := m, n
	for i > 0 && j > 0 {
		if oldTokens[i-1] == newTokens[j-1] {
			matches = append(matches, match{i - 1, j - 1})
			i--
			j--
		} else if table[(i-1)*stride+j] > table[i*stride+j-1] {
			i--
		} else {
			j--
		}
	}

	// Reverse matches (backtracking gives them in reverse order)
	for left, right := 0, len(matches)-1; left < right; left, right = left+1, right-1 {
		matches[left], matches[right] = matches[right], matches[left]
	}

	// Estimate total text length for pre-sizing builders
	oldTotalLen, newTotalLen := 0, 0
	for _, t := range oldTokens {
		oldTotalLen += len(t)
	}
	for _, t := range newTokens {
		newTotalLen += len(t)
	}

	// Build segments directly, merging adjacent same-status segments
	// Track current segment being built
	var oldText, newText strings.Builder
	oldText.Grow(oldTotalLen)
	newText.Grow(newTotalLen)
	oldChanged, newChanged := false, false
	haveOld, haveNew := false, false

	flushOld := func() {
		if haveOld {
			oldSegs = append(oldSegs, diffview.Segment{Text: oldText.String(), Changed: oldChanged})
			oldText.Reset()
			haveOld = false
		}
	}
	flushNew := func() {
		if haveNew {
			newSegs = append(newSegs, diffview.Segment{Text: newText.String(), Changed: newChanged})
			newText.Reset()
			haveNew = false
		}
	}

	addOld := func(text string, changed bool) {
		if haveOld && oldChanged != changed {
			flushOld()
		}
		oldText.WriteString(text)
		oldChanged = changed
		haveOld = true
	}
	addNew := func(text string, changed bool) {
		if haveNew && newChanged != changed {
			flushNew()
		}
		newText.WriteString(text)
		newChanged = changed
		haveNew = true
	}

	oldIdx, newIdx := 0, 0

	for _, mt := range matches {
		// Gap before match = changed
		for oldIdx < mt.oldIdx {
			addOld(oldTokens[oldIdx], true)
			oldIdx++
		}
		for newIdx < mt.newIdx {
			addNew(newTokens[newIdx], true)
			newIdx++
		}

		// Match = unchanged
		addOld(oldTokens[mt.oldIdx], false)
		addNew(newTokens[mt.newIdx], false)
		oldIdx = mt.oldIdx + 1
		newIdx = mt.newIdx + 1
	}

	// Trailing gap
	for oldIdx < m {
		addOld(oldTokens[oldIdx], true)
		oldIdx++
	}
	for newIdx < n {
		addNew(newTokens[newIdx], true)
		newIdx++
	}

	flushOld()
	flushNew()

	return oldSegs, newSegs
}

// joinTokens concatenates tokens using a builder (single allocation for result).
func joinTokens(tokens []string) string {
	if len(tokens) == 0 {
		return ""
	}
	if len(tokens) == 1 {
		return tokens[0]
	}
	var b strings.Builder
	for _, t := range tokens {
		b.WriteString(t)
	}
	return b.String()
}
