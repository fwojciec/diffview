package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fwojciec/diffview/bubbletea"
	"github.com/fwojciec/diffview/jsonl"
)

// ErrNoCases is returned when the input file contains no cases.
var ErrNoCases = errors.New("no cases to review")

// judgmentsPath returns the path for the judgments file given an input path.
// foo.jsonl -> foo-judgments.jsonl
func judgmentsPath(inputPath string) string {
	dir := filepath.Dir(inputPath)
	base := filepath.Base(inputPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	return filepath.Join(dir, name+"-judgments"+ext)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: evalreview <cases.jsonl>")
		os.Exit(1)
	}

	inputPath := os.Args[1]

	// Set up context with signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Load cases
	loader := jsonl.NewLoader()
	cases, err := loader.Load(inputPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error loading cases:", err)
		os.Exit(1)
	}

	if len(cases) == 0 {
		fmt.Fprintln(os.Stderr, ErrNoCases)
		os.Exit(1)
	}

	// Load existing judgments if any
	store := jsonl.NewStore()
	outputPath := judgmentsPath(inputPath)
	existingJudgments, err := store.Load(outputPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error loading judgments:", err)
		os.Exit(1)
	}

	// Create model with options
	opts := []bubbletea.EvalModelOption{
		bubbletea.WithJudgmentStore(store, outputPath),
	}
	if len(existingJudgments) > 0 {
		opts = append(opts, bubbletea.WithExistingJudgments(existingJudgments))
	}

	m := bubbletea.NewEvalModel(cases, opts...)

	// Run the TUI
	p := tea.NewProgram(m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithContext(ctx),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
