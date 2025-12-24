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

// NewModel creates a new Model with the given diff and default dark theme.
func NewModel(diff *diffview.Diff) Model {
	return NewModelWithStyles(diff, defaultStyles())
}

// NewModelWithStyles creates a new Model with the given diff and styles.
func NewModelWithStyles(diff *diffview.Diff, styles diffview.Styles) Model {
	content, hunkPositions, filePositions := renderDiffWithPositions(diff, styles)
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
		},
		Deleted: diffview.ColorPair{
			Foreground: "#f38ba8", // Red
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
	fileIdx, fileTotal := m.currentFilePosition()
	hunkIdx, hunkTotal := m.currentHunkPosition()
	scrollPos := m.scrollPosition()
	keyHints := "j/k:scroll  n/N:hunk  ]/[:file  q:quit"
	return fmt.Sprintf("file %d/%d  hunk %d/%d  %s  %s", fileIdx, fileTotal, hunkIdx, hunkTotal, scrollPos, keyHints)
}

// scrollPosition returns a string indicating the scroll position.
func (m Model) scrollPosition() string {
	if m.viewport.AtTop() {
		return "Top"
	}
	if m.viewport.AtBottom() {
		return "Bot"
	}
	// Calculate percentage
	percent := m.viewport.ScrollPercent() * 100
	return fmt.Sprintf("%d%%", int(percent))
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

// renderDiffWithPositions converts a Diff to a styled string and tracks hunk/file positions.
// Positions represent the line number where each file/hunk header begins.
func renderDiffWithPositions(diff *diffview.Diff, styles diffview.Styles) (content string, hunkPositions, filePositions []int) {
	if diff == nil {
		return "", nil, nil
	}

	// Create lipgloss styles from color pairs
	fileHeaderStyle := styleFromColorPair(styles.FileHeader)
	hunkHeaderStyle := styleFromColorPair(styles.HunkHeader)
	addedStyle := styleFromColorPair(styles.Added)
	deletedStyle := styleFromColorPair(styles.Deleted)
	contextStyle := styleFromColorPair(styles.Context)

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

			// Render lines with prefixes and styling
			for _, line := range hunk.Lines {
				prefix := linePrefixFor(line.Type)
				// Content may include trailing newline from parser; trim it
				lineContent := strings.TrimSuffix(line.Content, "\n")
				fullLine := prefix + lineContent

				// Apply appropriate style based on line type
				var styledLine string
				switch line.Type {
				case diffview.LineAdded:
					styledLine = addedStyle.Render(fullLine)
				case diffview.LineDeleted:
					styledLine = deletedStyle.Render(fullLine)
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

// styleFromColorPair creates a lipgloss style from a ColorPair.
func styleFromColorPair(cp diffview.ColorPair) lipgloss.Style {
	style := lipgloss.NewStyle()
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
