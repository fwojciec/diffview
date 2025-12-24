package bubbletea

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines the key bindings for the diff viewer.
type KeyMap struct {
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
}

// DefaultKeyMap returns the default vim-style key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
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
	}
}
