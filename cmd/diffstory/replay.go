package main

import (
	"errors"

	"github.com/fwojciec/diffview"
)

// ErrIndexOutOfBounds is returned when the requested case index is invalid.
var ErrIndexOutOfBounds = errors.New("case index out of bounds")

// ReplayApp loads a saved eval case for replay in the TUI.
type ReplayApp struct {
	Loader   diffview.EvalCaseLoader // Loader for JSONL files
	FilePath string                  // Path to JSONL file
	Index    int                     // Case index (0-based)
}

// Run loads the specified case and returns its diff and story.
func (a *ReplayApp) Run() (*diffview.Diff, *diffview.StoryClassification, error) {
	cases, err := a.Loader.Load(a.FilePath)
	if err != nil {
		return nil, nil, err
	}

	if a.Index < 0 || a.Index >= len(cases) {
		return nil, nil, ErrIndexOutOfBounds
	}

	evalCase := cases[a.Index]
	return &evalCase.Input.Diff, evalCase.Story, nil
}
