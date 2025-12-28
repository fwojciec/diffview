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
