package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fwojciec/diffview"
	"github.com/fwojciec/diffview/bubbletea"
	"github.com/fwojciec/diffview/fs"
	"github.com/fwojciec/diffview/gemini"
	"github.com/fwojciec/diffview/gitdiff"
)

// ErrNoChanges is returned when the diff contains no changes to analyze.
var ErrNoChanges = errors.New("no changes to analyze")

// App encapsulates the application logic for testing.
type App struct {
	Input      io.Reader                // Read diff from stdin (if FilePath is empty)
	FilePath   string                   // Read diff from file (takes precedence over Input)
	Classifier diffview.StoryClassifier // Classifier for story generation
}

// Run parses the diff input and classifies it.
// Returns the parsed diff and classification for TUI display.
func (a *App) Run(ctx context.Context) (*diffview.Diff, *diffview.StoryClassification, error) {
	var input io.Reader
	if a.FilePath != "" {
		f, err := os.Open(a.FilePath)
		if err != nil {
			return nil, nil, err
		}
		defer f.Close()
		input = f
	} else {
		input = a.Input
	}

	parser := gitdiff.NewParser()
	diff, err := parser.Parse(input)
	if err != nil {
		return nil, nil, err
	}

	if len(diff.Files) == 0 {
		return nil, nil, ErrNoChanges
	}

	// Build classification input with the parsed diff
	classInput := diffview.ClassificationInput{
		Diff: *diff,
	}

	classification, err := a.Classifier.Classify(ctx, classInput)
	if err != nil {
		return nil, nil, err
	}

	return diff, classification, nil
}

// ErrNoInput is returned when no diff input is provided.
var ErrNoInput = errors.New("no input: pipe a diff or provide a file path")

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Check for API key
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("GEMINI_API_KEY environment variable required")
	}

	// Set up Gemini client and classifier
	client, err := gemini.NewClient(ctx, apiKey)
	if err != nil {
		return fmt.Errorf("failed to create Gemini client: %w", err)
	}
	defer client.Close()

	geminiClassifier := gemini.NewClassifier(client, gemini.DefaultModel)
	classifier := fs.NewClassifier(geminiClassifier, fs.DefaultCacheDir())

	app := &App{
		Classifier: classifier,
	}

	// Check for file path argument
	if len(os.Args) >= 2 {
		app.FilePath = os.Args[1]
	} else {
		// Check if stdin is a pipe
		stat, err := os.Stdin.Stat()
		if err != nil {
			return fmt.Errorf("error checking stdin: %w", err)
		}
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			return ErrNoInput
		}
		app.Input = os.Stdin
	}

	diff, classification, err := app.Run(ctx)
	if err != nil {
		return err
	}

	// Launch StoryModel TUI
	m := bubbletea.NewStoryModel(diff, classification)
	p := tea.NewProgram(m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithContext(ctx),
	)

	_, err = p.Run()
	return err
}
