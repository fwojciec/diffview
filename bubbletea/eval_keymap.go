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
	DiffPanel  key.Binding
	StoryPanel key.Binding

	// Scrolling
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
}

// DefaultEvalKeyMap returns the default key bindings for the eval reviewer.
func DefaultEvalKeyMap() EvalKeyMap {
	return EvalKeyMap{
		NextCase: key.NewBinding(
			key.WithKeys("j"),
			key.WithHelp("j", "next case"),
		),
		PrevCase: key.NewBinding(
			key.WithKeys("k"),
			key.WithHelp("k", "previous case"),
		),
		NextUnjudged: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "next unjudged"),
		),
		PrevUnjudged: key.NewBinding(
			key.WithKeys("U"),
			key.WithHelp("U", "previous unjudged"),
		),
		DiffPanel: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "focus diff panel"),
		),
		StoryPanel: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "focus story panel"),
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
	}
}
