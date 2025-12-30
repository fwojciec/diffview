package bubbletea

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fwojciec/diffstory"
)

// Mode identifies the current interaction mode.
type Mode int

// Mode constants.
const (
	ModeReview Mode = iota
	ModeCritique
	ModeHelp
)

// EvalModel is the Bubble Tea model for evaluating diff stories.
type EvalModel struct {
	// Data
	cases        []diffview.EvalCase
	judgments    map[string]*diffview.Judgment
	currentIndex int

	// UI Components
	diffViewport     viewport.Model
	storyViewport    viewport.Model
	critiqueTextarea textarea.Model

	// State
	mode  Mode
	ready bool

	// Story mode state
	storyMode        bool               // true = section-by-section navigation, false = raw diff
	activeSection    int                // current section index (0-based)
	reviewedSections map[int]bool       // sections marked as reviewed for current case
	collapsedHunks   map[hunkKey]bool   // hunk collapse state
	hunkCategories   map[hunkKey]string // hunk → category for styling
	collapseText     map[hunkKey]string // hunk → collapse text
	splitRatio       int                // percentage of height for metadata pane (0-100)

	// Rendering
	width, height    int
	styles           diffview.Styles
	languageDetector diffview.LanguageDetector
	tokenizer        diffview.Tokenizer
	wordDiffer       diffview.WordDiffer

	// Persistence
	store      diffview.JudgmentStore
	outputPath string

	// Clipboard
	clipboard diffview.Clipboard

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
			m.judgments[j.CaseID] = &j
		}
	}
}

// WithEvalStyles sets the diff rendering styles.
func WithEvalStyles(s diffview.Styles) EvalModelOption {
	return func(m *EvalModel) {
		m.styles = s
	}
}

// WithEvalLanguageDetector sets the language detector for syntax highlighting.
func WithEvalLanguageDetector(d diffview.LanguageDetector) EvalModelOption {
	return func(m *EvalModel) {
		m.languageDetector = d
	}
}

// WithEvalTokenizer sets the tokenizer for syntax highlighting.
func WithEvalTokenizer(t diffview.Tokenizer) EvalModelOption {
	return func(m *EvalModel) {
		m.tokenizer = t
	}
}

// WithEvalWordDiffer sets the word differ for word-level highlighting.
func WithEvalWordDiffer(d diffview.WordDiffer) EvalModelOption {
	return func(m *EvalModel) {
		m.wordDiffer = d
	}
}

// WithClipboard sets the clipboard for copy operations.
func WithClipboard(c diffview.Clipboard) EvalModelOption {
	return func(m *EvalModel) {
		m.clipboard = c
	}
}

// NewEvalModel creates a new EvalModel with the given cases.
func NewEvalModel(cases []diffview.EvalCase, opts ...EvalModelOption) EvalModel {
	m := EvalModel{
		cases:            cases,
		judgments:        make(map[string]*diffview.Judgment),
		mode:             ModeReview,
		keymap:           DefaultEvalKeyMap(),
		styles:           defaultStyles(), // Use same defaults as viewer
		reviewedSections: make(map[int]bool),
		collapsedHunks:   make(map[hunkKey]bool),
		hunkCategories:   make(map[hunkKey]string),
		collapseText:     make(map[hunkKey]string),
		splitRatio:       30, // 30% metadata, 70% diff by default
	}

	for _, opt := range opts {
		opt(&m)
	}

	// Enable story mode by default if first case has sections
	if len(cases) > 0 && cases[0].Story != nil && len(cases[0].Story.Sections) > 0 {
		m.storyMode = true
		m.rebuildStoryMaps()
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
		switch m.mode {
		case ModeReview:
			return m.handleReviewKeys(msg)
		case ModeCritique:
			return m.handleCritiqueKeys(msg)
		case ModeHelp:
			return m.handleHelpKeys(msg)
		}

	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)
	}

	// Update the diff viewport
	var cmd tea.Cmd
	m.diffViewport, cmd = m.diffViewport.Update(msg)
	return m, cmd
}

func (m EvalModel) handleReviewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keymap.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keymap.NextCase):
		if m.currentIndex < len(m.cases)-1 {
			m.currentIndex++
			m.rebuildStoryMaps()
			m.updateStoryModeForCase()
			m.updateViewportContent()
		}
		return m, nil

	case key.Matches(msg, m.keymap.PrevCase):
		if m.currentIndex > 0 {
			m.currentIndex--
			m.rebuildStoryMaps()
			m.updateStoryModeForCase()
			m.updateViewportContent()
		}
		return m, nil

	case key.Matches(msg, m.keymap.NextUnjudged):
		if idx := m.findNextUnjudged(); idx != -1 && idx != m.currentIndex {
			m.currentIndex = idx
			m.rebuildStoryMaps()
			m.updateStoryModeForCase()
			m.updateViewportContent()
		}
		return m, nil

	case key.Matches(msg, m.keymap.PrevUnjudged):
		if idx := m.findPrevUnjudged(); idx != -1 && idx != m.currentIndex {
			m.currentIndex = idx
			m.rebuildStoryMaps()
			m.updateStoryModeForCase()
			m.updateViewportContent()
		}
		return m, nil

	case key.Matches(msg, m.keymap.ScrollDown):
		m.diffViewport.ScrollDown(1)
		return m, nil

	case key.Matches(msg, m.keymap.ScrollUp):
		m.diffViewport.ScrollUp(1)
		return m, nil

	case key.Matches(msg, m.keymap.HalfPageUp):
		m.diffViewport.HalfPageUp()
		return m, nil

	case key.Matches(msg, m.keymap.HalfPageDown):
		m.diffViewport.HalfPageDown()
		return m, nil

	case key.Matches(msg, m.keymap.GotoTop):
		m.diffViewport.GotoTop()
		return m, nil

	case key.Matches(msg, m.keymap.GotoBottom):
		m.diffViewport.GotoBottom()
		return m, nil

	case key.Matches(msg, m.keymap.ToggleMode):
		m.toggleStoryMode()
		return m, nil

	case key.Matches(msg, m.keymap.NextSection):
		if m.storyMode {
			m.gotoNextSection()
		}
		return m, nil

	case key.Matches(msg, m.keymap.PrevSection):
		if m.storyMode {
			m.gotoPrevSection()
		}
		return m, nil

	case key.Matches(msg, m.keymap.IncreaseSplit):
		m.adjustSplit(10)
		return m, nil

	case key.Matches(msg, m.keymap.DecreaseSplit):
		m.adjustSplit(-10)
		return m, nil

	case key.Matches(msg, m.keymap.Pass):
		m.recordJudgment(true)
		return m, nil

	case key.Matches(msg, m.keymap.Fail):
		m.recordJudgment(false)
		return m, nil

	case key.Matches(msg, m.keymap.Critique):
		return m.enterCritiqueMode()

	case key.Matches(msg, m.keymap.CopyCase):
		m.copyCurrentCase()
		return m, nil

	case key.Matches(msg, m.keymap.Help):
		m.mode = ModeHelp
		return m, nil
	}

	return m, nil
}

func (m EvalModel) handleCritiqueKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keymap.ExitCritique):
		return m.exitCritiqueMode()
	}

	// Pass all other keys to textarea
	var cmd tea.Cmd
	m.critiqueTextarea, cmd = m.critiqueTextarea.Update(msg)
	return m, cmd
}

func (m EvalModel) handleHelpKeys(_ tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Any key dismisses help
	m.mode = ModeReview
	return m, nil
}

func (m EvalModel) enterCritiqueMode() (tea.Model, tea.Cmd) {
	if len(m.cases) == 0 {
		return m, nil
	}

	// Initialize textarea with existing critique if any
	ta := textarea.New()
	ta.Placeholder = "Enter detailed critique..."
	ta.ShowLineNumbers = false
	ta.SetWidth(m.width - 4)
	ta.SetHeight(m.height - 6)

	c := m.cases[m.currentIndex]
	if j := m.judgments[c.Input.CaseID()]; j != nil && j.Critique != "" {
		ta.SetValue(j.Critique)
	}

	ta.Focus()
	m.critiqueTextarea = ta
	m.mode = ModeCritique

	return m, textarea.Blink
}

func (m EvalModel) exitCritiqueMode() (tea.Model, tea.Cmd) {
	// Save critique to judgment
	if len(m.cases) > 0 {
		c := m.cases[m.currentIndex]
		caseID := c.Input.CaseID()
		critique := m.critiqueTextarea.Value()

		// Get or create judgment
		j := m.judgments[caseID]
		if j == nil {
			j = &diffview.Judgment{
				CaseID:   caseID,
				Index:    m.currentIndex,
				JudgedAt: time.Now(),
			}
			m.judgments[caseID] = j
		}
		j.Critique = critique
		j.JudgedAt = time.Now()

		m.persistJudgments()
	}

	m.mode = ModeReview
	return m, nil
}

func (m *EvalModel) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	// Calculate panel heights using split ratio
	// Reserve: DIFF header (1), STORY header (1), judgment bar (1), status bar (1) = 4
	// Plus newlines after each viewport (2) = 6 total reserved
	usableHeight := msg.Height - 6
	if usableHeight < 2 {
		usableHeight = 2 // Minimum height for tiny terminals
	}
	metadataHeight := usableHeight * m.splitRatio / 100
	diffHeight := usableHeight - metadataHeight

	if !m.ready {
		m.diffViewport = viewport.New(msg.Width, diffHeight)
		m.storyViewport = viewport.New(msg.Width, metadataHeight)
		m.updateViewportContent()
		m.ready = true
	} else {
		m.diffViewport.Width = msg.Width
		m.diffViewport.Height = diffHeight
		m.storyViewport.Width = msg.Width
		m.storyViewport.Height = metadataHeight
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

	// Use filtered diff in story mode, full diff otherwise
	diffToRender, originalIndices := m.filteredDiffWithIndices()

	// Render diff content using styled renderer
	diffContent := renderDiff(renderConfig{
		diff:             diffToRender,
		styles:           m.styles,
		renderer:         nil, // Use default renderer
		width:            m.width,
		languageDetector: m.languageDetector,
		tokenizer:        m.tokenizer,
		wordDiffer:       m.wordDiffer,
		collapsedHunks:   m.collapsedHunks,
		hunkCategories:   m.hunkCategories,
		collapseText:     m.collapseText,
		originalIndices:  originalIndices,
	})

	m.diffViewport.SetContent(diffContent)
	m.diffViewport.GotoTop()

	// Render metadata content based on mode
	var metadataContent strings.Builder
	if m.storyMode && c.Story != nil && m.activeSection < len(c.Story.Sections) {
		// Story mode: show section-level metadata only
		section := c.Story.Sections[m.activeSection]
		metadataContent.WriteString(m.renderSectionHeader(section))
	} else if c.Story != nil {
		// Raw mode: show full classification tree
		metadataContent.WriteString(fmt.Sprintf("[%s] %s\n", c.Story.ChangeType, c.Story.Narrative))
		metadataContent.WriteString(fmt.Sprintf("%s\n\n", c.Story.Summary))
		for _, section := range c.Story.Sections {
			metadataContent.WriteString(fmt.Sprintf("• %s: %s\n", section.Role, section.Title))
			metadataContent.WriteString(fmt.Sprintf("  %s\n", section.Explanation))
			if len(section.Hunks) > 0 {
				var hunkRefs []string
				for _, h := range section.Hunks {
					hunkRefs = append(hunkRefs, fmt.Sprintf("%s:H%d", h.File, h.HunkIndex))
				}
				metadataContent.WriteString(fmt.Sprintf("  hunks: %s\n", strings.Join(hunkRefs, ", ")))
			}
		}
	} else {
		metadataContent.WriteString("[Not yet classified]")
	}

	// Add critique if present (full text, not truncated)
	if j := m.judgments[c.Input.CaseID()]; j != nil && j.Critique != "" {
		metadataContent.WriteString("\n\nCRITIQUE:\n")
		metadataContent.WriteString(j.Critique)
	}

	m.storyViewport.SetContent(metadataContent.String())
	m.storyViewport.GotoTop()
}

func (m *EvalModel) recordJudgment(pass bool) {
	if len(m.cases) == 0 {
		return
	}

	c := m.cases[m.currentIndex]
	caseID := c.Input.CaseID()

	// Preserve existing critique when toggling pass/fail
	var critique string
	if existing := m.judgments[caseID]; existing != nil {
		critique = existing.Critique
	}

	j := &diffview.Judgment{
		CaseID:   caseID,
		Index:    m.currentIndex,
		Judged:   true,
		Pass:     pass,
		Critique: critique,
		JudgedAt: time.Now(),
	}
	m.judgments[caseID] = j

	m.persistJudgments()
}

// isUnjudged returns true if the case at the given index hasn't been judged.
func (m EvalModel) isUnjudged(idx int) bool {
	if idx < 0 || idx >= len(m.cases) {
		return false
	}
	j := m.judgments[m.cases[idx].Input.CaseID()]
	return j == nil || !j.Judged
}

// findNextUnjudged returns the index of the next unjudged case, wrapping around.
// Returns -1 if no unjudged cases exist.
func (m EvalModel) findNextUnjudged() int {
	n := len(m.cases)
	if n == 0 {
		return -1
	}
	// Search from current+1 to end, then from start to current
	for i := 1; i <= n; i++ {
		idx := (m.currentIndex + i) % n
		if m.isUnjudged(idx) {
			return idx
		}
	}
	return -1
}

// findPrevUnjudged returns the index of the previous unjudged case, wrapping around.
// Returns -1 if no unjudged cases exist.
func (m EvalModel) findPrevUnjudged() int {
	n := len(m.cases)
	if n == 0 {
		return -1
	}
	// Search backwards from current-1 to start, then from end to current
	for i := 1; i <= n; i++ {
		idx := (m.currentIndex - i + n) % n
		if m.isUnjudged(idx) {
			return idx
		}
	}
	return -1
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

func (m *EvalModel) copyCurrentCase() {
	if m.clipboard == nil || len(m.cases) == 0 {
		return
	}

	c := m.cases[m.currentIndex]
	content := formatCaseForExport(c)
	// Best-effort copy - errors are silently ignored in UI
	_ = m.clipboard.Copy(content)
}

// rebuildStoryMaps rebuilds the hunk maps from the current case's story.
// Call this when switching cases or when story mode is enabled.
func (m *EvalModel) rebuildStoryMaps() {
	// Clear existing maps
	m.collapsedHunks = make(map[hunkKey]bool)
	m.hunkCategories = make(map[hunkKey]string)
	m.collapseText = make(map[hunkKey]string)
	m.reviewedSections = make(map[int]bool)
	m.activeSection = 0

	if len(m.cases) == 0 {
		return
	}

	c := m.cases[m.currentIndex]
	if c.Story == nil {
		return
	}

	// Build lookup maps from story classification
	for _, section := range c.Story.Sections {
		for _, ref := range section.Hunks {
			key := hunkKey{file: ref.File, hunkIndex: ref.HunkIndex}
			m.hunkCategories[key] = ref.Category
			if ref.CollapseText != "" {
				m.collapseText[key] = ref.CollapseText
			}
			// Collapse if explicitly marked or noise category
			if ref.Collapsed || ref.Category == "noise" {
				m.collapsedHunks[key] = true
			}
		}
	}
}

// toggleStoryMode toggles between story mode and raw mode.
// Story mode is only available when the current case has sections.
func (m *EvalModel) toggleStoryMode() {
	if len(m.cases) == 0 {
		return
	}

	c := m.cases[m.currentIndex]
	// Only allow story mode if the case has sections
	if c.Story == nil || len(c.Story.Sections) == 0 {
		m.storyMode = false
		return
	}

	m.storyMode = !m.storyMode
	if m.storyMode {
		m.rebuildStoryMaps()
	}
	m.updateViewportContent()
}

// updateStoryModeForCase updates story mode based on the current case.
// Enables story mode if the case has sections, disables if it doesn't.
func (m *EvalModel) updateStoryModeForCase() {
	if len(m.cases) == 0 {
		m.storyMode = false
		return
	}

	c := m.cases[m.currentIndex]
	// Enable story mode if the case has sections
	m.storyMode = c.Story != nil && len(c.Story.Sections) > 0
}

// gotoNextSection moves to the next section and marks the current one as reviewed.
func (m *EvalModel) gotoNextSection() {
	if len(m.cases) == 0 {
		return
	}

	c := m.cases[m.currentIndex]
	if c.Story == nil || len(c.Story.Sections) == 0 {
		return
	}

	// Mark current section as reviewed
	m.reviewedSections[m.activeSection] = true

	// Move to next section if not at end
	if m.activeSection < len(c.Story.Sections)-1 {
		m.activeSection++
		m.updateViewportContent()
		m.diffViewport.GotoTop()
	}
}

// gotoPrevSection moves to the previous section.
func (m *EvalModel) gotoPrevSection() {
	if len(m.cases) == 0 {
		return
	}

	c := m.cases[m.currentIndex]
	if c.Story == nil || len(c.Story.Sections) == 0 {
		return
	}

	// Move to previous section if not at start
	if m.activeSection > 0 {
		m.activeSection--
		m.updateViewportContent()
		m.diffViewport.GotoTop()
	}
}

// adjustSplit adjusts the split ratio by the given delta (positive = more metadata).
// Clamps the ratio between 10% and 90%.
func (m *EvalModel) adjustSplit(delta int) {
	m.splitRatio += delta
	if m.splitRatio < 10 {
		m.splitRatio = 10
	}
	if m.splitRatio > 90 {
		m.splitRatio = 90
	}
	m.recalculateViewportSizes()
}

// recalculateViewportSizes updates viewport dimensions based on current split ratio.
func (m *EvalModel) recalculateViewportSizes() {
	if !m.ready || m.height == 0 {
		return
	}
	// Reserve: DIFF header (1), STORY header (1), judgment bar (1), status bar (1) = 4
	// Plus newlines after each viewport (2) = 6 total reserved
	usableHeight := m.height - 6
	if usableHeight < 2 {
		usableHeight = 2
	}
	metadataHeight := usableHeight * m.splitRatio / 100
	diffHeight := usableHeight - metadataHeight

	m.storyViewport.Width = m.width
	m.storyViewport.Height = metadataHeight
	m.diffViewport.Width = m.width
	m.diffViewport.Height = diffHeight
}

// renderSectionHeader formats the section header for display in the diff panel.
func (m *EvalModel) renderSectionHeader(section diffview.Section) string {
	header := fmt.Sprintf("[%s] %s", section.Role, section.Title)
	if section.Explanation != "" {
		header += "\n" + section.Explanation
	}
	return header
}

// renderSectionProgress returns a string of indicators showing section review status.
// ✓ = reviewed, ● = current, ○ = pending
func (m *EvalModel) renderSectionProgress(sections []diffview.Section) string {
	var indicators []string
	for i := range sections {
		if m.reviewedSections[i] {
			indicators = append(indicators, "✓")
		} else if i == m.activeSection {
			indicators = append(indicators, "●")
		} else {
			indicators = append(indicators, "○")
		}
	}
	return strings.Join(indicators, " ")
}

// filteredDiffWithIndices returns a diff containing only hunks from the active section,
// along with a mapping from (file, filtered position) to original hunk index.
// If not in story mode or no sections exist, returns the full diff with nil indices.
func (m *EvalModel) filteredDiffWithIndices() (*diffview.Diff, map[hunkKey]int) {
	if len(m.cases) == 0 {
		return nil, nil
	}

	c := m.cases[m.currentIndex]
	diff := &c.Input.Diff

	// Return full diff if not in story mode or no sections
	if !m.storyMode || c.Story == nil || len(c.Story.Sections) == 0 {
		return diff, nil
	}

	// Validate active section index
	if m.activeSection < 0 || m.activeSection >= len(c.Story.Sections) {
		return diff, nil
	}

	// Build a set of hunks in the active section
	section := c.Story.Sections[m.activeSection]
	activeHunks := make(map[hunkKey]bool, len(section.Hunks))
	for _, ref := range section.Hunks {
		activeHunks[hunkKey{file: ref.File, hunkIndex: ref.HunkIndex}] = true
	}

	// Create filtered diff with only files/hunks from active section
	// Also build mapping from filtered position to original index
	originalIndices := make(map[hunkKey]int)
	var filteredFiles []diffview.FileDiff
	for _, file := range diff.Files {
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

// formatCaseForExport formats an EvalCase as markdown for LLM-assisted review.
func formatCaseForExport(c diffview.EvalCase) string {
	var sb strings.Builder

	sb.WriteString("# Diff Classification Review\n\n")

	// Input section: raw diff
	sb.WriteString("## Input: Raw Diff\n\n")
	sb.WriteString(fmt.Sprintf("Repository: %s\n", c.Input.Repo))
	if c.Input.Branch != "" {
		sb.WriteString(fmt.Sprintf("Branch: %s\n", c.Input.Branch))
	}
	if len(c.Input.Commits) > 0 {
		sb.WriteString("\nCommits:\n")
		for _, commit := range c.Input.Commits {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", commit.Hash, commit.Message))
		}
	}
	sb.WriteString("\n```diff\n")
	for _, file := range c.Input.Diff.Files {
		sb.WriteString(fmt.Sprintf("=== %s (%s) ===\n", formatFilePath(file), formatFileOp(file.Operation)))
		for _, hunk := range file.Hunks {
			sb.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n",
				hunk.OldStart, hunk.OldCount, hunk.NewStart, hunk.NewCount))
			for _, line := range hunk.Lines {
				prefix := formatLinePrefix(line.Type)
				sb.WriteString(prefix)
				sb.WriteString(line.Content)
				if !strings.HasSuffix(line.Content, "\n") {
					sb.WriteString("\n")
				}
			}
		}
	}
	sb.WriteString("```\n\n")

	// Output section: story classification
	sb.WriteString("## Output: Story Classification\n\n")
	if c.Story != nil {
		sb.WriteString(fmt.Sprintf("Change Type: %s\n", c.Story.ChangeType))
		sb.WriteString(fmt.Sprintf("Narrative: %s\n", c.Story.Narrative))
		sb.WriteString(fmt.Sprintf("Summary: %s\n\n", c.Story.Summary))

		if len(c.Story.Sections) > 0 {
			sb.WriteString("Sections:\n")
			for i, section := range c.Story.Sections {
				sb.WriteString(fmt.Sprintf("%d. [%s]: %s\n", i+1, section.Role, section.Title))
				sb.WriteString(fmt.Sprintf("   %s\n", section.Explanation))
				if len(section.Hunks) > 0 {
					var hunkRefs []string
					for _, h := range section.Hunks {
						hunkRefs = append(hunkRefs, fmt.Sprintf("%s:H%d", h.File, h.HunkIndex))
					}
					sb.WriteString(fmt.Sprintf("   Hunks: %s\n", strings.Join(hunkRefs, ", ")))
				}
			}
		}
	} else {
		sb.WriteString("[Not yet classified]\n")
	}

	// Task section: LLM review prompt
	sb.WriteString("\n## Your Task\n\n")
	sb.WriteString("Evaluate whether this classification accurately captures the diff:\n")
	sb.WriteString("- Does the narrative match the actual changes?\n")
	sb.WriteString("- Are hunks assigned to appropriate sections?\n")
	sb.WriteString("- Is anything miscategorized or missing?\n")
	sb.WriteString("- Should this pass or fail? Why?\n")

	return sb.String()
}

func formatFilePath(file diffview.FileDiff) string {
	if file.NewPath != "" {
		return file.NewPath
	}
	return file.OldPath
}

func formatFileOp(op diffview.FileOp) string {
	switch op {
	case diffview.FileAdded:
		return "added"
	case diffview.FileDeleted:
		return "deleted"
	case diffview.FileModified:
		return "modified"
	case diffview.FileRenamed:
		return "renamed"
	case diffview.FileCopied:
		return "copied"
	default:
		return "modified"
	}
}

func formatLinePrefix(lt diffview.LineType) string {
	switch lt {
	case diffview.LineAdded:
		return "+"
	case diffview.LineDeleted:
		return "-"
	default:
		return " "
	}
}

// View implements tea.Model.
func (m EvalModel) View() string {
	if !m.ready {
		return "Loading..."
	}

	// Critique mode shows full-screen textarea
	if m.mode == ModeCritique {
		return m.renderCritiqueView()
	}

	// Help mode shows keybinding overlay
	if m.mode == ModeHelp {
		return m.renderHelpView()
	}

	var s strings.Builder

	// Metadata panel (top) - section info in story mode, classification tree in raw mode
	panelName := "STORY"
	if m.storyMode {
		panelName = "SECTION"
	}
	s.WriteString(m.renderPanelHeader(panelName))
	s.WriteString("\n")
	s.WriteString(m.storyViewport.View())
	s.WriteString("\n")

	// Diff panel (bottom) - filtered hunks in story mode, full diff in raw mode
	s.WriteString(m.renderPanelHeader("DIFF"))
	s.WriteString("\n")
	s.WriteString(m.diffViewport.View())
	s.WriteString("\n")

	// Judgment bar
	s.WriteString(m.renderJudgmentBar())
	s.WriteString("\n")

	// Status bar
	s.WriteString(m.renderStatusBar())

	return s.String()
}

func (m EvalModel) renderCritiqueView() string {
	var s strings.Builder

	header := lipgloss.NewStyle().Bold(true).Render("CRITIQUE")
	s.WriteString(header)
	s.WriteString("\n\n")
	s.WriteString(m.critiqueTextarea.View())
	s.WriteString("\n\n")
	s.WriteString(lipgloss.NewStyle().Faint(true).Render("[Esc] save and exit"))

	return s.String()
}

func (m EvalModel) renderHelpView() string {
	var s strings.Builder

	headerStyle := lipgloss.NewStyle().Bold(true)
	keyStyle := lipgloss.NewStyle().Bold(true)
	descStyle := lipgloss.NewStyle().Faint(true)

	s.WriteString(headerStyle.Render("HELP"))
	s.WriteString("\n\n")

	// Navigation
	s.WriteString(headerStyle.Render("Navigation"))
	s.WriteString("\n")
	s.WriteString(fmt.Sprintf("  %s  %s\n", keyStyle.Render("n/N"), descStyle.Render("next/previous case")))
	s.WriteString(fmt.Sprintf("  %s  %s\n", keyStyle.Render("u/U"), descStyle.Render("next/previous unjudged")))
	s.WriteString("\n")

	// Scrolling
	s.WriteString(headerStyle.Render("Scrolling"))
	s.WriteString("\n")
	s.WriteString(fmt.Sprintf("  %s  %s\n", keyStyle.Render("j/k"), descStyle.Render("scroll down/up")))
	s.WriteString(fmt.Sprintf("  %s  %s\n", keyStyle.Render("ctrl+d/u"), descStyle.Render("half page down/up")))
	s.WriteString(fmt.Sprintf("  %s  %s\n", keyStyle.Render("g/G"), descStyle.Render("go to top/bottom")))
	s.WriteString("\n")

	// View
	s.WriteString(headerStyle.Render("View"))
	s.WriteString("\n")
	s.WriteString(fmt.Sprintf("  %s  %s\n", keyStyle.Render("=/+/-"), descStyle.Render("resize split")))
	s.WriteString(fmt.Sprintf("  %s    %s\n", keyStyle.Render("m"), descStyle.Render("toggle story/raw mode")))
	s.WriteString(fmt.Sprintf("  %s  %s\n", keyStyle.Render("]/["), descStyle.Render("next/prev section (story mode)")))
	s.WriteString("\n")

	// Judgment
	s.WriteString(headerStyle.Render("Judgment"))
	s.WriteString("\n")
	s.WriteString(fmt.Sprintf("  %s    %s\n", keyStyle.Render("p"), descStyle.Render("mark pass")))
	s.WriteString(fmt.Sprintf("  %s    %s\n", keyStyle.Render("f"), descStyle.Render("mark fail")))
	s.WriteString(fmt.Sprintf("  %s    %s\n", keyStyle.Render("c"), descStyle.Render("enter critique")))
	s.WriteString("\n")

	// Other
	s.WriteString(headerStyle.Render("Other"))
	s.WriteString("\n")
	s.WriteString(fmt.Sprintf("  %s    %s\n", keyStyle.Render("y"), descStyle.Render("copy case to clipboard")))
	s.WriteString(fmt.Sprintf("  %s    %s\n", keyStyle.Render("?"), descStyle.Render("toggle help")))
	s.WriteString(fmt.Sprintf("  %s    %s\n", keyStyle.Render("q"), descStyle.Render("quit")))
	s.WriteString("\n\n")

	s.WriteString(descStyle.Render("Press any key to close"))

	return s.String()
}

func (m EvalModel) renderPanelHeader(name string) string {
	style := lipgloss.NewStyle().Bold(true)
	return style.Render(name)
}

func (m EvalModel) renderJudgmentBar() string {
	if len(m.cases) == 0 {
		return ""
	}

	c := m.cases[m.currentIndex]
	j := m.judgments[c.Input.CaseID()]

	passMarker := "○"
	failMarker := "○"
	critique := "[not set]"

	if j != nil {
		if j.Judged {
			if j.Pass {
				passMarker = "●"
			} else {
				failMarker = "●"
			}
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

// RenderDataView formats the classification as a structured tree for data view.
// It shows change_type, narrative, summary at top, followed by sections with
// their role, explanation, and hunk references.
// The width parameter is reserved for future text wrapping of long content.
func RenderDataView(story *diffview.StoryClassification, width int) string {
	if story == nil {
		return "[Not yet classified]"
	}

	var s strings.Builder

	// Classification metadata
	s.WriteString(fmt.Sprintf("change_type: %s\n", story.ChangeType))
	s.WriteString(fmt.Sprintf("narrative:   %s\n", story.Narrative))
	s.WriteString(fmt.Sprintf("summary:     %s\n", story.Summary))
	s.WriteString("\n")

	// Sections header
	s.WriteString("── sections ──────────────────────────────────────────\n")
	s.WriteString("\n")

	// Each section
	for _, section := range story.Sections {
		s.WriteString(fmt.Sprintf("[%s] %s\n", section.Role, section.Title))
		s.WriteString(fmt.Sprintf("  explanation: %s\n", section.Explanation))
		if len(section.Hunks) > 0 {
			s.WriteString("  hunks:\n")
			for _, h := range section.Hunks {
				state := "visible"
				if h.Collapsed {
					state = "collapsed"
				}
				s.WriteString(fmt.Sprintf("    %s:H%d    %s      %s\n", h.File, h.HunkIndex, h.Category, state))
			}
		}
		s.WriteString("\n")
	}

	return s.String()
}

func (m EvalModel) renderStatusBar() string {
	if len(m.cases) == 0 {
		return "No cases"
	}

	// Count judged cases and build indicator string
	judged := 0
	var indicators []string
	for _, c := range m.cases {
		j, ok := m.judgments[c.Input.CaseID()]
		if !ok {
			indicators = append(indicators, "○") // unjudged
		} else if !j.Judged {
			// Has judgment record but not explicitly passed/failed
			indicators = append(indicators, "●") // partial-judgment
		} else {
			judged++
			if j.Pass {
				indicators = append(indicators, "✓") // pass
			} else {
				indicators = append(indicators, "✗") // fail
			}
		}
	}

	caseInfo := fmt.Sprintf("case %d/%d", m.currentIndex+1, len(m.cases))
	progress := fmt.Sprintf("%d/%d reviewed", judged, len(m.cases))
	indicatorBar := strings.Join(indicators, " ")

	// Mode and section indicator
	var modeInfo string
	var sectionProgress string
	if m.storyMode {
		c := m.cases[m.currentIndex]
		if c.Story != nil && len(c.Story.Sections) > 0 {
			modeInfo = fmt.Sprintf("story mode │ section %d/%d", m.activeSection+1, len(c.Story.Sections))
			sectionProgress = m.renderSectionProgress(c.Story.Sections)
		} else {
			modeInfo = "story mode"
		}
	} else {
		modeInfo = "raw mode"
	}

	help := "[p]ass [f]ail [c]ritique [y]ank n/N case ]/[ section [?]help [q]uit"

	if sectionProgress != "" {
		return fmt.Sprintf("%s │ %s │ %s │ %s │ %s │ %s", modeInfo, sectionProgress, caseInfo, progress, indicatorBar, help)
	}
	return fmt.Sprintf("%s │ %s │ %s │ %s │ %s", modeInfo, caseInfo, progress, indicatorBar, help)
}
