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
	"github.com/fwojciec/diffview/worddiff"
)

// Model is the Bubble Tea model for viewing diffs.
type Model struct {
	diff          *diffview.Diff
	viewport      viewport.Model
	ready         bool
	content       string
	keymap        KeyMap
	pendingKey    string
	hunkPositions []int // line numbers where each hunk starts
	filePositions []int // line numbers where each file starts
	width         int   // terminal width for status bar
}

// ModelOption configures a Model.
type ModelOption func(*modelConfig)

type modelConfig struct {
	renderer *lipgloss.Renderer
}

// WithRenderer sets a custom lipgloss renderer for the model.
// This is primarily useful for testing to force specific color output.
func WithRenderer(r *lipgloss.Renderer) ModelOption {
	return func(cfg *modelConfig) {
		cfg.renderer = r
	}
}

// NewModel creates a new Model with the given diff and default dark theme.
func NewModel(diff *diffview.Diff, opts ...ModelOption) Model {
	return NewModelWithStyles(diff, defaultStyles(), opts...)
}

// NewModelWithStyles creates a new Model with the given diff and styles.
func NewModelWithStyles(diff *diffview.Diff, styles diffview.Styles, opts ...ModelOption) Model {
	cfg := &modelConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	content, hunkPositions, filePositions := renderDiffWithPositions(diff, styles, cfg.renderer)
	return Model{
		diff:          diff,
		content:       content,
		keymap:        DefaultKeyMap(),
		hunkPositions: hunkPositions,
		filePositions: filePositions,
	}
}

// defaultStyles returns the default dark theme styles.
func defaultStyles() diffview.Styles {
	return diffview.Styles{
		Added: diffview.ColorPair{
			Foreground: "#a6e3a1", // Green
			Background: "#2d3f2d", // Subtle green background
		},
		Deleted: diffview.ColorPair{
			Foreground: "#f38ba8", // Red
			Background: "#3f2d2d", // Subtle red background
		},
		Context: diffview.ColorPair{
			Foreground: "#cdd6f4", // Light gray
		},
		HunkHeader: diffview.ColorPair{
			Foreground: "#89b4fa", // Blue
		},
		FileHeader: diffview.ColorPair{
			Foreground: "#f9e2af", // Yellow
			Background: "#313244", // Dark surface
		},
		LineNumber: diffview.ColorPair{
			Foreground: "#6c7086", // Muted gray
		},
		AddedHighlight: diffview.ColorPair{
			Foreground: "#1e1e2e", // Dark text on bright background
			Background: "#a6e3a1", // Bright green background
		},
		DeletedHighlight: diffview.ColorPair{
			Foreground: "#1e1e2e", // Dark text on bright background
			Background: "#f38ba8", // Bright red background
		},
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
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
		m.width = msg.Width
		statusBarHeight := 1
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-statusBarHeight)
			m.viewport.SetContent(m.content)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
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

// statusBarView renders the status bar with position info.
func (m Model) statusBarView() string {
	// Styles for status bar components
	barStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#313244")).
		Foreground(lipgloss.Color("#cdd6f4"))

	dimStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#313244")).
		Foreground(lipgloss.Color("#6c7086"))

	sepStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#313244")).
		Foreground(lipgloss.Color("#45475a"))

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

// renderDiffWithPositions converts a Diff to a styled string and tracks hunk/file positions.
// Positions represent the line number where each file/hunk header begins.
// If renderer is nil, the default lipgloss renderer is used.
func renderDiffWithPositions(diff *diffview.Diff, styles diffview.Styles, renderer *lipgloss.Renderer) (content string, hunkPositions, filePositions []int) {
	if diff == nil {
		return "", nil, nil
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
	addedHighlightStyle := styleFromColorPair(styles.AddedHighlight, renderer)
	deletedHighlightStyle := styleFromColorPair(styles.DeletedHighlight, renderer)

	// Word differ for inline highlighting
	differ := worddiff.NewDiffer()

	var sb strings.Builder
	lineNum := 0
	for _, file := range diff.Files {
		// Only render file if it has hunks (skip binary/empty files)
		if len(file.Hunks) == 0 {
			continue
		}

		// Track file position at the first header line
		filePositions = append(filePositions, lineNum)

		// Render file headers with styling
		sb.WriteString(fileHeaderStyle.Render(fmt.Sprintf("--- %s", file.OldPath)))
		sb.WriteString("\n")
		lineNum++
		sb.WriteString(fileHeaderStyle.Render(fmt.Sprintf("+++ %s", file.NewPath)))
		sb.WriteString("\n")
		lineNum++

		for _, hunk := range file.Hunks {
			// Track hunk position at the header line
			hunkPositions = append(hunkPositions, lineNum)

			// Render hunk header with styling
			header := formatHunkHeader(hunk)
			sb.WriteString(hunkHeaderStyle.Render(header))
			sb.WriteString("\n")
			lineNum++

			// Render lines with gutter and prefixes
			// Use index-based loop to detect deleted+added pairs
			lines := hunk.Lines
			for i := 0; i < len(lines); i++ {
				line := lines[i]

				// Check for deleted+added pair for word-level diff
				if line.Type == diffview.LineDeleted && i+1 < len(lines) && lines[i+1].Type == diffview.LineAdded {
					deletedLine := line
					addedLine := lines[i+1]

					// Compute word-level diff
					deletedContent := strings.TrimSuffix(deletedLine.Content, "\n")
					addedContent := strings.TrimSuffix(addedLine.Content, "\n")
					deletedSegs, addedSegs := differ.Diff(deletedContent, addedContent)

					// Render deleted line with word highlighting
					sb.WriteString(formatGutter(deletedLine.OldLineNum, deletedLine.NewLineNum, gutterWidth, lineNumStyle))
					styledDeleted := renderLineWithSegments("-", deletedSegs, deletedStyle, deletedHighlightStyle, lineWidth)
					sb.WriteString(styledDeleted)
					sb.WriteString("\n")
					lineNum++

					// Render added line with word highlighting
					sb.WriteString(formatGutter(addedLine.OldLineNum, addedLine.NewLineNum, gutterWidth, lineNumStyle))
					styledAdded := renderLineWithSegments("+", addedSegs, addedStyle, addedHighlightStyle, lineWidth)
					sb.WriteString(styledAdded)
					sb.WriteString("\n")
					lineNum++

					// Skip the added line since we already processed it
					i++
					continue
				}

				// Standard line rendering (no word-level diff)
				gutter := formatGutter(line.OldLineNum, line.NewLineNum, gutterWidth, lineNumStyle)
				sb.WriteString(gutter)

				prefix := linePrefixFor(line.Type)
				lineContent := strings.TrimSuffix(line.Content, "\n")
				fullLine := prefix + lineContent

				var styledLine string
				switch line.Type {
				case diffview.LineAdded:
					styledLine = addedStyle.Render(padLine(fullLine, lineWidth))
				case diffview.LineDeleted:
					styledLine = deletedStyle.Render(padLine(fullLine, lineWidth))
				default:
					styledLine = contextStyle.Render(fullLine)
				}
				sb.WriteString(styledLine)
				sb.WriteString("\n")
				lineNum++
			}
		}
	}
	return sb.String(), hunkPositions, filePositions
}

// renderLineWithSegments renders a line with word-level highlighting.
// Segments marked as Changed get the highlight style, others get the base style.
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
	// Note: We need to account for prefix length and Unicode display width
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
// Format: "  12    14 │" for lines with both numbers
// Format: "  12     - │" for deleted lines (no new line number)
// Format: "   -    14 │" for added lines (no old line number)
func formatGutter(oldLineNum, newLineNum, width int, style lipgloss.Style) string {
	oldStr := formatLineNum(oldLineNum, width)
	newStr := formatLineNum(newLineNum, width)
	gutter := fmt.Sprintf("%s %s │", oldStr, newStr)
	return style.Render(gutter)
}

// formatLineNum formats a line number for the gutter.
// Returns right-aligned number or "-" for zero (missing) line numbers.
func formatLineNum(num, width int) string {
	if num == 0 {
		return fmt.Sprintf("%*s", width, "-")
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

// lineWidth is the width to pad added/deleted lines to for full-width backgrounds.
// Using a large fixed value ensures backgrounds extend across typical terminal widths.
const lineWidth = 256

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
	programOpts []tea.ProgramOption
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

// NewViewer creates a new Viewer.
func NewViewer(opts ...ViewerOption) *Viewer {
	v := &Viewer{}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// View displays the diff and blocks until the user exits.
func (v *Viewer) View(ctx context.Context, diff *diffview.Diff) error {
	m := NewModel(diff)
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
