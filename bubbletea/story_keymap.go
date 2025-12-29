package bubbletea

import "github.com/charmbracelet/bubbles/key"

// StoryKeyMap defines the key bindings for the story-aware diff viewer.
// It includes all standard navigation keys plus story-specific keys
// for section navigation and hunk collapsing.
type StoryKeyMap struct {
	// Standard navigation (inherited from KeyMap)
	Up           key.Binding
	Down         key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
	GotoTop      key.Binding
	GotoBottom   key.Binding
	NextHunk     key.Binding
	PrevHunk     key.Binding
	NextFile     key.Binding
	PrevFile     key.Binding
	Quit         key.Binding

	// Section navigation (story-specific)
	NextSection key.Binding
	PrevSection key.Binding

	// Hunk collapsing (story-specific)
	ToggleCollapse    key.Binding
	ToggleCollapseAll key.Binding

	// Export
	SaveCase key.Binding
}

// DefaultStoryKeyMap returns the default key bindings for story mode.
func DefaultStoryKeyMap() StoryKeyMap {
	return StoryKeyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/↑", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/↓", "down"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("ctrl+u", "half page up"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "half page down"),
		),
		GotoTop: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("gg", "go to top"),
		),
		GotoBottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "go to bottom"),
		),
		NextHunk: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next hunk"),
		),
		PrevHunk: key.NewBinding(
			key.WithKeys("N"),
			key.WithHelp("N", "previous hunk"),
		),
		NextFile: key.NewBinding(
			key.WithKeys("]"),
			key.WithHelp("]", "next file"),
		),
		PrevFile: key.NewBinding(
			key.WithKeys("["),
			key.WithHelp("[", "previous file"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		NextSection: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "next section"),
		),
		PrevSection: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "previous section"),
		),
		ToggleCollapse: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "toggle collapse"),
		),
		ToggleCollapseAll: key.NewBinding(
			key.WithKeys("z"),
			key.WithHelp("z", "toggle all collapsed"),
		),
		SaveCase: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "save case to eval dataset"),
		),
	}
}
