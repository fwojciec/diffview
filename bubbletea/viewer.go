// Package bubbletea provides a terminal UI viewer for diffs using the Bubble Tea framework.
package bubbletea

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fwojciec/diffview"
)

// Compile-time interface verification.
var _ diffview.Viewer = (*Viewer)(nil)

// Model is the Bubble Tea model for viewing diffs.
type Model struct {
	diff             *diffview.Diff
	styles           diffview.Styles
	palette          diffview.Palette
	renderer         *lipgloss.Renderer
	languageDetector diffview.LanguageDetector
	tokenizer        diffview.Tokenizer
	wordDiffer       diffview.WordDiffer
	viewport         viewport.Model
	ready            bool
	keymap           KeyMap
	pendingKey       string
	hunkPositions    []int // line numbers where each hunk starts
	filePositions    []int // line numbers where each file starts
	width            int   // terminal width for rendering
}

// ModelOption configures a Model.
type ModelOption func(*modelConfig)

type modelConfig struct {
	renderer         *lipgloss.Renderer
	theme            diffview.Theme
	languageDetector diffview.LanguageDetector
	tokenizer        diffview.Tokenizer
	wordDiffer       diffview.WordDiffer
}

// WithRenderer sets a custom lipgloss renderer for the model.
// This is primarily useful for testing to force specific color output.
func WithRenderer(r *lipgloss.Renderer) ModelOption {
	return func(cfg *modelConfig) {
		cfg.renderer = r
	}
}

// WithTheme sets the theme for the model.
// If nil is passed, the model uses default styles and palette.
func WithTheme(t diffview.Theme) ModelOption {
	return func(cfg *modelConfig) {
		cfg.theme = t
	}
}

// WithLanguageDetector sets the language detector for syntax highlighting.
func WithLanguageDetector(d diffview.LanguageDetector) ModelOption {
	return func(cfg *modelConfig) {
		cfg.languageDetector = d
	}
}

// WithTokenizer sets the tokenizer for syntax highlighting.
func WithTokenizer(t diffview.Tokenizer) ModelOption {
	return func(cfg *modelConfig) {
		cfg.tokenizer = t
	}
}

// WithWordDiffer sets the word differ for word-level highlighting.
func WithWordDiffer(d diffview.WordDiffer) ModelOption {
	return func(cfg *modelConfig) {
		cfg.wordDiffer = d
	}
}

// NewModel creates a new Model with the given diff.
// Use WithTheme to set a custom theme, otherwise uses hardcoded defaults.
func NewModel(diff *diffview.Diff, opts ...ModelOption) Model {
	cfg := &modelConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	var styles diffview.Styles
	var palette diffview.Palette
	if cfg.theme != nil {
		styles = cfg.theme.Styles()
		palette = cfg.theme.Palette()
	} else {
		styles = defaultStyles()
		palette = defaultPalette()
	}

	// Compute positions eagerly - they don't depend on terminal width
	hunkPositions, filePositions := computePositions(diff)

	return Model{
		diff:             diff,
		styles:           styles,
		palette:          palette,
		renderer:         cfg.renderer,
		languageDetector: cfg.languageDetector,
		tokenizer:        cfg.tokenizer,
		wordDiffer:       cfg.wordDiffer,
		keymap:           DefaultKeyMap(),
		hunkPositions:    hunkPositions,
		filePositions:    filePositions,
	}
}

// defaultStyles returns the default dark theme styles (GitHub-inspired).
// These values should match what stylesFromPalette(githubDarkPalette()) produces.
func defaultStyles() diffview.Styles {
	return diffview.Styles{
		Added: diffview.ColorPair{
			Foreground: "#e6edf3", // Normal text (neutral)
			Background: "#142a1f", // Subtle green background (15% blend of #3fb950 with #0d1117)
		},
		Deleted: diffview.ColorPair{
			Foreground: "#e6edf3", // Normal text (neutral)
			Background: "#301a1e", // Subtle red background (15% blend of #f85149 with #0d1117)
		},
		Context: diffview.ColorPair{
			Foreground: "#8b949e", // Muted foreground (Context color)
		},
		HunkHeader: diffview.ColorPair{
			Foreground: "#58a6ff", // Blue accent (UIAccent)
		},
		FileHeader: diffview.ColorPair{
			Foreground: "#d29922", // Modified/warning yellow
			Background: "#161b22", // Elevated surface (UIBackground)
		},
		FileSeparator: diffview.ColorPair{
			Foreground: "#8b949e", // Muted text (UIForeground)
		},
		LineNumber: diffview.ColorPair{
			Foreground: "#8b949e", // Muted foreground (Context color)
		},
		AddedGutter: diffview.ColorPair{
			Foreground: "#e6edf3", // Same as code line foreground
			Background: "#1e4b2a", // Stronger green background (35% blend of #3fb950 with #0d1117)
		},
		DeletedGutter: diffview.ColorPair{
			Foreground: "#e6edf3", // Same as code line foreground
			Background: "#5f2728", // Stronger red background (35% blend of #f85149 with #0d1117)
		},
		AddedHighlight: diffview.ColorPair{
			Foreground: "#e6edf3", // Same as code line foreground (neutral)
			Background: "#1e4b2a", // Same as gutter (35% blend)
		},
		DeletedHighlight: diffview.ColorPair{
			Foreground: "#e6edf3", // Same as code line foreground (neutral)
			Background: "#5f2728", // Same as gutter (35% blend)
		},
	}
}

// defaultPalette returns the default palette (GitHub-inspired dark).
func defaultPalette() diffview.Palette {
	return diffview.Palette{
		// Base colors - GitHub dark mode canvas
		Background: "#0d1117",
		Foreground: "#e6edf3",

		// Diff colors - GitHub success/danger semantic colors
		Added:    "#3fb950",
		Deleted:  "#f85149",
		Modified: "#d29922",
		Context:  "#8b949e",

		// Syntax highlighting colors - GitHub dark mode syntax
		Keyword:     "#ff7b72",
		String:      "#a5d6ff",
		Number:      "#79c0ff",
		Comment:     "#8b949e",
		Operator:    "#ff7b72",
		Function:    "#d2a8ff",
		Type:        "#ffa657",
		Constant:    "#79c0ff",
		Punctuation: "#8b949e",

		// UI colors - GitHub dark mode surfaces
		UIBackground: "#161b22",
		UIForeground: "#8b949e",
		UIAccent:     "#58a6ff",
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// newStyle creates a new lipgloss style using the model's renderer.
// This ensures consistent color output in both production and tests.
func (m Model) newStyle() lipgloss.Style {
	if m.renderer != nil {
		return m.renderer.NewStyle()
	}
	return lipgloss.NewStyle()
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle multi-key sequences (gg for go to top)
		if m.pendingKey == "g" && key.Matches(msg, m.keymap.GotoTop) {
			m.viewport.GotoTop()
			m.pendingKey = ""
			return m, nil
		}

		// Check for start of multi-key sequence
		if key.Matches(msg, m.keymap.GotoTop) {
			m.pendingKey = "g"
			return m, nil
		}

		// Clear pending key on any other key press
		m.pendingKey = ""

		switch {
		case key.Matches(msg, m.keymap.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keymap.GotoBottom):
			m.viewport.GotoBottom()
			return m, nil
		case key.Matches(msg, m.keymap.HalfPageUp):
			m.viewport.HalfPageUp()
			return m, nil
		case key.Matches(msg, m.keymap.HalfPageDown):
			m.viewport.HalfPageDown()
			return m, nil
		case key.Matches(msg, m.keymap.Up):
			m.viewport.ScrollUp(1)
			return m, nil
		case key.Matches(msg, m.keymap.Down):
			m.viewport.ScrollDown(1)
			return m, nil
		case key.Matches(msg, m.keymap.NextHunk):
			m.gotoNextPosition(m.hunkPositions)
			return m, nil
		case key.Matches(msg, m.keymap.PrevHunk):
			m.gotoPrevPosition(m.hunkPositions)
			return m, nil
		case key.Matches(msg, m.keymap.NextFile):
			m.gotoNextPosition(m.filePositions)
			return m, nil
		case key.Matches(msg, m.keymap.PrevFile):
			m.gotoPrevPosition(m.filePositions)
			return m, nil
		}
	case tea.WindowSizeMsg:
		statusBarHeight := 1
		widthChanged := m.width != msg.Width
		m.width = msg.Width

		if !m.ready {
			// First render - create viewport and render content
			m.viewport = viewport.New(msg.Width, msg.Height-statusBarHeight)
			m.viewport.SetContent(m.renderContent())
			m.ready = true
		} else if widthChanged {
			// Width changed - re-render content
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - statusBarHeight
			m.viewport.SetContent(m.renderContent())
		} else {
			// Only height changed
			m.viewport.Height = msg.Height - statusBarHeight
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View implements tea.Model.
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}
	return lipgloss.JoinVertical(lipgloss.Left, m.viewport.View(), m.statusBarView())
}

// renderContent renders the diff content with current model configuration.
func (m Model) renderContent() string {
	return renderDiff(renderConfig{
		diff:             m.diff,
		styles:           m.styles,
		renderer:         m.renderer,
		width:            m.width,
		languageDetector: m.languageDetector,
		tokenizer:        m.tokenizer,
		wordDiffer:       m.wordDiffer,
	})
}

// statusBarView renders the status bar with position info.
func (m Model) statusBarView() string {
	// Create styles using palette colors and renderer
	barStyle := m.newStyle().
		Background(lipgloss.Color(m.palette.UIBackground)).
		Foreground(lipgloss.Color(m.palette.Foreground))

	dimStyle := m.newStyle().
		Background(lipgloss.Color(m.palette.UIBackground)).
		Foreground(lipgloss.Color(m.palette.Context))

	sepStyle := m.newStyle().
		Background(lipgloss.Color(m.palette.UIBackground)).
		Foreground(lipgloss.Color(m.palette.UIForeground))

	// Format position info with fixed widths
	fileIdx, fileTotal := m.currentFilePosition()
	hunkIdx, hunkTotal := m.currentHunkPosition()

	// Calculate digit widths for consistent formatting
	fileWidth := digitWidth(fileTotal)
	hunkWidth := digitWidth(hunkTotal)

	filePos := fmt.Sprintf("file %*d/%-*d", fileWidth, fileIdx, fileWidth, fileTotal)
	hunkPos := fmt.Sprintf("hunk %*d/%-*d", hunkWidth, hunkIdx, hunkWidth, hunkTotal)
	scrollPos := m.scrollPosition()

	// Build status bar with separators
	sep := sepStyle.Render(" │ ")
	content := barStyle.Render(filePos) + sep +
		barStyle.Render(hunkPos) + sep +
		barStyle.Render(scrollPos) + sep +
		dimStyle.Render("j/k:scroll  n/N:hunk  ]/[:file  q:quit") +
		barStyle.Render("  ") // Right padding

	// Right-align by padding left side with background
	contentWidth := lipgloss.Width(content)
	if m.width > contentWidth {
		padding := barStyle.Render(strings.Repeat(" ", m.width-contentWidth))
		content = padding + content
	}

	return content
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

// scrollPosition returns a string indicating the scroll position (always 3 chars).
func (m Model) scrollPosition() string {
	if m.viewport.AtTop() {
		return "Top"
	}
	if m.viewport.AtBottom() {
		return "Bot"
	}
	// Calculate percentage, format to 3 chars (e.g., " 4%", "50%")
	percent := int(m.viewport.ScrollPercent() * 100)
	return fmt.Sprintf("%2d%%", percent)
}

// currentFilePosition returns the current file index (1-based) and total file count.
func (m Model) currentFilePosition() (current, total int) {
	total = len(m.filePositions)
	if total == 0 {
		return 0, 0
	}

	currentLine := m.viewport.YOffset
	current = 1 // Default to first file

	// Find which file we're currently in
	for i, pos := range m.filePositions {
		if pos <= currentLine {
			current = i + 1 // 1-based index
		} else {
			break
		}
	}

	return current, total
}

// currentHunkPosition returns the current hunk index (1-based) and total hunk count.
func (m Model) currentHunkPosition() (current, total int) {
	total = len(m.hunkPositions)
	if total == 0 {
		return 0, 0
	}

	currentLine := m.viewport.YOffset
	current = 1 // Default to first hunk

	// Find which hunk we're currently in
	for i, pos := range m.hunkPositions {
		if pos <= currentLine {
			current = i + 1 // 1-based index
		} else {
			break
		}
	}

	return current, total
}

// HunkPositions returns the line numbers where each hunk starts.
func (m Model) HunkPositions() []int {
	return m.hunkPositions
}

// FilePositions returns the line numbers where each file starts.
func (m Model) FilePositions() []int {
	return m.filePositions
}

// gotoNextPosition scrolls to the next position.
// It finds the current position (first one >= currentLine) and navigates to the next.
func (m *Model) gotoNextPosition(positions []int) {
	if len(positions) == 0 {
		return
	}
	currentLine := m.viewport.YOffset
	// Find index of current position (first one >= currentLine)
	currentIdx := -1
	for i, pos := range positions {
		if pos >= currentLine {
			currentIdx = i
			break
		}
	}
	// If no position >= currentLine, we're past all positions
	if currentIdx == -1 {
		return
	}
	// Navigate to next position if it exists
	nextIdx := currentIdx + 1
	if nextIdx < len(positions) {
		m.viewport.SetYOffset(positions[nextIdx])
	}
}

// gotoPrevPosition scrolls to the previous position.
// It finds the current position (first one >= currentLine) and navigates to the previous.
func (m *Model) gotoPrevPosition(positions []int) {
	if len(positions) == 0 {
		return
	}
	currentLine := m.viewport.YOffset
	// Find index of current position (first one >= currentLine)
	currentIdx := -1
	for i, pos := range positions {
		if pos >= currentLine {
			currentIdx = i
			break
		}
	}
	// If no position >= currentLine, we're past all positions, go to last
	if currentIdx == -1 {
		m.viewport.SetYOffset(positions[len(positions)-1])
		return
	}
	// Navigate to previous position if it exists
	prevIdx := currentIdx - 1
	if prevIdx >= 0 {
		m.viewport.SetYOffset(positions[prevIdx])
	}
}

// minGutterWidth is the minimum width of each line number column in the gutter.
const minGutterWidth = 4

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

// renderConfig holds all rendering parameters for renderDiff.
type renderConfig struct {
	diff             *diffview.Diff
	styles           diffview.Styles
	renderer         *lipgloss.Renderer
	width            int
	languageDetector diffview.LanguageDetector
	tokenizer        diffview.Tokenizer
	wordDiffer       diffview.WordDiffer
}

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

		for _, hunk := range file.Hunks {
			// Render hunk header with styling
			header := formatHunkHeader(hunk)
			sb.WriteString(hunkHeaderStyle.Render(header))
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
					gutterStyle = addedGutterStyle
					lineStyle = addedStyle
					highlightStyle = addedHighlightStyle
				case diffview.LineDeleted:
					gutterStyle = deletedGutterStyle
					lineStyle = deletedStyle
					highlightStyle = deletedHighlightStyle
				default:
					gutterStyle = lineNumStyle
					lineStyle = contextStyle
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

					if tokens != nil {
						// Render with syntax highlighting (prefix + tokens)
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
							styledLine = addedStyle.Render(padLine(fullLine, width))
						case diffview.LineDeleted:
							styledLine = deletedStyle.Render(padLine(fullLine, width))
						default:
							styledLine = contextStyle.Render(fullLine)
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

// Viewer implements diffview.Viewer using a Bubble Tea TUI.
type Viewer struct {
	theme            diffview.Theme
	languageDetector diffview.LanguageDetector
	tokenizer        diffview.Tokenizer
	wordDiffer       diffview.WordDiffer
	programOpts      []tea.ProgramOption
}

// ViewerOption configures a Viewer.
type ViewerOption func(*Viewer)

// WithProgramOptions adds additional tea.ProgramOption to the viewer.
// This is primarily useful for testing.
func WithProgramOptions(opts ...tea.ProgramOption) ViewerOption {
	return func(v *Viewer) {
		v.programOpts = append(v.programOpts, opts...)
	}
}

// WithViewerLanguageDetector sets the language detector for syntax highlighting.
func WithViewerLanguageDetector(d diffview.LanguageDetector) ViewerOption {
	return func(v *Viewer) {
		v.languageDetector = d
	}
}

// WithViewerTokenizer sets the tokenizer for syntax highlighting.
func WithViewerTokenizer(t diffview.Tokenizer) ViewerOption {
	return func(v *Viewer) {
		v.tokenizer = t
	}
}

// WithViewerWordDiffer sets the word differ for word-level highlighting.
func WithViewerWordDiffer(d diffview.WordDiffer) ViewerOption {
	return func(v *Viewer) {
		v.wordDiffer = d
	}
}

// NewViewer creates a new Viewer with the given theme.
func NewViewer(theme diffview.Theme, opts ...ViewerOption) *Viewer {
	v := &Viewer{theme: theme}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// View displays the diff and blocks until the user exits.
func (v *Viewer) View(ctx context.Context, diff *diffview.Diff) error {
	m := NewModel(diff,
		WithTheme(v.theme),
		WithLanguageDetector(v.languageDetector),
		WithTokenizer(v.tokenizer),
		WithWordDiffer(v.wordDiffer),
	)
	opts := []tea.ProgramOption{
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithContext(ctx),
	}
	opts = append(opts, v.programOpts...)
	p := tea.NewProgram(m, opts...)
	_, err := p.Run()
	return err
}
