package bubbletea

import "github.com/charmbracelet/bubbles/key"

// EvalKeyMap defines the key bindings for the eval reviewer.
type EvalKeyMap struct {
	// Navigation
	NextCase     key.Binding
	PrevCase     key.Binding
	NextUnjudged key.Binding
	PrevUnjudged key.Binding

	// Panel switching
	TogglePanel key.Binding

	// Scrolling
	ScrollDown   key.Binding
	ScrollUp     key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding
	GotoTop      key.Binding
	GotoBottom   key.Binding

	// Hunk navigation (when diff panel active)
	NextHunk key.Binding
	PrevHunk key.Binding

	// Judgment
	Pass     key.Binding
	Fail     key.Binding
	Critique key.Binding

	// Critique mode
	ExitCritique key.Binding

	// Export
	CopyCase key.Binding

	// General
	Quit key.Binding
	Help key.Binding
}

// DefaultEvalKeyMap returns the default key bindings for the eval reviewer.
func DefaultEvalKeyMap() EvalKeyMap {
	return EvalKeyMap{
		NextCase: key.NewBinding(
			key.WithKeys("]"),
			key.WithHelp("]", "next case"),
		),
		PrevCase: key.NewBinding(
			key.WithKeys("["),
			key.WithHelp("[", "previous case"),
		),
		NextUnjudged: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "next unjudged"),
		),
		PrevUnjudged: key.NewBinding(
			key.WithKeys("U"),
			key.WithHelp("U", "previous unjudged"),
		),
		TogglePanel: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("Tab", "toggle panel"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j", "scroll down"),
		),
		ScrollUp: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k", "scroll up"),
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
			key.WithHelp("g", "go to top"),
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
		Pass: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "mark pass"),
		),
		Fail: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "mark fail"),
		),
		Critique: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "enter critique"),
		),
		ExitCritique: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "exit critique mode"),
		),
		CopyCase: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "copy case to clipboard"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}
