// Package bubbletea provides a terminal UI viewer for diffs using the Bubble Tea framework.
package bubbletea

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fwojciec/diffview"
)

// Model is the Bubble Tea model for viewing diffs.
type Model struct {
	diff     *diffview.Diff
	viewport viewport.Model
	ready    bool
	content  string
}

// NewModel creates a new Model with the given diff.
func NewModel(diff *diffview.Diff) Model {
	return Model{
		diff:    diff,
		content: renderDiff(diff),
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
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
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

// renderDiff converts a Diff to a string for display.
func renderDiff(diff *diffview.Diff) string {
	if diff == nil {
		return ""
	}

	var sb strings.Builder
	for _, file := range diff.Files {
		for _, hunk := range file.Hunks {
			for _, line := range hunk.Lines {
				sb.WriteString(line.Content)
				sb.WriteString("\n")
			}
		}
	}
	return sb.String()
}

// Viewer implements diffview.Viewer using a Bubble Tea TUI.
type Viewer struct{}

// NewViewer creates a new Viewer.
func NewViewer() *Viewer {
	return &Viewer{}
}

// View displays the diff and blocks until the user exits.
func (v *Viewer) View(_ context.Context, diff *diffview.Diff) error {
	m := NewModel(diff)
	p := tea.NewProgram(m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := p.Run()
	return err
}
