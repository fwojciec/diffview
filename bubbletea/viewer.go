// Package bubbletea provides a terminal UI viewer for diffs using the Bubble Tea framework.
package bubbletea

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
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
}

// NewModel creates a new Model with the given diff.
func NewModel(diff *diffview.Diff) Model {
	content, hunkPositions, filePositions := renderDiffWithPositions(diff)
	return Model{
		diff:          diff,
		content:       content,
		keymap:        DefaultKeyMap(),
		hunkPositions: hunkPositions,
		filePositions: filePositions,
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
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height)
			m.viewport.SetContent(m.content)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height
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
	return m.viewport.View()
}

// HunkPositions returns the line numbers where each hunk starts.
func (m Model) HunkPositions() []int {
	return m.hunkPositions
}

// FilePositions returns the line numbers where each file starts.
func (m Model) FilePositions() []int {
	return m.filePositions
}

// gotoNextPosition scrolls to the next position after the current viewport offset.
func (m *Model) gotoNextPosition(positions []int) {
	currentLine := m.viewport.YOffset
	for _, pos := range positions {
		if pos > currentLine {
			m.viewport.SetYOffset(pos)
			return
		}
	}
	// Already at or past last position, stay where we are
}

// gotoPrevPosition scrolls to the previous position before the current viewport offset.
func (m *Model) gotoPrevPosition(positions []int) {
	currentLine := m.viewport.YOffset
	// Find the last position that's before the current line
	for i := len(positions) - 1; i >= 0; i-- {
		if positions[i] < currentLine {
			m.viewport.SetYOffset(positions[i])
			return
		}
	}
	// Already at or before first position, stay where we are
}

// renderDiffWithPositions converts a Diff to a string and tracks hunk/file positions.
// Positions represent the line number where each hunk/file's content begins.
// Note: If file/hunk headers are added in the future, positions should be updated
// to point to the header lines rather than content lines.
func renderDiffWithPositions(diff *diffview.Diff) (content string, hunkPositions, filePositions []int) {
	if diff == nil {
		return "", nil, nil
	}

	var sb strings.Builder
	lineNum := 0
	for _, file := range diff.Files {
		filePositions = append(filePositions, lineNum)
		for _, hunk := range file.Hunks {
			hunkPositions = append(hunkPositions, lineNum)
			for _, line := range hunk.Lines {
				sb.WriteString(line.Content)
				sb.WriteString("\n")
				lineNum++
			}
		}
	}
	return sb.String(), hunkPositions, filePositions
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
