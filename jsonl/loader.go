// Package jsonl provides JSONL file handling for eval cases and judgments.
package jsonl

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fwojciec/diffview"
)

// Compile-time interface verification.
var _ diffview.EvalCaseLoader = (*Loader)(nil)

// Loader loads EvalCase records from JSONL files.
type Loader struct{}

// NewLoader creates a new Loader.
func NewLoader() *Loader {
	return &Loader{}
}

// maxLineSize is the maximum size for a single JSONL line (4MB).
// This accommodates large PR-level diffs while preventing memory issues.
const maxLineSize = 4 * 1024 * 1024

// Load reads a JSONL file and returns all EvalCase records.
func (l *Loader) Load(path string) ([]diffview.EvalCase, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cases []diffview.EvalCase
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, maxLineSize), maxLineSize)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var c diffview.EvalCase
		if err := json.Unmarshal([]byte(line), &c); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
		cases = append(cases, c)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return cases, nil
}
