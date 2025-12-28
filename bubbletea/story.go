package bubbletea

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fwojciec/diffview"
)

// StoryModel displays a diff with story-aware navigation and styling.
type StoryModel struct {
	diff  *diffview.Diff
	story *diffview.StoryClassification

	// Pre-computed mappings (built on construction)
	hunkToSection    map[hunkKey]int    // hunk → section index
	sectionPositions []int              // line numbers where sections start
	sectionIndices   []int              // maps position index to story section index
	hunkCategories   map[hunkKey]string // hunk → category for styling
	collapseText     map[hunkKey]string // hunk → collapse text
	collapsedHunks   map[hunkKey]bool   // tracks runtime collapse state

	// UI state
	viewport   viewport.Model
	keymap     StoryKeyMap
	styles     diffview.Styles
	palette    diffview.Palette
	renderer   *lipgloss.Renderer
	width      int
	ready      bool
	pendingKey string
}

// StoryModelOption configures a StoryModel.
type StoryModelOption func(*storyModelConfig)

type storyModelConfig struct {
	renderer *lipgloss.Renderer
	theme    diffview.Theme
}

// WithStoryRenderer sets a custom lipgloss renderer for the model.
func WithStoryRenderer(r *lipgloss.Renderer) StoryModelOption {
	return func(cfg *storyModelConfig) {
		cfg.renderer = r
	}
}

// WithStoryTheme sets the theme for the model.
func WithStoryTheme(t diffview.Theme) StoryModelOption {
	return func(cfg *storyModelConfig) {
		cfg.theme = t
	}
}

// NewStoryModel creates a new StoryModel with the given diff and classification.
func NewStoryModel(diff *diffview.Diff, story *diffview.StoryClassification, opts ...StoryModelOption) StoryModel {
	cfg := &storyModelConfig{}
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

	// Build lookup maps from story classification
	hunkToSection := make(map[hunkKey]int)
	hunkCategories := make(map[hunkKey]string)
	collapseText := make(map[hunkKey]string)
	collapsedHunks := make(map[hunkKey]bool)

	if story != nil {
		for sectionIdx, section := range story.Sections {
			for _, ref := range section.Hunks {
				key := hunkKey{file: ref.File, hunkIndex: ref.HunkIndex}
				hunkToSection[key] = sectionIdx
				hunkCategories[key] = ref.Category
				if ref.CollapseText != "" {
					collapseText[key] = ref.CollapseText
				}
				// Collapse if explicitly marked or noise category
				if ref.Collapsed || ref.Category == "noise" {
					collapsedHunks[key] = true
				}
			}
		}
	}

	return StoryModel{
		diff:           diff,
		story:          story,
		hunkToSection:  hunkToSection,
		hunkCategories: hunkCategories,
		collapseText:   collapseText,
		collapsedHunks: collapsedHunks,
		keymap:         DefaultStoryKeyMap(),
		styles:         styles,
		palette:        palette,
		renderer:       cfg.renderer,
	}
}

// Init implements tea.Model.
func (m StoryModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m StoryModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		case key.Matches(msg, m.keymap.NextSection):
			m.gotoNextSection()
			return m, nil
		case key.Matches(msg, m.keymap.PrevSection):
			m.gotoPrevSection()
			return m, nil
		case key.Matches(msg, m.keymap.ToggleCollapse):
			m.toggleCurrentHunkCollapse()
			return m, nil
		case key.Matches(msg, m.keymap.ToggleCollapseAll):
			m.toggleAllCollapse()
			return m, nil
		}
	case tea.WindowSizeMsg:
		statusBarHeight := 1
		widthChanged := m.width != msg.Width
		m.width = msg.Width

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-statusBarHeight)
			m.viewport.SetContent(m.renderContent())
			m.sectionPositions, m.sectionIndices = m.computeSectionPositions()
			m.ready = true
		} else if widthChanged {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - statusBarHeight
			m.viewport.SetContent(m.renderContent())
			m.sectionPositions, m.sectionIndices = m.computeSectionPositions()
		} else {
			m.viewport.Height = msg.Height - statusBarHeight
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View implements tea.Model.
func (m StoryModel) View() string {
	if !m.ready {
		return "Loading..."
	}
	return lipgloss.JoinVertical(lipgloss.Left, m.viewport.View(), m.statusBarView())
}

// renderContent renders the diff content with story-aware configuration.
func (m StoryModel) renderContent() string {
	return renderDiff(renderConfig{
		diff:           m.diff,
		styles:         m.styles,
		renderer:       m.renderer,
		width:          m.width,
		collapsedHunks: m.collapsedHunks,
		hunkCategories: m.hunkCategories,
		collapseText:   m.collapseText,
	})
}

// computeSectionPositions calculates the line positions where each section starts.
// A section starts at the first hunk that belongs to it.
// Returns both positions and the mapping from position index to story section index.
func (m StoryModel) computeSectionPositions() (positions []int, indices []int) {
	if m.story == nil || len(m.story.Sections) == 0 {
		return nil, nil
	}
	if m.diff == nil {
		return nil, nil
	}

	// Compute hunk positions first
	hunkPositions, _ := m.computePositions()

	// Build section positions by finding the first hunk in each section
	sectionPositions := make([]int, len(m.story.Sections))
	sectionFound := make([]bool, len(m.story.Sections))

	hunkIdx := 0
	for _, file := range m.diff.Files {
		if !shouldRenderFile(file) {
			continue
		}
		path := filePath(file)
		for i := range file.Hunks {
			key := hunkKey{file: path, hunkIndex: i}
			if sectionIdx, ok := m.hunkToSection[key]; ok {
				if hunkIdx < len(hunkPositions) && !sectionFound[sectionIdx] {
					sectionPositions[sectionIdx] = hunkPositions[hunkIdx]
					sectionFound[sectionIdx] = true
				}
			}
			hunkIdx++
		}
	}

	// Filter to only include found sections, keeping track of original indices
	for i := range m.story.Sections {
		if sectionFound[i] {
			positions = append(positions, sectionPositions[i])
			indices = append(indices, i)
		}
	}

	return positions, indices
}

// computePositions calculates line positions accounting for collapse state.
func (m StoryModel) computePositions() (hunkPositions, filePositions []int) {
	if m.diff == nil {
		return nil, nil
	}

	lineNum := 0
	for _, file := range m.diff.Files {
		if !shouldRenderFile(file) {
			continue
		}

		path := filePath(file)
		filePositions = append(filePositions, lineNum)
		lineNum++ // file header

		if len(file.Hunks) == 0 {
			lineNum++ // "(empty)" line
		} else {
			for hunkIdx, hunk := range file.Hunks {
				hunkPositions = append(hunkPositions, lineNum)

				key := hunkKey{file: path, hunkIndex: hunkIdx}
				if m.collapsedHunks[key] {
					lineNum++ // collapsed: single line
				} else {
					lineNum++                  // header
					lineNum += len(hunk.Lines) // content
				}
			}
		}
	}
	return hunkPositions, filePositions
}

// gotoNextSection scrolls to the next section.
func (m *StoryModel) gotoNextSection() {
	if len(m.sectionPositions) == 0 {
		return
	}
	currentLine := m.viewport.YOffset
	// Find index of current section (first one >= currentLine)
	currentIdx := -1
	for i, pos := range m.sectionPositions {
		if pos >= currentLine {
			currentIdx = i
			break
		}
	}
	// If no position >= currentLine, we're past all sections
	if currentIdx == -1 {
		return
	}
	// Navigate to next section if it exists
	nextIdx := currentIdx + 1
	if nextIdx < len(m.sectionPositions) {
		m.viewport.SetYOffset(m.sectionPositions[nextIdx])
	}
}

// toggleCurrentHunkCollapse toggles the collapse state of the hunk at the current position.
func (m *StoryModel) toggleCurrentHunkCollapse() {
	hunkPositions, _ := m.computePositions()
	if len(hunkPositions) == 0 {
		return
	}

	currentLine := m.viewport.YOffset

	// Find current hunk (last position <= currentLine)
	currentHunkIdx := -1
	for i, pos := range hunkPositions {
		if pos <= currentLine {
			currentHunkIdx = i
		} else {
			break
		}
	}

	// Default to first hunk if at top
	if currentHunkIdx == -1 && len(hunkPositions) > 0 {
		currentHunkIdx = 0
	}

	if currentHunkIdx == -1 {
		return
	}

	// Find the hunk key for this index
	key := m.hunkKeyAtIndex(currentHunkIdx)
	if key == nil {
		return
	}

	// Toggle collapse state
	m.collapsedHunks[*key] = !m.collapsedHunks[*key]

	// Re-render content
	m.viewport.SetContent(m.renderContent())
	m.sectionPositions, m.sectionIndices = m.computeSectionPositions()
}

// toggleAllCollapse toggles all hunks between fully collapsed and fully expanded.
func (m *StoryModel) toggleAllCollapse() {
	if len(m.hunkToSection) == 0 {
		return
	}

	// Count how many are currently collapsed
	collapsedCount := 0
	for _, collapsed := range m.collapsedHunks {
		if collapsed {
			collapsedCount++
		}
	}

	// If more than half are collapsed, expand all; otherwise collapse all
	newState := collapsedCount <= len(m.hunkToSection)/2

	for key := range m.hunkToSection {
		m.collapsedHunks[key] = newState
	}

	// Re-render content
	m.viewport.SetContent(m.renderContent())
	m.sectionPositions, m.sectionIndices = m.computeSectionPositions()
}

// hunkKeyAtIndex returns the hunk key for the hunk at the given rendered index.
func (m StoryModel) hunkKeyAtIndex(idx int) *hunkKey {
	i := 0
	for _, file := range m.diff.Files {
		if !shouldRenderFile(file) {
			continue
		}
		path := filePath(file)
		for hunkIdx := range file.Hunks {
			if i == idx {
				key := hunkKey{file: path, hunkIndex: hunkIdx}
				return &key
			}
			i++
		}
	}
	return nil
}

// gotoPrevSection scrolls to the previous section.
func (m *StoryModel) gotoPrevSection() {
	if len(m.sectionPositions) == 0 {
		return
	}
	currentLine := m.viewport.YOffset
	// Find index of current section (first one >= currentLine)
	currentIdx := -1
	for i, pos := range m.sectionPositions {
		if pos >= currentLine {
			currentIdx = i
			break
		}
	}
	// If no position >= currentLine, we're past all sections, go to last
	if currentIdx == -1 {
		m.viewport.SetYOffset(m.sectionPositions[len(m.sectionPositions)-1])
		return
	}
	// Navigate to previous section if it exists
	prevIdx := currentIdx - 1
	if prevIdx >= 0 {
		m.viewport.SetYOffset(m.sectionPositions[prevIdx])
	}
}

// newStyle creates a new lipgloss style using the model's renderer.
func (m StoryModel) newStyle() lipgloss.Style {
	if m.renderer != nil {
		return m.renderer.NewStyle()
	}
	return lipgloss.NewStyle()
}

// statusBarView renders the status bar with position info.
func (m StoryModel) statusBarView() string {
	barStyle := m.newStyle().
		Background(lipgloss.Color(m.palette.UIBackground)).
		Foreground(lipgloss.Color(m.palette.Foreground))

	dimStyle := m.newStyle().
		Background(lipgloss.Color(m.palette.UIBackground)).
		Foreground(lipgloss.Color(m.palette.Context))

	sepStyle := m.newStyle().
		Background(lipgloss.Color(m.palette.UIBackground)).
		Foreground(lipgloss.Color(m.palette.UIForeground))

	// Format position info
	hunkPositions, filePositions := m.computePositions()
	fileIdx, fileTotal := m.currentPosition(filePositions)
	hunkIdx, hunkTotal := m.currentPosition(hunkPositions)
	sectionIdx, sectionTotal, sectionTitle := m.currentSection()

	fileWidth := digitWidth(fileTotal)
	hunkWidth := digitWidth(hunkTotal)
	sectionWidth := digitWidth(sectionTotal)

	filePos := fmt.Sprintf("file %*d/%-*d", fileWidth, fileIdx, fileWidth, fileTotal)
	hunkPos := fmt.Sprintf("hunk %*d/%-*d", hunkWidth, hunkIdx, hunkWidth, hunkTotal)
	scrollPos := m.scrollPosition()

	// Build status bar with separators
	sep := sepStyle.Render(" │ ")
	content := barStyle.Render(filePos) + sep +
		barStyle.Render(hunkPos) + sep

	// Add section indicator if sections exist
	if sectionTotal > 0 {
		sectionPos := fmt.Sprintf("section %*d/%-*d: %s", sectionWidth, sectionIdx, sectionWidth, sectionTotal, sectionTitle)
		content += barStyle.Render(sectionPos) + sep
	}

	content += barStyle.Render(scrollPos) + sep +
		dimStyle.Render("j/k:scroll  s/S:section  o:collapse  z:all  q:quit") +
		barStyle.Render("  ")

	// Right-align by padding left side with background
	contentWidth := lipgloss.Width(content)
	if m.width > contentWidth {
		padding := barStyle.Render(strings.Repeat(" ", m.width-contentWidth))
		content = padding + content
	}

	return content
}

// currentPosition returns the current position (1-based) and total count.
func (m StoryModel) currentPosition(positions []int) (current, total int) {
	total = len(positions)
	if total == 0 {
		return 0, 0
	}

	currentLine := m.viewport.YOffset
	current = 1

	for i, pos := range positions {
		if pos <= currentLine {
			current = i + 1
		} else {
			break
		}
	}

	return current, total
}

// scrollPosition returns a string indicating the scroll position.
func (m StoryModel) scrollPosition() string {
	if m.viewport.AtTop() {
		return "Top"
	}
	if m.viewport.AtBottom() {
		return "Bot"
	}
	percent := int(m.viewport.ScrollPercent() * 100)
	return fmt.Sprintf("%2d%%", percent)
}

// currentSection returns the current section index (1-based), total count, and title.
// Returns (0, 0, "") if there are no sections.
func (m StoryModel) currentSection() (current, total int, title string) {
	if m.story == nil || len(m.story.Sections) == 0 || len(m.sectionPositions) == 0 {
		return 0, 0, ""
	}

	total = len(m.sectionPositions)
	currentLine := m.viewport.YOffset
	current = 1

	// Find which section we're currently in based on viewport position.
	// The current section is the last one whose start position is at or before the viewport top.
	positionIdx := 0
	for i, pos := range m.sectionPositions {
		if pos <= currentLine {
			current = i + 1
			positionIdx = i
		} else {
			break
		}
	}

	// Special case: when at bottom, if a later section starts within the viewport,
	// show that section instead. This handles the case where the viewport can't
	// scroll far enough to have section N's position <= YOffset.
	if m.viewport.AtBottom() && positionIdx < len(m.sectionPositions)-1 {
		nextSectionPos := m.sectionPositions[positionIdx+1]
		viewportBottom := currentLine + m.viewport.Height
		if nextSectionPos < viewportBottom {
			positionIdx++
			current = positionIdx + 1
		}
	}

	// Get the title from the story section using the stored index mapping
	if positionIdx < len(m.sectionIndices) {
		storySectionIdx := m.sectionIndices[positionIdx]
		if storySectionIdx < len(m.story.Sections) {
			title = m.story.Sections[storySectionIdx].Title
		}
	}

	return current, total, title
}
