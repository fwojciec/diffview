package jsonl

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/fwojciec/diffview"
)

// Compile-time interface verification.
var _ diffview.EvalCaseSaver = (*Saver)(nil)

// Saver appends EvalCase records to JSONL files.
type Saver struct{}

// NewSaver creates a new Saver.
func NewSaver() *Saver {
	return &Saver{}
}

// Save appends an EvalCase to a JSONL file, creating parent directories if needed.
func (s *Saver) Save(path string, c diffview.EvalCase) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		return err
	}
	if _, err := f.WriteString("\n"); err != nil {
		return err
	}

	return nil
}
