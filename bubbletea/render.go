package bubbletea

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/fwojciec/diffview"
)

// hunkKey identifies a specific hunk within a diff.
type hunkKey struct {
	file      string
	hunkIndex int
}

// renderConfig holds all rendering parameters for renderDiff.
type renderConfig struct {
	diff             *diffview.Diff
	styles           diffview.Styles
	renderer         *lipgloss.Renderer
	width            int
	languageDetector diffview.LanguageDetector
	tokenizer        diffview.Tokenizer
	wordDiffer       diffview.WordDiffer

	// Story-aware rendering options (optional)
	collapsedHunks map[hunkKey]bool   // Which hunks are collapsed
	hunkCategories map[hunkKey]string // Category for each hunk (for styling)
	collapseText   map[hunkKey]string // Summary text for collapsed hunks
}

// minGutterWidth is the minimum width of each line number column in the gutter.
const minGutterWidth = 4

// renderDiff converts a Diff to a styled string.
// If renderer is nil, the default lipgloss renderer is used.
// Width is the terminal width for full-width backgrounds.
func renderDiff(cfg renderConfig) string {
	diff := cfg.diff
	styles := cfg.styles
	renderer := cfg.renderer
	width := cfg.width
	if diff == nil {
		return ""
	}

	// Calculate dynamic gutter width based on max line number in the diff
	gutterWidth := calculateGutterWidth(diff)

	// Create lipgloss styles from color pairs
	fileHeaderStyle := styleFromColorPair(styles.FileHeader, renderer)
	hunkHeaderStyle := styleFromColorPair(styles.HunkHeader, renderer)
	addedStyle := styleFromColorPair(styles.Added, renderer)
	deletedStyle := styleFromColorPair(styles.Deleted, renderer)
	contextStyle := styleFromColorPair(styles.Context, renderer)
	lineNumStyle := styleFromColorPair(styles.LineNumber, renderer)
	addedGutterStyle := styleFromColorPair(styles.AddedGutter, renderer)
	deletedGutterStyle := styleFromColorPair(styles.DeletedGutter, renderer)
	addedHighlightStyle := styleFromColorPair(styles.AddedHighlight, renderer)
	deletedHighlightStyle := styleFromColorPair(styles.DeletedHighlight, renderer)

	// Create dimmed style for non-core categories
	dimmedStyle := createDimmedStyle(styles, renderer)

	var sb strings.Builder
	for _, file := range diff.Files {
		// Skip files that shouldn't be rendered (binary files, mode-only changes)
		if !shouldRenderFile(file) {
			continue
		}

		// Detect language for syntax highlighting
		path := filePath(file)
		var language string
		if cfg.languageDetector != nil {
			language = cfg.languageDetector.DetectFromPath(path)
		}

		// Render enhanced file header with box-drawing and change statistics
		// Format: ── filename ─────────────────── +N -M ──
		added, deleted := file.Stats()
		stats := fmt.Sprintf("+%d -%d", added, deleted)

		// Build header: "── " + path + " " + fill + " " + stats + " ──"
		prefix := "── "
		suffix := " ──"
		middle := prefix + path + " "
		end := " " + stats + suffix

		// Calculate fill width
		fillWidth := width - lipgloss.Width(middle) - lipgloss.Width(end)
		if fillWidth < 3 {
			fillWidth = 3
		}
		fill := strings.Repeat("─", fillWidth)

		header := middle + fill + end
		sb.WriteString(fileHeaderStyle.Render(header))
		sb.WriteString("\n")

		// Handle empty files (no hunks)
		if len(file.Hunks) == 0 {
			emptyLine := contextStyle.Render("(empty)")
			sb.WriteString(emptyLine)
			sb.WriteString("\n")
			continue
		}

		for hunkIdx, hunk := range file.Hunks {
			key := hunkKey{file: path, hunkIndex: hunkIdx}

			// Determine if this hunk should be dimmed based on category
			// (must happen before collapsed check so collapsed hunks can be dimmed too)
			isDimmed := false
			if cfg.hunkCategories != nil {
				category := cfg.hunkCategories[key]
				isDimmed = category == "refactoring" || category == "systematic" || category == "noise"
			}

			// Check if this hunk is collapsed
			if cfg.collapsedHunks != nil && cfg.collapsedHunks[key] {
				// Render collapsed hunk as a single line, with dimming if applicable
				collapseStyle := hunkHeaderStyle
				if isDimmed {
					collapseStyle = dimmedStyle
				}
				sb.WriteString(renderCollapsedHunk(hunk, key, cfg, collapseStyle))
				sb.WriteString("\n")
				continue
			}

			// Select styles based on dimming
			currentHunkHeaderStyle := hunkHeaderStyle
			currentAddedStyle := addedStyle
			currentDeletedStyle := deletedStyle
			currentContextStyle := contextStyle
			currentAddedGutterStyle := addedGutterStyle
			currentDeletedGutterStyle := deletedGutterStyle
			currentLineNumStyle := lineNumStyle

			if isDimmed {
				currentHunkHeaderStyle = dimmedStyle
				currentAddedStyle = dimmedStyle
				currentDeletedStyle = dimmedStyle
				currentContextStyle = dimmedStyle
				currentAddedGutterStyle = dimmedStyle
				currentDeletedGutterStyle = dimmedStyle
				currentLineNumStyle = dimmedStyle
			}

			// Render hunk header with styling
			header := formatHunkHeader(hunk)
			sb.WriteString(currentHunkHeaderStyle.Render(header))
			sb.WriteString("\n")

			// Compute word diff segments for paired lines (delete followed by add)
			lineSegments := computeLinePairSegments(hunk.Lines, cfg.wordDiffer)

			// Render lines with gutter and prefixes
			for i, line := range hunk.Lines {
				// Line number gutter with diff-aware styling
				var gutterStyle lipgloss.Style
				var lineStyle lipgloss.Style
				var highlightStyle lipgloss.Style
				switch line.Type {
				case diffview.LineAdded:
					gutterStyle = currentAddedGutterStyle
					lineStyle = currentAddedStyle
					if isDimmed {
						highlightStyle = dimmedStyle
					} else {
						highlightStyle = addedHighlightStyle
					}
				case diffview.LineDeleted:
					gutterStyle = currentDeletedGutterStyle
					lineStyle = currentDeletedStyle
					if isDimmed {
						highlightStyle = dimmedStyle
					} else {
						highlightStyle = deletedHighlightStyle
					}
				default:
					gutterStyle = currentLineNumStyle
					lineStyle = currentContextStyle
				}
				sb.WriteString(formatGutter(line.OldLineNum, line.NewLineNum, gutterWidth, gutterStyle))

				// Add padding space between gutter and code prefix, styled with code line's background
				sb.WriteString(lineStyle.Render(" "))

				// Get prefix and content
				prefix := linePrefixFor(line.Type)
				lineContent := strings.TrimSuffix(line.Content, "\n")
				fullLine := prefix + lineContent

				// Check if this line has word-level diff segments
				segments := lineSegments[i]

				var styledLine string
				if segments != nil {
					// Render with word-level highlighting
					styledLine = renderLineWithSegments(prefix, segments, lineStyle, highlightStyle, width)
				} else {
					// Try to tokenize for syntax highlighting
					var tokens []diffview.Token
					if cfg.tokenizer != nil && language != "" {
						tokens = cfg.tokenizer.Tokenize(language, lineContent)
					}

					if tokens != nil && !isDimmed {
						// Render with syntax highlighting (prefix + tokens)
						// Skip syntax highlighting for dimmed hunks so they appear uniformly muted
						var colors diffview.ColorPair
						switch line.Type {
						case diffview.LineAdded:
							colors = styles.Added
						case diffview.LineDeleted:
							colors = styles.Deleted
						default:
							colors = styles.Context
						}
						styledLine = renderLineWithTokens(prefix, tokens, colors, renderer, width)
					} else {
						// Plain rendering - entire line including prefix
						switch line.Type {
						case diffview.LineAdded:
							styledLine = currentAddedStyle.Render(padLine(fullLine, width))
						case diffview.LineDeleted:
							styledLine = currentDeletedStyle.Render(padLine(fullLine, width))
						default:
							styledLine = currentContextStyle.Render(fullLine)
						}
					}
				}
				sb.WriteString(styledLine)
				sb.WriteString("\n")
			}
		}
	}
	return sb.String()
}

// createDimmedStyle creates a dimmed style for non-core hunks.
func createDimmedStyle(styles diffview.Styles, renderer *lipgloss.Renderer) lipgloss.Style {
	var style lipgloss.Style
	if renderer != nil {
		style = renderer.NewStyle()
	} else {
		style = lipgloss.NewStyle()
	}
	// Use context foreground (muted) for dimmed content
	if styles.Context.Foreground != "" {
		style = style.Foreground(lipgloss.Color(styles.Context.Foreground))
	}
	return style
}

// renderCollapsedHunk renders a collapsed hunk as a single summary line.
// Format: @@ -50,8 +52,10 @@ ▸ [category] collapse text
func renderCollapsedHunk(hunk diffview.Hunk, key hunkKey, cfg renderConfig, headerStyle lipgloss.Style) string {
	// Build the hunk range portion
	rangeStr := fmt.Sprintf("@@ -%d,%d +%d,%d @@", hunk.OldStart, hunk.OldCount, hunk.NewStart, hunk.NewCount)

	// Get collapse text, defaulting to a generic message
	collapseText := "collapsed"
	if cfg.collapseText != nil {
		if text, ok := cfg.collapseText[key]; ok && text != "" {
			collapseText = text
		}
	}

	// Get category for display
	category := ""
	if cfg.hunkCategories != nil {
		category = cfg.hunkCategories[key]
	}

	// Build summary: "▸ [category] collapse text" or "▸ collapse text"
	var summary string
	if category != "" {
		summary = fmt.Sprintf("▸ [%s] %s", category, collapseText)
	} else {
		summary = fmt.Sprintf("▸ %s", collapseText)
	}

	return headerStyle.Render(rangeStr + " " + summary)
}

// computeLinePairSegments identifies paired delete/add lines and computes word-level diff segments.
// Returns a map from line index to segments. Lines without word-level diffs have nil segments.
// Only applies word-level highlighting when there's meaningful shared content (>30% unchanged).
//
// Handles both simple pairs (one delete followed by one add) and runs of consecutive
// deletes followed by consecutive adds (pairs them 1:1 in order).
func computeLinePairSegments(lines []diffview.Line, wordDiffer diffview.WordDiffer) map[int][]diffview.Segment {
	if wordDiffer == nil {
		return nil
	}

	result := make(map[int][]diffview.Segment)

	// Find runs of consecutive deleted lines followed by runs of added lines
	for i := 0; i < len(lines); i++ {
		if lines[i].Type != diffview.LineDeleted {
			continue
		}

		// Found start of a delete run - count consecutive deletes
		deleteStart := i
		deleteEnd := i
		for deleteEnd < len(lines) && lines[deleteEnd].Type == diffview.LineDeleted {
			deleteEnd++
		}

		// Check if immediately followed by added lines
		if deleteEnd >= len(lines) || lines[deleteEnd].Type != diffview.LineAdded {
			i = deleteEnd - 1 // Skip to end of delete run
			continue
		}

		// Count consecutive adds
		addStart := deleteEnd
		addEnd := addStart
		for addEnd < len(lines) && lines[addEnd].Type == diffview.LineAdded {
			addEnd++
		}

		// Pair up deletes and adds 1:1
		deleteCount := deleteEnd - deleteStart
		addCount := addEnd - addStart
		pairCount := deleteCount
		if addCount < pairCount {
			pairCount = addCount
		}

		for j := 0; j < pairCount; j++ {
			delIdx := deleteStart + j
			addIdx := addStart + j

			oldContent := strings.TrimSuffix(lines[delIdx].Content, "\n")
			newContent := strings.TrimSuffix(lines[addIdx].Content, "\n")
			oldSegs, newSegs := wordDiffer.Diff(oldContent, newContent)

			// Only use word-level highlighting if there's meaningful shared content.
			if hasSignificantUnchangedContent(oldSegs) && hasSignificantUnchangedContent(newSegs) {
				result[delIdx] = oldSegs
				result[addIdx] = newSegs
			}
		}

		i = addEnd - 1 // Skip to end of add run
	}

	return result
}

// hasSignificantUnchangedContent checks if segments have enough unchanged content
// to make word-level highlighting useful (at least 30% unchanged).
func hasSignificantUnchangedContent(segments []diffview.Segment) bool {
	if len(segments) == 0 {
		return false
	}

	var unchangedLen, totalLen int
	for _, seg := range segments {
		segLen := len(seg.Text)
		totalLen += segLen
		if !seg.Changed {
			unchangedLen += segLen
		}
	}

	if totalLen == 0 {
		return false
	}

	// Require at least 30% unchanged content for word-level diff to be useful
	return float64(unchangedLen)/float64(totalLen) >= 0.30
}

// renderLineWithSegments renders a line with word-level diff highlighting.
// Unchanged segments use baseStyle, changed segments use highlightStyle.
func renderLineWithSegments(prefix string, segments []diffview.Segment, baseStyle, highlightStyle lipgloss.Style, width int) string {
	var sb strings.Builder

	// Render prefix with base style
	sb.WriteString(baseStyle.Render(prefix))

	// Render each segment with appropriate style
	for _, seg := range segments {
		if seg.Changed {
			sb.WriteString(highlightStyle.Render(seg.Text))
		} else {
			sb.WriteString(baseStyle.Render(seg.Text))
		}
	}

	// Calculate current length and pad if needed
	currentLen := lipgloss.Width(prefix)
	for _, seg := range segments {
		currentLen += lipgloss.Width(seg.Text)
	}

	if currentLen < width {
		padding := strings.Repeat(" ", width-currentLen)
		sb.WriteString(baseStyle.Render(padding))
	}

	return sb.String()
}

// renderLineWithTokens renders a line with syntax highlighting.
// Each token gets its syntax foreground color combined with the diff background.
func renderLineWithTokens(prefix string, tokens []diffview.Token, colors diffview.ColorPair, renderer *lipgloss.Renderer, width int) string {
	var sb strings.Builder

	// Helper to create a new style with the renderer
	newStyle := func() lipgloss.Style {
		if renderer != nil {
			return renderer.NewStyle()
		}
		return lipgloss.NewStyle()
	}

	// Create base style with diff colors
	baseStyle := newStyle()
	if colors.Foreground != "" {
		baseStyle = baseStyle.Foreground(lipgloss.Color(colors.Foreground))
	}
	if colors.Background != "" {
		baseStyle = baseStyle.Background(lipgloss.Color(colors.Background))
	}

	// Render prefix with base style
	sb.WriteString(baseStyle.Render(prefix))

	// Render each token with syntax foreground + diff background
	for _, tok := range tokens {
		// Build style from scratch for each token
		style := newStyle()

		// Always apply diff background
		if colors.Background != "" {
			style = style.Background(lipgloss.Color(colors.Background))
		}

		// Use syntax foreground if provided, otherwise use diff foreground
		if tok.Style.Foreground != "" {
			style = style.Foreground(lipgloss.Color(tok.Style.Foreground))
		} else if colors.Foreground != "" {
			style = style.Foreground(lipgloss.Color(colors.Foreground))
		}

		// Apply bold if specified by syntax
		if tok.Style.Bold {
			style = style.Bold(true)
		}

		sb.WriteString(style.Render(tok.Text))
	}

	// Calculate current length and pad if needed
	currentLen := lipgloss.Width(prefix)
	for _, tok := range tokens {
		currentLen += lipgloss.Width(tok.Text)
	}

	if currentLen < width {
		padding := strings.Repeat(" ", width-currentLen)
		sb.WriteString(baseStyle.Render(padding))
	}

	return sb.String()
}

// calculateGutterWidth determines the appropriate gutter width for a diff
// based on the maximum line number present in any hunk.
func calculateGutterWidth(diff *diffview.Diff) int {
	maxLineNum := 0
	for _, file := range diff.Files {
		for _, hunk := range file.Hunks {
			for _, line := range hunk.Lines {
				if line.OldLineNum > maxLineNum {
					maxLineNum = line.OldLineNum
				}
				if line.NewLineNum > maxLineNum {
					maxLineNum = line.NewLineNum
				}
			}
		}
	}
	width := digitWidth(maxLineNum)
	if width < minGutterWidth {
		return minGutterWidth
	}
	return width
}

// formatGutter formats the gutter column with old and new line numbers.
// Format: "  12    14 " for lines with both numbers
// Format: "  12       " for deleted lines (no new line number - empty space)
// Format: "       14 " for added lines (no old line number - empty space)
// No divider character - the color transition provides visual separation.
func formatGutter(oldLineNum, newLineNum, width int, style lipgloss.Style) string {
	oldStr := formatLineNum(oldLineNum, width)
	newStr := formatLineNum(newLineNum, width)
	gutter := fmt.Sprintf("%s %s ", oldStr, newStr)
	return style.Render(gutter)
}

// formatLineNum formats a line number for the gutter.
// Returns right-aligned number or empty space for zero (missing) line numbers.
func formatLineNum(num, width int) string {
	if num == 0 {
		return fmt.Sprintf("%*s", width, "")
	}
	return fmt.Sprintf("%*d", width, num)
}

// styleFromColorPair creates a lipgloss style from a ColorPair.
// If renderer is nil, the default lipgloss renderer is used.
func styleFromColorPair(cp diffview.ColorPair, renderer *lipgloss.Renderer) lipgloss.Style {
	var style lipgloss.Style
	if renderer != nil {
		style = renderer.NewStyle()
	} else {
		style = lipgloss.NewStyle()
	}
	if cp.Foreground != "" {
		style = style.Foreground(lipgloss.Color(cp.Foreground))
	}
	if cp.Background != "" {
		style = style.Background(lipgloss.Color(cp.Background))
	}
	return style
}

// formatHunkHeader formats a hunk header in standard diff format.
func formatHunkHeader(hunk diffview.Hunk) string {
	header := fmt.Sprintf("@@ -%d,%d +%d,%d @@", hunk.OldStart, hunk.OldCount, hunk.NewStart, hunk.NewCount)
	if hunk.Section != "" {
		header += " " + hunk.Section
	}
	return header
}

// linePrefixFor returns the appropriate prefix for a line type.
func linePrefixFor(lineType diffview.LineType) string {
	switch lineType {
	case diffview.LineAdded:
		return "+"
	case diffview.LineDeleted:
		return "-"
	default:
		return " "
	}
}

// padLine pads a line with spaces to the specified display width.
// Uses lipgloss.Width() to correctly handle multi-byte Unicode characters.
// If the line is already wider, it is returned unchanged.
func padLine(line string, width int) string {
	lineWidth := lipgloss.Width(line)
	if lineWidth >= width {
		return line
	}
	return line + strings.Repeat(" ", width-lineWidth)
}

// shouldRenderFile returns true if the file should be rendered in the diff view.
// Binary files are skipped, but empty text files (new or deleted) are shown.
func shouldRenderFile(file diffview.FileDiff) bool {
	// Always skip binary files
	if file.IsBinary {
		return false
	}
	// Render files with hunks
	if len(file.Hunks) > 0 {
		return true
	}
	// Render empty new/deleted files
	if file.Operation == diffview.FileAdded || file.Operation == diffview.FileDeleted {
		return true
	}
	// Render renames/copies (even without content changes)
	if file.Operation == diffview.FileRenamed || file.Operation == diffview.FileCopied {
		return true
	}
	// Skip mode-only changes without hunks (or add logic to show them later)
	return false
}

// filePath returns the display path for a file in the diff.
// Uses NewPath for most operations, OldPath for deleted files.
func filePath(file diffview.FileDiff) string {
	var path string
	if file.Operation == diffview.FileDeleted {
		path = file.OldPath
	} else {
		path = file.NewPath
	}
	// Strip "a/" or "b/" prefix if present
	path = strings.TrimPrefix(path, "a/")
	path = strings.TrimPrefix(path, "b/")
	return path
}

// digitWidth returns the number of digits needed to display n.
func digitWidth(n int) int {
	if n <= 0 {
		return 1
	}
	width := 0
	for n > 0 {
		width++
		n /= 10
	}
	return width
}

// computePositions calculates the line numbers where each hunk and file starts.
// This is independent of terminal width and can be computed eagerly.
func computePositions(diff *diffview.Diff) (hunkPositions, filePositions []int) {
	if diff == nil {
		return nil, nil
	}

	lineNum := 0
	for _, file := range diff.Files {
		// Skip files that shouldn't be rendered (binary files, mode-only changes)
		if !shouldRenderFile(file) {
			continue
		}

		// Track file position at the header line
		filePositions = append(filePositions, lineNum)

		// Enhanced file header (single line: ── file ─── +N -M ──)
		lineNum++

		if len(file.Hunks) == 0 {
			// Empty file: one line for "(empty)" indicator
			lineNum++
		} else {
			for _, hunk := range file.Hunks {
				// Track hunk position at the header line
				hunkPositions = append(hunkPositions, lineNum)

				// Hunk header
				lineNum++

				// Content lines
				lineNum += len(hunk.Lines)
			}
		}
	}
	return hunkPositions, filePositions
}
