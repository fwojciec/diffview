package bubbletea

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fwojciec/diffview"
)

// Panel identifies which panel is active.
type Panel int

// Panel constants.
const (
	PanelDiff Panel = iota
	PanelStory
)

// Mode identifies the current interaction mode.
type Mode int

// Mode constants.
const (
	ModeReview Mode = iota
	ModeCritique
)

// EvalModel is the Bubble Tea model for evaluating diff stories.
type EvalModel struct {
	// Data
	cases        []diffview.EvalCase
	judgments    map[string]*diffview.Judgment
	currentIndex int

	// UI Components
	diffViewport  viewport.Model
	storyViewport viewport.Model

	// State
	activePanel Panel
	mode        Mode
	ready       bool

	// Rendering
	width, height int

	// Persistence
	store      diffview.JudgmentStore
	outputPath string

	// Keybindings
	keymap EvalKeyMap
}

// EvalModelOption configures an EvalModel.
type EvalModelOption func(*EvalModel)

// WithJudgmentStore sets the store for persisting judgments.
func WithJudgmentStore(store diffview.JudgmentStore, outputPath string) EvalModelOption {
	return func(m *EvalModel) {
		m.store = store
		m.outputPath = outputPath
	}
}

// WithExistingJudgments loads previously recorded judgments.
func WithExistingJudgments(judgments []diffview.Judgment) EvalModelOption {
	return func(m *EvalModel) {
		for i := range judgments {
			j := judgments[i]
			m.judgments[j.Commit] = &j
		}
	}
}

// NewEvalModel creates a new EvalModel with the given cases.
func NewEvalModel(cases []diffview.EvalCase, opts ...EvalModelOption) EvalModel {
	m := EvalModel{
		cases:       cases,
		judgments:   make(map[string]*diffview.Judgment),
		activePanel: PanelDiff,
		mode:        ModeReview,
		keymap:      DefaultEvalKeyMap(),
	}

	for _, opt := range opts {
		opt(&m)
	}

	return m
}

// Init implements tea.Model.
func (m EvalModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m EvalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.mode == ModeReview {
			return m.handleReviewKeys(msg)
		}
		return m.handleCritiqueKeys(msg)

	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)
	}

	// Update the active viewport
	var cmd tea.Cmd
	if m.activePanel == PanelDiff {
		m.diffViewport, cmd = m.diffViewport.Update(msg)
	} else {
		m.storyViewport, cmd = m.storyViewport.Update(msg)
	}
	return m, cmd
}

func (m EvalModel) handleReviewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keymap.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keymap.NextCase):
		if m.currentIndex < len(m.cases)-1 {
			m.currentIndex++
			m.updateViewportContent()
		}
		return m, nil

	case key.Matches(msg, m.keymap.PrevCase):
		if m.currentIndex > 0 {
			m.currentIndex--
			m.updateViewportContent()
		}
		return m, nil

	case key.Matches(msg, m.keymap.DiffPanel):
		m.activePanel = PanelDiff
		return m, nil

	case key.Matches(msg, m.keymap.StoryPanel):
		m.activePanel = PanelStory
		return m, nil

	case key.Matches(msg, m.keymap.HalfPageUp):
		if m.activePanel == PanelDiff {
			m.diffViewport.HalfPageUp()
		} else {
			m.storyViewport.HalfPageUp()
		}
		return m, nil

	case key.Matches(msg, m.keymap.HalfPageDown):
		if m.activePanel == PanelDiff {
			m.diffViewport.HalfPageDown()
		} else {
			m.storyViewport.HalfPageDown()
		}
		return m, nil

	case key.Matches(msg, m.keymap.Pass):
		m.recordJudgment(true)
		return m, nil

	case key.Matches(msg, m.keymap.Fail):
		m.recordJudgment(false)
		return m, nil
	}

	return m, nil
}

func (m EvalModel) handleCritiqueKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keymap.ExitCritique):
		m.mode = ModeReview
		return m, nil
	}
	return m, nil
}

func (m *EvalModel) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	// Calculate panel heights
	// Reserve: judgment bar (1), status bar (2), borders (3)
	usableHeight := msg.Height - 6
	if usableHeight < 2 {
		usableHeight = 2 // Minimum height for tiny terminals
	}
	diffHeight := usableHeight * 50 / 100
	storyHeight := usableHeight - diffHeight

	if !m.ready {
		m.diffViewport = viewport.New(msg.Width, diffHeight)
		m.storyViewport = viewport.New(msg.Width, storyHeight)
		m.updateViewportContent()
		m.ready = true
	} else {
		m.diffViewport.Width = msg.Width
		m.diffViewport.Height = diffHeight
		m.storyViewport.Width = msg.Width
		m.storyViewport.Height = storyHeight
	}

	return m, nil
}

func (m *EvalModel) updateViewportContent() {
	if len(m.cases) == 0 {
		m.diffViewport.SetContent("No cases loaded")
		m.storyViewport.SetContent("")
		return
	}

	c := m.cases[m.currentIndex]

	// Render diff content
	var diffContent strings.Builder
	for _, ah := range c.Hunks {
		diffContent.WriteString(formatHunkHeader(ah.Hunk))
		diffContent.WriteString("\n")
		for _, line := range ah.Hunk.Lines {
			prefix := linePrefixFor(line.Type)
			diffContent.WriteString(prefix)
			diffContent.WriteString(line.Content)
			diffContent.WriteString("\n")
		}
	}
	m.diffViewport.SetContent(diffContent.String())
	m.diffViewport.GotoTop()

	// Render story content
	var storyContent strings.Builder
	storyContent.WriteString(fmt.Sprintf("[%s] %s\n\n", c.Story.ChangeType, c.Story.Summary))
	for _, part := range c.Story.Parts {
		storyContent.WriteString(fmt.Sprintf("• %s: %s\n", part.Role, part.Explanation))
		if len(part.HunkIDs) > 0 {
			storyContent.WriteString(fmt.Sprintf("  hunks: %s\n", strings.Join(part.HunkIDs, ", ")))
		}
	}
	m.storyViewport.SetContent(storyContent.String())
	m.storyViewport.GotoTop()
}

func (m *EvalModel) recordJudgment(pass bool) {
	if len(m.cases) == 0 {
		return
	}

	c := m.cases[m.currentIndex]

	// Preserve existing critique when toggling pass/fail
	var critique string
	if existing := m.judgments[c.Commit]; existing != nil {
		critique = existing.Critique
	}

	j := &diffview.Judgment{
		Commit:   c.Commit,
		Index:    m.currentIndex,
		Pass:     pass,
		Critique: critique,
		JudgedAt: time.Now(),
	}
	m.judgments[c.Commit] = j

	m.persistJudgments()
}

func (m *EvalModel) persistJudgments() {
	if m.store == nil || m.outputPath == "" {
		return
	}
	judgments := make([]diffview.Judgment, 0, len(m.judgments))
	for _, j := range m.judgments {
		judgments = append(judgments, *j)
	}
	// Sort by index for deterministic output
	sort.Slice(judgments, func(i, k int) bool {
		return judgments[i].Index < judgments[k].Index
	})
	// Best-effort save - errors are logged but don't block the UI
	// TODO: Consider adding error display in status bar
	_ = m.store.Save(m.outputPath, judgments)
}

// View implements tea.Model.
func (m EvalModel) View() string {
	if !m.ready {
		return "Loading..."
	}

	var s strings.Builder

	// Diff panel header
	diffHeader := m.renderPanelHeader("DIFF", m.activePanel == PanelDiff)
	s.WriteString(diffHeader)
	s.WriteString("\n")
	s.WriteString(m.diffViewport.View())
	s.WriteString("\n")

	// Story panel header
	storyHeader := m.renderPanelHeader("STORY", m.activePanel == PanelStory)
	s.WriteString(storyHeader)
	s.WriteString("\n")
	s.WriteString(m.storyViewport.View())
	s.WriteString("\n")

	// Judgment bar
	s.WriteString(m.renderJudgmentBar())
	s.WriteString("\n")

	// Status bar
	s.WriteString(m.renderStatusBar())

	return s.String()
}

func (m EvalModel) renderPanelHeader(name string, active bool) string {
	style := lipgloss.NewStyle().Bold(true)
	if active {
		return style.Render(fmt.Sprintf("%s [active]", name))
	}
	return style.Render(name)
}

func (m EvalModel) renderJudgmentBar() string {
	if len(m.cases) == 0 {
		return ""
	}

	c := m.cases[m.currentIndex]
	j := m.judgments[c.Commit]

	passMarker := "○"
	failMarker := "○"
	critique := "[not set]"

	if j != nil {
		if j.Pass {
			passMarker = "●"
		} else {
			failMarker = "●"
		}
		if j.Critique != "" {
			critique = j.Critique
			if len(critique) > 30 {
				critique = critique[:27] + "..."
			}
		}
	}

	return fmt.Sprintf("%s Pass  %s Fail    Critique: %s", passMarker, failMarker, critique)
}

func (m EvalModel) renderStatusBar() string {
	if len(m.cases) == 0 {
		return "No cases"
	}

	// Count judged cases
	judged := 0
	for _, c := range m.cases {
		if _, ok := m.judgments[c.Commit]; ok {
			judged++
		}
	}

	caseInfo := fmt.Sprintf("case %d/%d", m.currentIndex+1, len(m.cases))
	progress := fmt.Sprintf("%d/%d reviewed", judged, len(m.cases))
	help := "[d]iff [s]tory [p]ass [f]ail [j/k]nav [q]uit"

	return fmt.Sprintf("%s │ %s │ %s", caseInfo, progress, help)
}
