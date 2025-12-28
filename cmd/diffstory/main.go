package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

// spinner displays a progress indicator on stderr while a long-running operation executes.
type spinner struct {
	frames   []string
	interval time.Duration
	message  string
	w        io.Writer
	stop     chan struct{}
	done     chan struct{}
}

// newSpinner creates a spinner that writes to the given writer.
func newSpinner(w io.Writer, message string) *spinner {
	return &spinner{
		frames:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		interval: 80 * time.Millisecond,
		message:  message,
		w:        w,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// Start begins the spinner animation in a goroutine.
func (s *spinner) Start() {
	go func() {
		defer close(s.done)
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		frame := 0
		// Print initial frame immediately
		fmt.Fprintf(s.w, "\r%s %s", s.frames[frame], s.message)
		frame = (frame + 1) % len(s.frames)

		for {
			select {
			case <-s.stop:
				// Clear the spinner line using display width for Unicode correctness
				clearLen := lipgloss.Width(s.frames[0]) + 1 + lipgloss.Width(s.message)
				fmt.Fprintf(s.w, "\r%s\r", strings.Repeat(" ", clearLen))
				return
			case <-ticker.C:
				fmt.Fprintf(s.w, "\r%s %s", s.frames[frame], s.message)
				frame = (frame + 1) % len(s.frames)
			}
		}
	}()
}

// Stop halts the spinner and clears its output.
func (s *spinner) Stop() {
	close(s.stop)
	<-s.done
}

// isTerminal returns true if the given file is a terminal.
func isTerminal(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

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

	// Show spinner while processing (only if stderr is a terminal)
	var spin *spinner
	if isTerminal(os.Stderr) {
		spin = newSpinner(os.Stderr, "Classifying diff...")
		spin.Start()
	}

	diff, classification, err := app.Run(ctx)

	// Stop spinner before TUI or error output
	if spin != nil {
		spin.Stop()
	}

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
