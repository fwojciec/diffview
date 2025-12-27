package jsonl

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fwojciec/diffview"
)

// Compile-time interface verification.
var _ diffview.JudgmentStore = (*Store)(nil)

// Store persists and retrieves Judgment records as JSONL.
type Store struct{}

// NewStore creates a new Store.
func NewStore() *Store {
	return &Store{}
}

// Load reads judgments from a JSONL file. Returns empty slice if file doesn't exist.
func (s *Store) Load(path string) ([]diffview.Judgment, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var judgments []diffview.Judgment
	scanner := bufio.NewScanner(f)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var j diffview.Judgment
		if err := json.Unmarshal([]byte(line), &j); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
		judgments = append(judgments, j)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return judgments, nil
}

// Save writes judgments to a JSONL file, creating parent directories if needed.
func (s *Store) Save(path string, judgments []diffview.Judgment) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, j := range judgments {
		data, err := json.Marshal(j)
		if err != nil {
			return err
		}
		if _, err := f.Write(data); err != nil {
			return err
		}
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}

	return nil
}
