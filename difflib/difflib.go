package difflib

import (
	"regexp"
	"strings"

	"github.com/fwojciec/diffview"
	"github.com/pmezard/go-difflib/difflib"
)

// Differ tokenizes strings and computes word-level diffs.
type Differ struct {
	tokenPattern *regexp.Regexp
}

// NewDiffer creates a new Differ instance.
func NewDiffer() *Differ {
	return &Differ{
		tokenPattern: regexp.MustCompile(
			`[a-zA-Z_][a-zA-Z0-9_]*|` + // identifiers
				`[0-9]+\.[0-9]+|[0-9]+|` + // numbers (float or integer)
				`"[^"]*"|'[^']*'|` + // string literals
				`[+\-*/=<>!&|^%:]+|` + // operators (including :)
				`[(){}\[\];,.]|` + // punctuation
				`\s+|` + // whitespace
				`.`, // catch-all for any remaining character
		),
	}
}

// Tokenize splits a string into tokens.
func (d *Differ) Tokenize(s string) []string {
	return d.tokenPattern.FindAllString(s, -1)
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

	matcher := difflib.NewMatcher(oldTokens, newTokens)

	// Use QuickRatio for threshold check - it's an upper bound estimate.
	// If QuickRatio < threshold, actual ratio is guaranteed to be below too.
	if matcher.QuickRatio() < similarityThreshold {
		return []diffview.Segment{{Text: old, Changed: true}},
			[]diffview.Segment{{Text: new, Changed: true}}
	}

	blocks := matcher.GetMatchingBlocks()

	oldSegs, newSegs = buildSegments(oldTokens, newTokens, blocks)
	return mergeSegments(oldSegs), mergeSegments(newSegs)
}

// buildSegments creates segments from matching blocks.
func buildSegments(oldTokens, newTokens []string, blocks []difflib.Match) (oldSegs, newSegs []diffview.Segment) {
	oldIdx, newIdx := 0, 0

	for _, block := range blocks {
		// Gap before match = changed
		if oldIdx < block.A {
			oldSegs = append(oldSegs, diffview.Segment{
				Text:    strings.Join(oldTokens[oldIdx:block.A], ""),
				Changed: true,
			})
		}
		if newIdx < block.B {
			newSegs = append(newSegs, diffview.Segment{
				Text:    strings.Join(newTokens[newIdx:block.B], ""),
				Changed: true,
			})
		}

		// Match = unchanged
		if block.Size > 0 {
			text := strings.Join(oldTokens[block.A:block.A+block.Size], "")
			oldSegs = append(oldSegs, diffview.Segment{Text: text, Changed: false})
			newSegs = append(newSegs, diffview.Segment{Text: text, Changed: false})
		}

		oldIdx = block.A + block.Size
		newIdx = block.B + block.Size
	}

	return oldSegs, newSegs
}

// mergeSegments combines adjacent segments with the same Changed status.
func mergeSegments(segments []diffview.Segment) []diffview.Segment {
	if len(segments) == 0 {
		return nil
	}

	merged := make([]diffview.Segment, 0, len(segments))
	current := segments[0]

	for i := 1; i < len(segments); i++ {
		if segments[i].Changed == current.Changed {
			current.Text += segments[i].Text
		} else {
			merged = append(merged, current)
			current = segments[i]
		}
	}
	merged = append(merged, current)

	return merged
}
