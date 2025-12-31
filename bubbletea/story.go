package bubbletea

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fwojciec/diffstory"
)

// StoryModel displays a diff with story-aware navigation and styling.
type StoryModel struct {
	diff  *diffview.Diff
	story *diffview.StoryClassification

	// Pre-computed mappings (built on construction)
	hunkToSection     map[hunkKey]int    // hunk → section index
	hunkCategories    map[hunkKey]string // hunk → category for styling
	collapseText      map[hunkKey]string // hunk → collapse text
	collapsedHunks    map[hunkKey]bool   // tracks runtime collapse state
	llmCollapsedHunks map[hunkKey]bool   // tracks which hunks were originally collapsed by LLM

	// Section filtering
	activeSection int  // 0 = intro (if showIntro) or first code section
	showIntro     bool // whether intro slide is enabled

	// Syntax highlighting
	languageDetector diffview.LanguageDetector
	tokenizer        diffview.Tokenizer
	wordDiffer       diffview.WordDiffer

	// Case saving
	input         *diffview.ClassificationInput // optional: full input for constructing EvalCase
	caseSaver     diffview.EvalCaseSaver
	caseSaverPath string

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
	renderer         *lipgloss.Renderer
	theme            diffview.Theme
	languageDetector diffview.LanguageDetector
	tokenizer        diffview.Tokenizer
	wordDiffer       diffview.WordDiffer
	showIntro        bool
	input            *diffview.ClassificationInput
	caseSaver        diffview.EvalCaseSaver
	caseSaverPath    string
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

// WithStoryLanguageDetector sets the language detector for syntax highlighting.
func WithStoryLanguageDetector(d diffview.LanguageDetector) StoryModelOption {
	return func(cfg *storyModelConfig) {
		cfg.languageDetector = d
	}
}

// WithStoryTokenizer sets the tokenizer for syntax highlighting.
func WithStoryTokenizer(t diffview.Tokenizer) StoryModelOption {
	return func(cfg *storyModelConfig) {
		cfg.tokenizer = t
	}
}

// WithStoryWordDiffer sets the word differ for word-level highlighting.
func WithStoryWordDiffer(d diffview.WordDiffer) StoryModelOption {
	return func(cfg *storyModelConfig) {
		cfg.wordDiffer = d
	}
}

// WithIntroSlide enables the intro slide, starting the viewer at an overview
// rather than jumping directly into code.
func WithIntroSlide() StoryModelOption {
	return func(cfg *storyModelConfig) {
		cfg.showIntro = true
	}
}

// WithStoryInput sets the classification input for constructing EvalCase when saving.
func WithStoryInput(input diffview.ClassificationInput) StoryModelOption {
	return func(cfg *storyModelConfig) {
		cfg.input = &input
	}
}

// WithStoryCaseSaver sets the saver for exporting cases to an eval dataset.
func WithStoryCaseSaver(s diffview.EvalCaseSaver, path string) StoryModelOption {
	return func(cfg *storyModelConfig) {
		cfg.caseSaver = s
		cfg.caseSaverPath = path
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
	llmCollapsedHunks := make(map[hunkKey]bool)

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
					llmCollapsedHunks[key] = true // Track original LLM decision
				}
			}
		}
	}

	return StoryModel{
		diff:              diff,
		story:             story,
		hunkToSection:     hunkToSection,
		hunkCategories:    hunkCategories,
		collapseText:      collapseText,
		collapsedHunks:    collapsedHunks,
		llmCollapsedHunks: llmCollapsedHunks,
		showIntro:         cfg.showIntro,
		languageDetector:  cfg.languageDetector,
		tokenizer:         cfg.tokenizer,
		wordDiffer:        cfg.wordDiffer,
		input:             cfg.input,
		caseSaver:         cfg.caseSaver,
		caseSaverPath:     cfg.caseSaverPath,
		keymap:            DefaultStoryKeyMap(),
		styles:            styles,
		palette:           palette,
		renderer:          cfg.renderer,
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
		case key.Matches(msg, m.keymap.ToggleCollapseAll):
			m.toggleAllCollapse()
			return m, nil
		case key.Matches(msg, m.keymap.SaveCase):
			m.saveCurrentCase()
			return m, nil
		}
	case tea.WindowSizeMsg:
		statusBarHeight := 1
		widthChanged := m.width != msg.Width
		m.width = msg.Width

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-statusBarHeight)
			m.viewport.SetContent(m.renderContent())
			m.ready = true
		} else if widthChanged {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - statusBarHeight
			m.viewport.SetContent(m.renderContent())
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

// onIntro returns true if the viewer is on the intro slide.
func (m StoryModel) onIntro() bool {
	return m.showIntro && m.activeSection == 0
}

// codeSectionIndex returns the index into story.Sections for the current view.
// Returns -1 if on the intro slide.
func (m StoryModel) codeSectionIndex() int {
	if m.showIntro {
		return m.activeSection - 1
	}
	return m.activeSection
}

// totalSections returns the total number of navigable sections (including intro if enabled).
func (m StoryModel) totalSections() int {
	if m.story == nil {
		return 0
	}
	total := len(m.story.Sections)
	if m.showIntro {
		total++
	}
	return total
}

// renderContent renders the diff content with story-aware configuration.
func (m StoryModel) renderContent() string {
	if m.onIntro() {
		return m.renderIntro()
	}
	diff, originalIndices := m.filteredDiffWithIndices()
	return renderDiff(renderConfig{
		diff:             diff,
		styles:           m.styles,
		renderer:         m.renderer,
		width:            m.width,
		languageDetector: m.languageDetector,
		tokenizer:        m.tokenizer,
		wordDiffer:       m.wordDiffer,
		collapsedHunks:   m.collapsedHunks,
		hunkCategories:   m.hunkCategories,
		collapseText:     m.collapseText,
		originalIndices:  originalIndices,
	})
}

// renderIntro renders the intro slide content.
func (m StoryModel) renderIntro() string {
	var b strings.Builder

	hasSummary := m.story != nil && m.story.Summary != ""
	hasSections := m.story != nil && len(m.story.Sections) > 0

	// Summary with change type prefix
	if hasSummary {
		b.WriteString("\n")
		if m.story.ChangeType != "" {
			fmt.Fprintf(&b, "[%s] ", m.story.ChangeType)
		}
		b.WriteString(m.story.Summary)
		b.WriteString("\n")
	}

	// Narrative diagram
	if m.story != nil && m.story.Narrative != "" && hasSections {
		if diagram := NarrativeDiagram(m.story.Narrative, m.story.Sections, m.renderer); diagram != "" {
			b.WriteString("\n")
			b.WriteString(diagram)
			b.WriteString("\n")
		} else if explanation := narrativeExplanation(m.story.Narrative); explanation != "" {
			// Fallback to text explanation if diagram not available
			fmt.Fprintf(&b, "\nStory: %s\n", explanation)
		}
	}

	// Section list
	if hasSections {
		b.WriteString("\nSections:\n")
		for i, section := range m.story.Sections {
			if section.Role != "" {
				fmt.Fprintf(&b, "  %d. [%s] %s\n", i+1, section.Role, section.Title)
			} else {
				fmt.Fprintf(&b, "  %d. %s\n", i+1, section.Title)
			}
		}
	}

	// Fallback if no content
	if !hasSummary && !hasSections {
		b.WriteString("\n(No classification available)\n")
	}

	// Navigation hint
	b.WriteString("\n\n[s] next section\n")

	return b.String()
}

// filteredDiffWithIndices returns a diff containing only hunks from the active section,
// along with a mapping from (file, filtered position) to original hunk index.
// If there are no sections or the active section is invalid, returns the full diff with nil indices.
func (m StoryModel) filteredDiffWithIndices() (*diffview.Diff, map[hunkKey]int) {
	if m.diff == nil || m.story == nil || len(m.story.Sections) == 0 {
		return m.diff, nil
	}
	idx := m.codeSectionIndex()
	if idx < 0 || idx >= len(m.story.Sections) {
		return m.diff, nil
	}

	// Build a set of hunks in the active section
	section := m.story.Sections[idx]
	activeHunks := make(map[hunkKey]bool, len(section.Hunks))
	for _, ref := range section.Hunks {
		activeHunks[hunkKey{file: ref.File, hunkIndex: ref.HunkIndex}] = true
	}

	// Create filtered diff with only files/hunks from active section
	// Also build mapping from filtered position to original index
	originalIndices := make(map[hunkKey]int)
	var filteredFiles []diffview.FileDiff
	for _, file := range m.diff.Files {
		path := filePath(file)
		var filteredHunks []diffview.Hunk
		for hunkIdx, hunk := range file.Hunks {
			if activeHunks[hunkKey{file: path, hunkIndex: hunkIdx}] {
				// Map filtered position -> original index
				filteredPos := len(filteredHunks)
				originalIndices[hunkKey{file: path, hunkIndex: filteredPos}] = hunkIdx
				filteredHunks = append(filteredHunks, hunk)
			}
		}
		// Only include file if it has hunks in this section
		if len(filteredHunks) > 0 {
			filteredFile := file
			filteredFile.Hunks = filteredHunks
			filteredFiles = append(filteredFiles, filteredFile)
		}
	}

	return &diffview.Diff{Files: filteredFiles}, originalIndices
}

// filteredDiff returns a diff containing only hunks from the active section.
// If there are no sections or the active section is invalid, returns the full diff.
func (m StoryModel) filteredDiff() *diffview.Diff {
	diff, _ := m.filteredDiffWithIndices()
	return diff
}

// computePositions calculates line positions for the current section's filtered diff.
// Returns hunk positions (in display order) and HunkRefs (for looking up original indices).
func (m StoryModel) computePositions() (hunkPositions []int, hunkRefs []diffview.HunkRef, filePositions []int) {
	filtered := m.filteredDiff()
	if filtered == nil {
		return nil, nil, nil
	}

	// Build map from hunkKey to HunkRef for O(1) lookup
	// This handles the case where section's HunkRefs may not be in file order
	refMap := make(map[hunkKey]diffview.HunkRef)
	idx := m.codeSectionIndex()
	if m.story != nil && idx >= 0 && idx < len(m.story.Sections) {
		for _, ref := range m.story.Sections[idx].Hunks {
			refMap[hunkKey{file: ref.File, hunkIndex: ref.HunkIndex}] = ref
		}
	}

	lineNum := 0
	for _, file := range filtered.Files {
		if !shouldRenderFile(file) {
			continue
		}

		path := filePath(file)
		filePositions = append(filePositions, lineNum)
		lineNum++ // file header

		if len(file.Hunks) == 0 {
			lineNum++ // "(empty)" line
		} else {
			// Track which original hunk index we're at for this file
			// filteredDiff() preserves order, so we scan through original indices
			origIdx := 0
			for _, hunk := range file.Hunks {
				hunkPositions = append(hunkPositions, lineNum)

				// When no sections exist, refMap is empty - create synthetic ref
				if len(refMap) == 0 {
					hunkRefs = append(hunkRefs, diffview.HunkRef{
						File:      path,
						HunkIndex: origIdx,
					})
					origIdx++
				} else {
					// Find original index by scanning until we hit one in the section
					for {
						key := hunkKey{file: path, hunkIndex: origIdx}
						if ref, ok := refMap[key]; ok {
							hunkRefs = append(hunkRefs, ref)
							origIdx++
							break
						}
						origIdx++
					}
				}

				// Use the just-appended ref for collapse check
				lastRef := hunkRefs[len(hunkRefs)-1]
				key := hunkKey{file: lastRef.File, hunkIndex: lastRef.HunkIndex}
				if m.collapsedHunks[key] {
					lineNum++ // collapsed: single line
				} else {
					lineNum++                  // header
					lineNum += len(hunk.Lines) // content
				}
			}
		}
	}
	return hunkPositions, hunkRefs, filePositions
}

// gotoNextSection switches to the next section.
func (m *StoryModel) gotoNextSection() {
	total := m.totalSections()
	if total == 0 {
		return
	}
	// Move to next section if possible
	if m.activeSection < total-1 {
		m.activeSection++
		m.viewport.SetContent(m.renderContent())
		m.viewport.GotoTop()
	}
}

// toggleAllCollapse toggles only LLM-collapsed hunks in the current section.
// Hunks that were never collapsed by the LLM are not affected.
func (m *StoryModel) toggleAllCollapse() {
	if m.story == nil || len(m.story.Sections) == 0 {
		return
	}
	idx := m.codeSectionIndex()
	if idx < 0 || idx >= len(m.story.Sections) {
		return // On intro slide or invalid section
	}

	sectionHunks := m.story.Sections[idx].Hunks
	if len(sectionHunks) == 0 {
		return
	}

	// Only consider hunks that were originally collapsed by LLM
	var llmCollapsedKeys []hunkKey
	for _, ref := range sectionHunks {
		key := hunkKey{file: ref.File, hunkIndex: ref.HunkIndex}
		if m.llmCollapsedHunks[key] {
			llmCollapsedKeys = append(llmCollapsedKeys, key)
		}
	}

	if len(llmCollapsedKeys) == 0 {
		return // No LLM-collapsed hunks to toggle
	}

	// Count how many LLM-collapsed hunks are currently collapsed
	collapsedCount := 0
	for _, key := range llmCollapsedKeys {
		if m.collapsedHunks[key] {
			collapsedCount++
		}
	}

	// If more than half are collapsed, expand all; otherwise collapse all
	newState := collapsedCount <= len(llmCollapsedKeys)/2

	for _, key := range llmCollapsedKeys {
		m.collapsedHunks[key] = newState
	}

	// Re-render content
	m.viewport.SetContent(m.renderContent())
}

// gotoPrevSection switches to the previous section.
func (m *StoryModel) gotoPrevSection() {
	total := m.totalSections()
	if total == 0 {
		return
	}
	// Move to previous section if possible
	if m.activeSection > 0 {
		m.activeSection--
		m.viewport.SetContent(m.renderContent())
		m.viewport.GotoTop()
	}
}

func (m *StoryModel) saveCurrentCase() {
	if m.caseSaver == nil || m.caseSaverPath == "" || m.input == nil || m.story == nil {
		return
	}

	evalCase := diffview.EvalCase{
		Input: *m.input,
		Story: m.story,
	}
	// Best-effort save - errors are silently ignored in UI
	_ = m.caseSaver.Save(m.caseSaverPath, evalCase)
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
	hunkPositions, _, filePositions := m.computePositions()
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
		dimStyle.Render("j/k:scroll  s/S:section  z:toggle noise  e:save  q:quit") +
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
	total = m.totalSections()
	if total == 0 {
		return 0, 0, ""
	}

	current = m.activeSection + 1 // Convert 0-based to 1-based

	if m.onIntro() {
		title = "overview"
	} else {
		idx := m.codeSectionIndex()
		if idx >= 0 && idx < len(m.story.Sections) {
			title = m.story.Sections[idx].Title
		}
	}

	return current, total, title
}

// narrativeExplanation returns a human-readable explanation of the narrative pattern.
func narrativeExplanation(narrative string) string {
	switch narrative {
	case "cause-effect":
		return "problem → fix → proof"
	case "core-periphery":
		return "core change → ripple effects"
	case "before-after":
		return "old pattern → new pattern"
	case "entry-implementation":
		return "contract → implementation"
	case "rule-instances":
		return "pattern → applications"
	default:
		return ""
	}
}
